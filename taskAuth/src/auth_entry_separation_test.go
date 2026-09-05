package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════
// 入口分离（OPT-20260824-001）：管理员登录入口 /api/auth/admin-login/
// 与客户登录入口 /api/auth/ 双向隔离 —— 管理员不得走客户入口，
// 客户不得走管理员入口。所有断言先验证凭据（密码/验证码），
// 角色判定仅在凭据通过后进行。
// ═══════════════════════════════════════════════════════════════════

// postAdminLogin posts email+password to the admin-only login entry.
func postAdminLogin(t *testing.T, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/admin-login/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", "admin-login-entry-separation")
	rec := httptest.NewRecorder()
	handleAdminLogin(rec, req)
	return rec
}

// postCustomerLogin posts email+password to the customer login entry.
func postCustomerLogin(t *testing.T, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", "customer-login-entry-separation")
	rec := httptest.NewRecorder()
	handleLogin(rec, req)
	return rec
}

// createStaffUser creates an active email user flagged as platform staff (employee).
func createStaffUser(t *testing.T, email, password string) string {
	t.Helper()
	userID, _, err := createUserWithEmailLogin(email, password)
	if err != nil {
		t.Fatalf("create staff user: %v", err)
	}
	if err := activateLoginMethodByEmail(email); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_staff = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("set is_staff: %v", err)
	}
	return userID
}

