package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setupTestMux creates a test HTTP mux with routes mounted.
// Uses the existing setupTestReferralDB for database setup.
func setupTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	setupTestReferralDB(t)

	// Create auth tables in test DB for superuser checks
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS auth_user (
		id VARCHAR(255) NOT NULL PRIMARY KEY,
		is_superuser TINYINT NOT NULL DEFAULT 0
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create auth_user: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS auth_super_admin (
		user_id VARCHAR(255) NOT NULL PRIMARY KEY
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create auth_super_admin: %v", err)
	}

	// Point authDB at the same test DB for tests
	authDB = db

	// Seed superadmin for admin endpoint tests
	_, _ = db.Exec(`DELETE FROM auth_super_admin`)
	_, _ = db.Exec(`DELETE FROM auth_user`)
	_, _ = db.Exec(`INSERT INTO auth_user (id, is_superuser) VALUES ('super1', 0)`)
	_, _ = db.Exec(`INSERT INTO auth_super_admin (user_id) VALUES ('super1')`)

	mux := http.NewServeMux()
	mountRoutes(mux)
	return mux
}

func withReferralUser(req *http.Request, userID string) *http.Request {
	req.Header.Set("X-User-Id", userID)
	return req
}

func newReferralApplyRequest(userID, intro string) *http.Request {
	body, _ := json.Marshal(map[string]interface{}{
		"personal_intro":        intro,
		"legal_name":            testValidLegalName,
		"identity_bind_consent": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/referral-codes/apply/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return withReferralUser(req, userID)
}

// OPT-20260819-007: internal_error 时响应 body 不得泄露 SQL/表字段原文。
func TestHandleReferralCodeApplyInternalErrorHidesSQLDetail(t *testing.T) {
	mux := setupTestMux(t)

	// 强制 DB 内部错误：drop referral_policy 使 getReferralPolicy 失败 → internal_error。
	if _, err := db.Exec(`DROP TABLE IF EXISTS referral_policy`); err != nil {
		t.Fatalf("drop referral_policy: %v", err)
	}

	req := newReferralApplyRequest("user-err-1", testValidPersonalIntro)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "HY000") || strings.Contains(body, "referral_policy") {
		t.Fatalf("response leaked SQL detail: %s", body)
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed["error"] != "internal_error" {
		t.Fatalf("error=%q", parsed["error"])
	}
	if !strings.Contains(parsed["detail"], "申请失败") {
		t.Fatalf("detail=%q, want generic message", parsed["detail"])
	}
}

// ── Health endpoint tests ──

func TestHealthEndpoint(t *testing.T) {
	mux := setupTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["status"] != "ok" || out["service"] != "taskReferral" {
		t.Fatalf("unexpected body: %v", out)
	}
}

// ── Referral Code Status tests ──

func TestReferralCodeStatusUnauthenticated(t *testing.T) {
	mux := setupTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/referral-codes/status/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestReferralCodeStatusMethodNotAllowed(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodPost, "/api/accounts/users/referral-codes/status/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestReferralCodeStatusNoApplication(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/accounts/users/referral-codes/status/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["can_apply"] != true {
		t.Fatalf("expected can_apply=true, got %v", out)
	}
	if out["has_active_code"] == true {
		t.Fatalf("expected has_active_code=false, got %v", out)
	}
	code, _ := out["access_code"].(string)
	assertOpaqueShareCode(t, code, "user1")
}

// ── Referral Code Apply tests ──

func TestReferralApplyUnauthenticated(t *testing.T) {
	mux := setupTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/referral-codes/apply/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestReferralApplyMethodNotAllowed(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/accounts/users/referral-codes/apply/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestReferralApplyApprovalMode(t *testing.T) {
	mux := setupTestMux(t)
	setReferralPolicyForTest(t, "approval", "")
	req := newReferralApplyRequest("user1", testValidPersonalIntro)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["status"] != "pending" {
		t.Fatalf("expected status=pending, got %v", out)
	}
}

func TestReferralApplyMissingIntro(t *testing.T) {
	mux := setupTestMux(t)
	setReferralPolicyForTest(t, "approval", "")
	req := newReferralApplyRequest("user-no-intro", "")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["error"] != "invalid_intro" {
		t.Fatalf("expected invalid_intro, got %v", out)
	}
}

func TestReferralApplyDuplicatePending(t *testing.T) {
	mux := setupTestMux(t)
	setReferralPolicyForTest(t, "approval", "")
	// First apply
	req1 := newReferralApplyRequest("dup1", testValidPersonalIntro)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)
	if rec1.Code != 200 {
		t.Fatalf("first apply: expected 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}
	// Duplicate apply
	req2 := newReferralApplyRequest("dup1", testValidPersonalIntro)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 400 {
		t.Fatalf("expected 400 for duplicate, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out)
	if out["error"] != "already_pending" {
		t.Fatalf("expected error=already_pending, got %v", out)
	}
}

// ── Admin Applications List tests ──

func TestAdminApplicationsUnauthenticated(t *testing.T) {
	mux := setupTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/applications/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminApplicationsNonSuperuser(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/applications/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("expected 403 for non-superuser, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminApplicationsSuperuser(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/applications/", nil), "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["items"] == nil {
		t.Fatal("expected items in response")
	}
	if out["total"] == nil {
		t.Fatal("expected total in response")
	}
}

func TestAdminApplicationsWithStatusFilter(t *testing.T) {
	mux := setupTestMux(t)
	// Create a pending application
	setReferralPolicyForTest(t, "approval", "")
	applyReq := newReferralApplyRequest("filter1", testValidPersonalIntro)
	applyRec := httptest.NewRecorder()
	mux.ServeHTTP(applyRec, applyReq)

	// List with status filter
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/applications/?status=pending&limit=10&offset=0", nil), "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	items, _ := out["items"].([]interface{})
	if len(items) < 1 {
		t.Fatalf("expected at least 1 pending application, got %v", out)
	}
}

// ── Admin Approve tests ──

func TestAdminApproveInvalidID(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodPost, "/api/system-admin/referral/applications/notanumber/approve/", nil), "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for invalid ID, got %d", rec.Code)
	}
}

func TestAdminApproveNonSuperuser(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodPost, "/api/system-admin/referral/applications/1/approve/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// ── Admin Reject tests ──

func TestAdminRejectWithReason(t *testing.T) {
	mux := setupTestMux(t)
	// Create pending application
	setReferralPolicyForTest(t, "approval", "")
	applyReq := newReferralApplyRequest("reject1", testValidPersonalIntro)
	applyRec := httptest.NewRecorder()
	mux.ServeHTTP(applyRec, applyReq)

	// List to find the application ID
	listReq := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/applications/?status=pending", nil), "super1")
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	var listOut map[string]interface{}
	_ = json.Unmarshal(listRec.Body.Bytes(), &listOut)
	items, _ := listOut["items"].([]interface{})
	if len(items) == 0 {
		t.Fatal("no pending applications to reject")
	}
	app, _ := items[0].(map[string]interface{})
	appID := int(app["id"].(float64))

	// Reject with reason
	body := `{"reason":"Invalid referral"}`
	rejectReq := withReferralUser(httptest.NewRequest(http.MethodPost,
		"/api/system-admin/referral/applications/"+itoa(appID)+"/reject/",
		bytes.NewBufferString(body)), "super1")
	rejectRec := httptest.NewRecorder()
	mux.ServeHTTP(rejectRec, rejectReq)
	if rejectRec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rejectRec.Code, rejectRec.Body.String())
	}
}

// ── Admin Policy tests ──

func TestAdminPolicyGetSuperuser(t *testing.T) {
	mux := setupTestMux(t)
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/policy/", nil), "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["data"] == nil {
		t.Fatal("expected data in response")
	}
}

func TestAdminPolicyUpdateMode(t *testing.T) {
	mux := setupTestMux(t)
	body := `{"mode":"open","message":"Referrals are open!"}`
	req := withReferralUser(httptest.NewRequest(http.MethodPut, "/api/system-admin/referral/policy/", bytes.NewBufferString(body)), "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	data, _ := out["data"].(map[string]interface{})
	if data["mode"] != "open" {
		t.Fatalf("expected mode=open, got %v", data)
	}
}

func TestAdminPolicyUpdateInvalidMode(t *testing.T) {
	mux := setupTestMux(t)
	body := `{"mode":"invalid"}`
	req := withReferralUser(httptest.NewRequest(http.MethodPut, "/api/system-admin/referral/policy/", bytes.NewBufferString(body)), "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for invalid mode, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminPolicyUnauthenticated(t *testing.T) {
	mux := setupTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/policy/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAdminReferralConfigSavesRatios(t *testing.T) {
	mux := setupTestMux(t)
	ensureBillingReferralConfigTable(t)

	body, _ := json.Marshal(map[string]interface{}{
		"referral_rate_percent": 18,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/referral/config/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			ProfitSharingRatioPercent int    `json:"profit_sharing_ratio_percent"`
			ReferralRatePercent       int    `json:"referral_rate_percent"`
			ProfitSharingRatioDisplay string `json:"profit_sharing_ratio_display"`
			ReferralRateDisplay       string `json:"referral_rate_display"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if out.Data.ProfitSharingRatioPercent != 5 || out.Data.ReferralRatePercent != 5 {
		t.Fatalf("data=%+v want fixed 5", out.Data)
	}
	if out.Data.ProfitSharingRatioDisplay != "5%" || out.Data.ReferralRateDisplay != "5%" {
		t.Fatalf("display=%s/%s want 5%%", out.Data.ProfitSharingRatioDisplay, out.Data.ReferralRateDisplay)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}
