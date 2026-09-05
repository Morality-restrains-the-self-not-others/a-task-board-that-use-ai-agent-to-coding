package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskAuth/domain"
)

func countLoginHistory(t *testing.T, userID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_login_history WHERE user_id = ?`, userID).Scan(&n); err != nil {
		t.Fatalf("count login history: %v", err)
	}
	return n
}

func countLoginHistoryOutcome(t *testing.T, userID, outcome string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_login_history WHERE user_id = ? AND outcome = ?`, userID, outcome).Scan(&n); err != nil {
		t.Fatalf("count login history outcome=%s: %v", outcome, err)
	}
	return n
}

func TestCustomerLoginRecordsHistoryIP(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-cust@test.com", "CustPassw0rd!")
	body, _ := json.Marshal(map[string]string{"email": "hist-cust@test.com", "password": "CustPassw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	req.Header.Set("User-Agent", "LoginHistoryTest/1.0")
	rec := httptest.NewRecorder()
	handleLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %d %s", rec.Code, rec.Body.String())
	}
	if n := countLoginHistory(t, uid); n != 1 {
		t.Fatalf("want 1 row, got %d", n)
	}
	var entry, ip, ua, method string
	if err := db.QueryRow(`
		SELECT entry, client_ip, user_agent, method_type FROM auth_login_history WHERE user_id = ?`, uid).
		Scan(&entry, &ip, &ua, &method); err != nil {
		t.Fatal(err)
	}
	if entry != domain.LoginEntryCustomer {
		t.Fatalf("entry=%q", entry)
	}
	if ip != "203.0.113.9" {
		t.Fatalf("ip=%q", ip)
	}
	if ua != "LoginHistoryTest/1.0" {
		t.Fatalf("ua=%q", ua)
	}
	if method != "email" {
		t.Fatalf("method=%q", method)
	}
}

func TestCustomerLoginRecordsPublicIPWhenXFFStartsWithDockerNAT(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-nat@test.com", "CustPassw0rd!")
	body, _ := json.Marshal(map[string]string{"email": "hist-nat@test.com", "password": "CustPassw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "172.26.0.1, 203.0.113.9")
	req.Header.Set("X-Real-IP", "172.26.0.1")
	req.RemoteAddr = "172.26.0.1:4000"
	rec := httptest.NewRecorder()
	handleLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %d %s", rec.Code, rec.Body.String())
	}
	var ip string
	if err := db.QueryRow(`SELECT client_ip FROM auth_login_history WHERE user_id = ?`, uid).Scan(&ip); err != nil {
		t.Fatal(err)
	}
	if ip != "203.0.113.9" {
		t.Fatalf("ip=%q want public hop", ip)
	}
}

func TestAdminLoginRecordsHistoryEntry(t *testing.T) {
	setupAuthTestDB(t)
	uid := createStaffUser(t, "hist-admin@test.com", "AdminPassw0rd!")
	rec := postAdminLogin(t, "hist-admin@test.com", "AdminPassw0rd!")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login %d %s", rec.Code, rec.Body.String())
	}
	var entry string
	if err := db.QueryRow(`SELECT entry FROM auth_login_history WHERE user_id = ?`, uid).Scan(&entry); err != nil {
		t.Fatal(err)
	}
	if entry != domain.LoginEntryAdmin {
		t.Fatalf("entry=%q", entry)
	}
}

func TestFailedLoginDoesNotRecordSuccessHistory(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-fail@test.com", "CustPassw0rd!")
	rec := postCustomerLogin(t, "hist-fail@test.com", "WrongPassw0rd!")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
	// 失败登录不得写入成功历史行（默认列表仍只展示成功）。
	if n := countLoginHistoryOutcome(t, uid, domain.LoginOutcomeSuccess); n != 0 {
		t.Fatalf("failed login must not write success history, got %d", n)
	}
	// OPT-20260825-035：失败尝试作为 outcome=password_mismatch 落一条留痕。
	if n := countLoginHistoryOutcome(t, uid, domain.LoginOutcomePasswordMismatch); n != 1 {
		t.Fatalf("failed login must record one password_mismatch attempt, got %d", n)
	}
}

func TestFailedLoginDoesNotStoreIdentifier(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-noident@test.com", "CustPassw0rd!")
	if rec := postCustomerLogin(t, "hist-noident@test.com", "WrongPassw0rd!"); rec.Code != http.StatusBadRequest {
		t.Fatalf("login %d", rec.Code)
	}
	// 失败尝试只按 user_id 留痕；email/手机/用户名等 identifier 无独立列（结构上不落库）。
	if n := countLoginHistoryOutcome(t, uid, domain.LoginOutcomePasswordMismatch); n != 1 {
		t.Fatalf("want 1 attempt row, got %d", n)
	}
}

func TestEmailUnverifiedLoginRecordsAttempt(t *testing.T) {
	setupAuthTestDB(t)
	uid, _, err := createUserWithEmailLogin("hist-unv@test.com", "CustPassw0rd!")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rec := postCustomerLogin(t, "hist-unv@test.com", "CustPassw0rd!")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
	if n := countLoginHistoryOutcome(t, uid, domain.LoginOutcomeEmailUnverified); n != 1 {
		t.Fatalf("email unverified must record attempt, got %d", n)
	}
}

func TestInactiveUserLoginRecordsNotActiveAttempt(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-inactive@test.com", "CustPassw0rd!")
	if _, err := db.Exec(`UPDATE auth_user SET is_active = 0 WHERE id = ?`, uid); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	rec := postCustomerLogin(t, "hist-inactive@test.com", "CustPassw0rd!")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
	if n := countLoginHistoryOutcome(t, uid, domain.LoginOutcomeNotActive); n != 1 {
		t.Fatalf("inactive login must record not_active attempt, got %d", n)
	}
}

func TestAdminEntryMismatchLoginRecordsAttempt(t *testing.T) {
	setupAuthTestDB(t)
	uid := createStaffUser(t, "hist-staff@test.com", "StaffPassw0rd!")
	// 管理员账号走客户入口：凭据正确但入口不匹配 → admin_entry_mismatch。
	rec := postCustomerLogin(t, "hist-staff@test.com", "StaffPassw0rd!")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", rec.Code, rec.Body.String())
	}
	if n := countLoginHistoryOutcome(t, uid, domain.LoginOutcomeAdminEntryMismatch); n != 1 {
		t.Fatalf("admin-entry mismatch must record attempt, got %d", n)
	}
}