// createSuperAdminRoleUser creates an active email user with the super_admin
// platform role but no legacy flags (covers the platform-role admin path).
func createSuperAdminRoleUser(t *testing.T, email, password string) string {
	t.Helper()
	userID, _, err := createUserWithEmailLogin(email, password)
	if err != nil {
		t.Fatalf("create role user: %v", err)
	}
	if err := activateLoginMethodByEmail(email); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
		VALUES (?, ?, 'role-super-admin', NULL, 'system', NOW())`, "t-role-"+userID, userID); err != nil {
		t.Fatalf("assign super_admin role: %v", err)
	}
	return userID
}

// createCustomerUser creates an active regular (non-admin) email user.
func createCustomerUser(t *testing.T, email, password string) string {
	t.Helper()
	userID, _, err := createUserWithEmailLogin(email, password)
	if err != nil {
		t.Fatalf("create customer user: %v", err)
	}
	if err := activateLoginMethodByEmail(email); err != nil {
		t.Fatalf("activate: %v", err)
	}
	return userID
}

func TestAdminLoginEndpointAllowsAdminStaff(t *testing.T) {
	setupAuthTestDB(t)
	createStaffUser(t, "admin-staff@test.com", "AdminPassw0rd!")

	rec := postAdminLogin(t, "admin-staff@test.com", "AdminPassw0rd!")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin staff login expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatalf("missing token: %v", resp)
	}
}

func TestAdminLoginEndpointAllowsSuperAdminRole(t *testing.T) {
	setupAuthTestDB(t)
	createSuperAdminRoleUser(t, "admin-role@test.com", "AdminPassw0rd!")

	rec := postAdminLogin(t, "admin-role@test.com", "AdminPassw0rd!")
	if rec.Code != http.StatusOK {
		t.Fatalf("super_admin role login expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestAdminLoginEndpointRejectsCustomer(t *testing.T) {
	setupAuthTestDB(t)
	createCustomerUser(t, "customer@test.com", "CustomerPassw0rd!")

	rec := postAdminLogin(t, "customer@test.com", "CustomerPassw0rd!")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("customer at admin entry expected 403, got %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if resp["error"] != "not_admin_account" {
		t.Fatalf("expected error=not_admin_account, got %v", resp)
	}
	// 客户账号在管理员入口不得签发 token
	if resp["token"] != nil {
		t.Fatalf("customer must not receive token: %v", resp)
	}
}

func TestAdminLoginEndpointWrongPasswordSameMessage(t *testing.T) {
	setupAuthTestDB(t)
	createStaffUser(t, "admin-wrongpw@test.com", "AdminPassw0rd!")

	rec := postAdminLogin(t, "admin-wrongpw@test.com", "WrongPassw0rd!")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("wrong password expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	// 密码错误与账号不存在必须同口径（不暴露管理员身份/角色信息）
	if !strings.Contains(rec.Body.String(), "邮箱或密码错误") {
		t.Fatalf("expected generic password message, got %s", rec.Body.String())
	}
}

func TestCustomerLoginRejectsAdminStaff(t *testing.T) {
	setupAuthTestDB(t)
	createStaffUser(t, "admin-at-customer@test.com", "AdminPassw0rd!")

	rec := postCustomerLogin(t, "admin-at-customer@test.com", "AdminPassw0rd!")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin at customer entry expected 403, got %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if resp["error"] != "admin_requires_admin_login" {
		t.Fatalf("expected error=admin_requires_admin_login, got %v", resp)
	}
	if resp["token"] != nil {
		t.Fatalf("admin must not receive customer-entry token: %v", resp)
	}
}

func TestCustomerPhoneCodeLoginRejected(t *testing.T) {
	// 2026-08-24：手机号+验证码登录已整体移除（含自动注册）。携带 code 且无
	// password 的请求一律 400 fail-closed，与账号角色无关。
	setupAuthTestDB(t)
	const national = "13900003333"
	userID, err := createUserWithPhoneLogin("+86", national, "whatever", "")
	if err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_staff = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("set is_staff: %v", err)
	}
	// 直接种一条未使用的验证码（绕过 SMS 发送，仅测登录判定路径）
	if _, err := db.Exec(`
		INSERT INTO auth_sms_verification_code (phone, country_calling_code, user_id, code, created_at, expires_at, is_used)
		VALUES (?, '+86', ?, '123456', NOW(), DATE_ADD(NOW(), INTERVAL 10 MINUTE), 0)`, national, userID); err != nil {
		t.Fatalf("seed sms code: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"phone": "+86" + national, "code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("phone+code login expected 400 (disabled), got %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if resp["error"] != "phone_code_login_disabled" {
		t.Fatalf("expected error=phone_code_login_disabled, got %v", resp)
	}
	if resp["token"] != nil {
		t.Fatalf("disabled login must not issue token: %v", resp)
	}
}

func TestAccessTokenLoginRejectsAdminStaff(t *testing.T) {
	setupAuthTestDB(t)
	email := "admin-token@test.com"
	userID := createStaffUser(t, email, "AdminPassw0rd!")

	_, rawToken, err := createAccessToken(userID, "chrome-plugin", "")
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{"username": email, "access_token": rawToken})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/login-with-access-token/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLoginWithAccessToken(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin access-token at customer entry expected 403, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestCustomerLoginAllowsRegularUser(t *testing.T) {
	setupAuthTestDB(t)
	createCustomerUser(t, "plain-customer@test.com", "CustomerPassw0rd!")

	rec := postCustomerLogin(t, "plain-customer@test.com", "CustomerPassw0rd!")
	if rec.Code != http.StatusOK {
		t.Fatalf("regular customer login expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatalf("missing token: %v", resp)
	}
}

func TestIsPlatformAdminStaff(t *testing.T) {
	setupAuthTestDB(t)
	staffID := createStaffUser(t, "flag-staff@test.com", "Passw0rd!")
	roleID := createSuperAdminRoleUser(t, "role-admin@test.com", "Passw0rd!")
	customerID := createCustomerUser(t, "flag-customer@test.com", "Passw0rd!")

	for _, tc := range []struct {
		userID string
		want   bool
		label  string
	}{
		{staffID, true, "is_staff flag"},
		{roleID, true, "super_admin platform role"},
		{customerID, false, "regular customer"},
	} {
		got, err := isPlatformAdminStaff(tc.userID)
		if err != nil {
			t.Fatalf("%s: %v", tc.label, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got=%v want=%v", tc.label, got, tc.want)
		}
	}
	// 未知用户不 panic，返回 false
	got, err := isPlatformAdminStaff(fmt.Sprintf("%d", generateSnowflakeID()))
	if err != nil {
		t.Fatalf("unknown user: %v", err)
	}
	if got {
		t.Fatal("unknown user must not be admin staff")
	}
}
