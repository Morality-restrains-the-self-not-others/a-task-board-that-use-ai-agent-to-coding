package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleGetUserMeSelfReference — /api/accounts/users/me/ 按凭据解析当前用户
// （Django 退役后修复的 404 回归，前端 Navbar 等 10+ 处依赖该端点）。
func TestHandleGetUserMeSelfReference(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("me-self@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/", nil)
	req.SetPathValue("user_id", "me")
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleGetUser(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /me/ expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var userPayload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &userPayload); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if userPayload["id"] != userID {
		t.Fatalf("me user_id mismatch: %v != %s", userPayload["id"], userID)
	}
	if userPayload["email"] != "me-self@test.com" {
		t.Fatalf("me email: %v", userPayload["email"])
	}
}

// TestHandleGetUserMeUnauthenticated — /me/ 无凭据时 401。
func TestHandleGetUserMeUnauthenticated(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/", nil)
	req.SetPathValue("user_id", "me")
	rec := httptest.NewRecorder()
	handleGetUser(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /me/ unauthenticated expected 401, got %d", rec.Code)
	}
}

// TestHandleGetUserMe401ClearsResidualCookies — 与 profile OPT-006 对齐：
// Navbar 优先打 /me/（OPT-20260810-015），清库后 /me/ 401 也必须清残留 HttpOnly cookie。
func TestHandleGetUserMe401ClearsResidualCookies(t *testing.T) {
	oldBase := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://www.daydaymoney.com"
	defer func() { cfg.GatewayPublicBase = oldBase }()

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/", nil)
	req.SetPathValue("user_id", "me")
	rec := httptest.NewRecorder()
	handleGetUser(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	cookies := rec.Result().Cookies()
	var gotUserID, gotToken bool
	for _, c := range cookies {
		if c.Name == "userId" && c.MaxAge < 0 {
			gotUserID = true
		}
		if c.Name == "token" && c.MaxAge < 0 {
			gotToken = true
		}
	}
	if !gotUserID || !gotToken {
		got := []string{}
		for _, c := range cookies {
			got = append(got, c.Name+"="+c.String())
		}
		t.Fatalf("GET /me/ 401 must clear userId/token cookies, got %v (raw: %v)",
			strings.Join(got, ", "), rec.Header()["Set-Cookie"])
	}
	sc := rec.Header().Values("Set-Cookie")
	for _, name := range []string{"userId", "token"} {
		var domainVariant, hostOnlyVariant bool
		for _, c := range sc {
			if !strings.Contains(c, name+"=") || !strings.Contains(c, "Max-Age=0") {
				continue
			}
			if strings.Contains(c, "Domain=") {
				domainVariant = true
			} else {
				hostOnlyVariant = true
			}
		}
		if !domainVariant || !hostOnlyVariant {
			t.Fatalf("GET /me/ 401 must clear BOTH domain+host-only variants for %s, got %v", name, sc)
		}
	}
}

// TestBuildUserDetailJSONIncludesPlatformRoles — /me/ 载荷包含 platform_roles。
func TestBuildUserDetailJSONIncludesPlatformRoles(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("plat-roles@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := ensureSuperAdminRow(userID); err != nil {
		t.Fatalf("ensureSuperAdminRow: %v", err)
	}
	payload, err := buildUserDetailJSON(userID)
	if err != nil {
		t.Fatalf("buildUserDetailJSON: %v", err)
	}
	roles, ok := payload["platform_roles"].([]string)
	if !ok {
		t.Fatalf("platform_roles missing or wrong type: %v", payload["platform_roles"])
	}
	found := false
	for _, r := range roles {
		if r == "super_admin" {
			found = true
		}
	}
	if !found {
		t.Fatalf("platform_roles should contain super_admin, got %v", roles)
	}
}

// TestLoadPlatformRolesFlagFallback — 无 RBAC 行时回退遗留标志位。
func TestLoadPlatformRolesFlagFallback(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("flag-fallback@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 仅设置遗留标志（模拟 028 回填后经遗留路径提升的用户），不写 RBAC 行
	if _, err := db.Exec(`UPDATE auth_user SET is_superuser = 1, is_staff = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("set flags: %v", err)
	}
	roles, err := loadPlatformRoles(userID)
	if err != nil {
		t.Fatalf("loadPlatformRoles: %v", err)
	}
	if len(roles) != 1 || roles[0] != "super_admin" {
		t.Fatalf("flag fallback expected [super_admin], got %v", roles)
	}

	// 普通用户 → 空列表
	userID2, _, err := createUserWithEmailLogin("flag-fallback2@test.com", "hash")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}
	roles2, err := loadPlatformRoles(userID2)
	if err != nil {
		t.Fatalf("loadPlatformRoles2: %v", err)
	}
	if len(roles2) != 0 {
		t.Fatalf("normal user expected no platform roles, got %v", roles2)
	}
}

// TestEnsureSuperAdminRowWritesRBAC — 遗留提升路径双写 RBAC 角色行。
func TestEnsureSuperAdminRowWritesRBAC(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("rbac-double-write@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := ensureSuperAdminRow(userID); err != nil {
		t.Fatalf("ensureSuperAdminRow: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_user_role
		WHERE user_id = ? AND role_id = 'role-super-admin' AND company_id IS NULL`, userID).Scan(&count); err != nil {
		t.Fatalf("count rbac: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 role-super-admin row, got %d", count)
	}
}

// TestHandleInternalUserPlatformRoles — 内部端点需密钥且返回平台角色。
func TestHandleInternalUserPlatformRoles(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"
	userID, _, err := createUserWithEmailLogin("internal-roles@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := ensureSuperAdminRow(userID); err != nil {
		t.Fatalf("ensureSuperAdminRow: %v", err)
	}

	// 无密钥 → 403
	req := httptest.NewRequest(http.MethodGet, "/api/internal/users/id/"+userID+"/platform-roles/", nil)
	req.SetPathValue("user_id", userID)
	rec := httptest.NewRecorder()
	handleInternalUserPlatformRoles(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("no secret expected 403, got %d", rec.Code)
	}

	// 有密钥 → 200 + super_admin
	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/users/id/"+userID+"/platform-roles/", nil)
	req2.SetPathValue("user_id", userID)
	req2.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec2 := httptest.NewRecorder()
	handleInternalUserPlatformRoles(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("with secret expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var body struct {
		Roles []string `json:"roles"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Roles) != 1 || body.Roles[0] != "super_admin" {
		t.Fatalf("expected [super_admin], got %v", body.Roles)
	}
}

// TestHandleMyUserRolesFlagFallback — /api/auth/user-roles/ 对仅标志位用户回退平台角色。
func TestHandleMyUserRolesFlagFallback(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("my-roles-fallback@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_staff = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("set staff flag: %v", err)
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/user-roles/", nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleMyUserRoles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Roles []map[string]any `json:"roles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	joined := ""
	for _, r := range body.Roles {
		joined += strings.TrimSpace(strings.TrimSpace(r["role"].(string))) + ","
	}
	if !strings.Contains(joined, "employee") {
		t.Fatalf("expected employee in roles, got %v", joined)
	}
}

// TestResolveLoginRedirectPlatformStaff — 平台员工登录后禁止落入 /onboarding/。
func TestResolveLoginRedirectPlatformStaff(t *testing.T) {
	cases := []struct {
		name     string
		user     map[string]interface{}
		expected string
	}{
		{
			name:     "superuser → system-admin",
			user:     map[string]interface{}{"is_superuser": true, "platform_roles": []string{"super_admin"}},
			expected: "/system-admin/",
		},
		{
			name:     "employee（无公司）→ system-admin，禁止 onboarding",
			user:     map[string]interface{}{"is_superuser": false, "platform_roles": []string{"employee"}, "companies": []interface{}{}},
			expected: "/system-admin/",
		},
		{
			name: "普通用户有公司 → work-panel",
			user: map[string]interface{}{"is_superuser": false, "platform_roles": []string{},
				"companies": []interface{}{map[string]interface{}{"id": "t1"}}},
			expected: "/tenant/t1/work-panel/",
		},
		{
			name:     "普通用户无公司 → onboarding（仅限非平台角色）",
			user:     map[string]interface{}{"is_superuser": false, "platform_roles": []string{}, "companies": []interface{}{}},
			expected: "/onboarding/",
		},
		{
			name:     "遗留标志回退：platform_roles 缺失但 is_superuser → system-admin",
			user:     map[string]interface{}{"is_superuser": true},
			expected: "/system-admin/",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveLoginRedirect(c.user); got != c.expected {
				t.Fatalf("resolveLoginRedirect = %q, want %q", got, c.expected)
			}
		})
	}
}

func TestImpersonationLandingRedirectAvoidsSystemAdmin(t *testing.T) {
	got := impersonationLandingRedirect(map[string]interface{}{
		"is_superuser":   true,
		"platform_roles": []string{"super_admin"},
		"companies":      []interface{}{},
	})
	if got == "" || strings.HasPrefix(got, "/system-admin") {
		t.Fatalf("impersonation landing must not be system-admin, got %q", got)
	}
	if got != "/onboarding/" {
		t.Fatalf("platform-admin target without company → onboarding, got %q", got)
	}
	withCo := impersonationLandingRedirect(map[string]interface{}{
		"is_superuser":   false,
		"platform_roles": []string{},
		"companies":      []interface{}{map[string]interface{}{"id": "t9"}},
	})
	if withCo != "/tenant/t9/work-panel/" {
		t.Fatalf("regular target → work-panel, got %q", withCo)
	}
}

// helper: 读取用户遗留标志
func readUserFlags(t *testing.T, userID string) (isSuper, isStaff bool) {
	t.Helper()
	if err := db.QueryRow(`SELECT is_superuser, is_staff FROM auth_user WHERE id = ?`, userID).Scan(&isSuper, &isStaff); err != nil {
		t.Fatalf("read flags: %v", err)
	}
	return
}

func readSuperAdminRow(t *testing.T, userID string) bool {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM auth_super_admin WHERE user_id = ?`, userID).Scan(&n)
	if err != nil {
		t.Fatalf("read super_admin row: %v", err)
	}
	return n > 0
}

func platformAuthReq(uid, role string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/user-roles/user_id/"+uid+"/", nil)
	req.SetPathValue("uid", uid)
	req.Header.Set("X-User-Id", uid)
	if role != "" {
		req.Header.Set("X-User-Roles", role)
	}
	return req
}

// TestAssignPlatformRoleSyncsFlags — RBAC 指派反向双写遗留标志（缺口修复）。
func TestAssignPlatformRoleSyncsFlags(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("assign-sync@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 1. 指派 employee → is_staff=1, is_superuser=0
	req := platformAuthReq(userID, "super_admin")
	req.Body = io.NopCloser(strings.NewReader(`{"role_name":"employee"}`))
	rec := httptest.NewRecorder()
	handleAssignPlatformRole(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("assign employee expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	isSuper, isStaff := readUserFlags(t, userID)
	if isSuper || !isStaff {
		t.Fatalf("employee assign: want is_superuser=false is_staff=true, got %v/%v", isSuper, isStaff)
	}
	if readSuperAdminRow(t, userID) {
		t.Fatal("employee assign must not create auth_super_admin row")
	}

	// 2. 提升为 super_admin → is_superuser=1, is_staff=1 + auth_super_admin 行
	req2 := platformAuthReq(userID, "super_admin")
	req2.Body = io.NopCloser(strings.NewReader(`{"role_name":"super_admin"}`))
	rec2 := httptest.NewRecorder()
	handleAssignPlatformRole(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("assign super_admin expected 200, got %d", rec2.Code)
	}
	isSuper, isStaff = readUserFlags(t, userID)
	if !isSuper || !isStaff {
		t.Fatalf("super_admin assign: want is_superuser=true is_staff=true, got %v/%v", isSuper, isStaff)
	}
	if !readSuperAdminRow(t, userID) {
		t.Fatal("super_admin assign must create auth_super_admin row")
	}
}

// TestRevokePlatformRoleClearsFlags — 撤销平台角色后标志随 RBAC 重算清零。
func TestRevokePlatformRoleClearsFlags(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("revoke-sync@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := ensureSuperAdminRow(userID); err != nil {
		t.Fatalf("ensureSuperAdminRow: %v", err)
	}
	if isSuper, isStaff := readUserFlags(t, userID); !isSuper || !isStaff {
		t.Fatalf("precondition: want super+staff, got %v/%v", isSuper, isStaff)
	}

	// 撤销 super_admin → 全部清零
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/user-roles/user_id/"+userID+"/role_name/super_admin/", nil)
	req.SetPathValue("uid", userID)
	req.SetPathValue("name", "super_admin")
	req.Header.Set("X-User-Id", userID)
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleRevokePlatformRole(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	isSuper, isStaff := readUserFlags(t, userID)
	if isSuper || isStaff {
		t.Fatalf("revoke: want is_superuser=false is_staff=false, got %v/%v", isSuper, isStaff)
	}
	if readSuperAdminRow(t, userID) {
		t.Fatal("revoke must remove auth_super_admin row")
	}
}

// TestGatewayForwardAuthRBACRoleWithoutFlags — 缺口回归测试：
// 仅 RBAC 指派（无遗留标志）的用户，网关 forward-auth 的 X-User-Roles 必须包含
// 平台角色（此前仅按标志推导 → 缺失）。
func TestGatewayForwardAuthRBACRoleWithoutFlags(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	// 仅写 RBAC 行，不设置任何遗留标志（模拟"反向双写缺口"存量数据）
	if _, err := db.Exec(`INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
		VALUES (?, ?, 'role-employee', NULL, 'system', NOW())`, "aur-test-"+userID, userID); err != nil {
		t.Fatalf("insert rbac row: %v", err)
	}
	if isSuper, isStaff := readUserFlags(t, userID); isSuper || isStaff {
		t.Fatalf("precondition: flags must be unset, got %v/%v", isSuper, isStaff)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	roles := rec.Header().Get("X-User-Roles")
	if !strings.Contains(roles, "employee") {
		t.Fatalf("X-User-Roles must contain employee from RBAC rows (gap fix), got %q", roles)
	}
	// 遗留 X-Auth-Superuser/X-Auth-Staff 头已移除：平台判定统一走 X-User-Roles
	if rec.Header().Get("X-Auth-Staff") != "" || rec.Header().Get("X-Auth-Superuser") != "" {
		t.Fatalf("legacy headers must not be injected anymore, got Staff=%q Superuser=%q",
			rec.Header().Get("X-Auth-Staff"), rec.Header().Get("X-Auth-Superuser"))
	}
}

// TestHandleMyUserRolesCollapsesDuplicatePlatformRows — MySQL UNIQUE(user_id,role_id,company_id)
// 对 company_id NULL 不生效，ensurePlatformRoleRows 每次 INSERT IGNORE 新主键都会再插一行。
// GET /api/auth/user-roles/ 必须折叠重复项（生产曾返回 21 条相同 super_admin）。
func TestHandleMyUserRolesCollapsesDuplicatePlatformRows(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("dup-roles@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	inserted := 0
	for i := 0; i < 5; i++ {
		_, err := db.Exec(`INSERT INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
			VALUES (?, ?, 'role-super-admin', NULL, 'system', NOW())`,
			fmt.Sprintf("aur-dup-%s-%d", userID, i), userID)
		if err == nil {
			inserted++
		}
	}
	if inserted == 0 {
		t.Fatal("expected at least one platform role row")
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/user-roles/", nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleMyUserRoles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Roles []map[string]any `json:"roles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	n := 0
	for _, r := range body.Roles {
		if r["role"] == "super_admin" && r["level"] == "platform" && r["company_id"] == "" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected 1 unique super_admin (raw inserts=%d), got %d in %v", inserted, n, body.Roles)
	}
}

// TestUniqueUserRoleItems — 纯函数折叠 (role, level, company_id) 重复项。
func TestUniqueUserRoleItems(t *testing.T) {
	in := []map[string]any{
		{"role": "super_admin", "level": "platform", "company_id": ""},
		{"role": "super_admin", "level": "platform", "company_id": ""},
		{"role": "member", "level": "tenant", "company_id": "c1"},
		{"role": "member", "level": "tenant", "company_id": "c1"},
		{"role": "member", "level": "tenant", "company_id": "c2"},
	}
	out := uniqueUserRoleItems(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 unique items, got %d %+v", len(out), out)
	}
}

// TestEnsurePlatformRoleRowsIdempotent — 重复调用不得因 NULL company_id 绕过 UNIQUE 而堆行。
func TestEnsurePlatformRoleRowsIdempotent(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("ensure-idempotent@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ensurePlatformRoleRows(userID, "role-super-admin")
	ensurePlatformRoleRows(userID, "role-super-admin")
	ensurePlatformRoleRows(userID, "role-super-admin")
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_user_role
		WHERE user_id = ? AND role_id = 'role-super-admin' AND company_id IS NULL`, userID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("ensurePlatformRoleRows called 3x: expected 1 row, got %d", count)
	}
}