func TestListOwnLoginHistoryDefaultExcludesFailures(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-mix@test.com", "CustPassw0rd!")
	// 先失败再成功：库里应有 1 条 success + 1 条 password_mismatch。
	if rec := postCustomerLogin(t, "hist-mix@test.com", "WrongPassw0rd!"); rec.Code != http.StatusBadRequest {
		t.Fatalf("fail login %d", rec.Code)
	}
	if rec := postCustomerLogin(t, "hist-mix@test.com", "CustPassw0rd!"); rec.Code != http.StatusOK {
		t.Fatalf("ok login %d", rec.Code)
	}
	if total := countLoginHistory(t, uid); total != 2 {
		t.Fatalf("expect 2 rows (1 success + 1 failure), got %d", total)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-history/", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleListOwnLoginHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Results []map[string]interface{} `json:"results"`
		Total   float64                  `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if int(body.Total) != 1 || len(body.Results) != 1 {
		t.Fatalf("default list must exclude failures: total=%v results=%d body=%s", body.Total, len(body.Results), rec.Body.String())
	}
	if body.Results[0]["outcome"] != domain.LoginOutcomeSuccess {
		t.Fatalf("default result outcome=%v", body.Results[0]["outcome"])
	}
}

func TestListOwnLoginHistoryIncludeFailures(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-mix2@test.com", "CustPassw0rd!")
	if rec := postCustomerLogin(t, "hist-mix2@test.com", "WrongPassw0rd!"); rec.Code != http.StatusBadRequest {
		t.Fatalf("fail login %d", rec.Code)
	}
	if rec := postCustomerLogin(t, "hist-mix2@test.com", "CustPassw0rd!"); rec.Code != http.StatusOK {
		t.Fatalf("ok login %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-history/?include_failures=1", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleListOwnLoginHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Results []map[string]interface{} `json:"results"`
		Total   float64                  `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if int(body.Total) != 2 || len(body.Results) != 2 {
		t.Fatalf("include_failures must return success + failure: total=%v results=%d body=%s", body.Total, len(body.Results), rec.Body.String())
	}
	outcomes := map[string]bool{}
	for _, row := range body.Results {
		outcomes[row["outcome"].(string)] = true
	}
	if !outcomes[domain.LoginOutcomeSuccess] || !outcomes[domain.LoginOutcomePasswordMismatch] {
		t.Fatalf("outcomes=%v want success+password_mismatch", outcomes)
	}
}

func TestListOwnLoginHistoryScopedAndLabeled(t *testing.T) {
	setupAuthTestDB(t)
	a := createCustomerUser(t, "hist-a@test.com", "CustPassw0rd!")
	b := createCustomerUser(t, "hist-b@test.com", "CustPassw0rd!")
	if rec := postCustomerLogin(t, "hist-a@test.com", "CustPassw0rd!"); rec.Code != http.StatusOK {
		t.Fatalf("a login %d", rec.Code)
	}
	if rec := postCustomerLogin(t, "hist-b@test.com", "CustPassw0rd!"); rec.Code != http.StatusOK {
		t.Fatalf("b login %d", rec.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-history/", nil)
	req.Header.Set("X-User-Id", a)
	rec := httptest.NewRecorder()
	handleListOwnLoginHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Results []map[string]interface{} `json:"results"`
		Total   float64                  `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if int(body.Total) != 1 || len(body.Results) != 1 {
		t.Fatalf("A must see only own row: total=%v results=%d body=%s", body.Total, len(body.Results), rec.Body.String())
	}
	if body.Results[0]["entry_label"] != "用户入口" {
		t.Fatalf("entry_label=%v", body.Results[0]["entry_label"])
	}
	_ = b
}

func TestListOwnLoginHistoryUnauthorized(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-history/", nil)
	rec := httptest.NewRecorder()
	handleListOwnLoginHistory(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
}

func TestAdminLoginHistoryRequiresSuperuserAnd404(t *testing.T) {
	setupAuthTestDB(t)
	target := createCustomerUser(t, "hist-tgt@test.com", "CustPassw0rd!")
	if rec := postCustomerLogin(t, "hist-tgt@test.com", "CustPassw0rd!"); rec.Code != http.StatusOK {
		t.Fatalf("login %d", rec.Code)
	}

	forbidden := httptest.NewRequest(http.MethodGet, "/api/system-admin/users/"+target+"/login-history/", nil)
	forbidden.Header.Set("X-User-Id", target)
	fRec := httptest.NewRecorder()
	handleSystemAdminUsers(fRec, forbidden)
	if fRec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", fRec.Code, fRec.Body.String())
	}

	okReq := httptest.NewRequest(http.MethodGet, "/api/system-admin/users/"+target+"/login-history/?limit=20", nil)
	okReq.Header.Set("X-User-Id", "bootstrap-admin")
	okRec := httptest.NewRecorder()
	handleSystemAdminUsers(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("admin list %d %s", okRec.Code, okRec.Body.String())
	}
	var body struct {
		Results []map[string]interface{} `json:"results"`
		Total   float64                  `json:"total"`
		Limit   float64                  `json:"limit"`
	}
	if err := json.Unmarshal(okRec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if int(body.Total) != 1 {
		t.Fatalf("total=%v", body.Total)
	}

	missing := httptest.NewRequest(http.MethodGet, "/api/system-admin/users/999999999999999999/login-history/", nil)
	missing.Header.Set("X-User-Id", "bootstrap-admin")
	mRec := httptest.NewRecorder()
	handleSystemAdminUsers(mRec, missing)
	if mRec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d %s", mRec.Code, mRec.Body.String())
	}
}

func TestLoginHistoryLimitCappedAt100(t *testing.T) {
	setupAuthTestDB(t)
	uid := createCustomerUser(t, "hist-lim@test.com", "CustPassw0rd!")
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-history/?limit=500", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleListOwnLoginHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["limit"] != float64(100) {
		t.Fatalf("limit=%v", body["limit"])
	}
}

func TestImpersonationDoesNotRecordTargetLoginHistory(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("hist-imp@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := activateLoginMethodByEmail("hist-imp@test.com"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	before := countLoginHistory(t, targetID)
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", tok, targetID, "hist-imp-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("impersonate %d %s", rec.Code, rec.Body.String())
	}
	if after := countLoginHistory(t, targetID); after != before {
		t.Fatalf("impersonation must not write target history: before=%d after=%d", before, after)
	}
}
