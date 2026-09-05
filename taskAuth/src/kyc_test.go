package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKycDefaultT0RechargeGateAllowed(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userID, _, err := createUserWithEmailLogin("kyc-t0@test.com", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/kyc/profile/?user_id="+userID, nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalKycProfile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile: %d %s", rec.Code, rec.Body.String())
	}
	var profile kycProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Tier != kycTierT0 || profile.Status != kycStatusNone {
		t.Fatalf("want T0/none, got %s/%s", profile.Tier, profile.Status)
	}

	// T0 now allows recharge with conservative limits (no phone verification required).
	gateReq := httptest.NewRequest(http.MethodGet, "/api/internal/kyc/recharge-gate/?user_id="+userID+"&amount_yuan=100", nil)
	gateReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	gateRec := httptest.NewRecorder()
	handleInternalKycRechargeGate(gateRec, gateReq)
	if gateRec.Code != http.StatusOK {
		t.Fatalf("gate: %d %s", gateRec.Code, gateRec.Body.String())
	}
	var gate kycRechargeGateResult
	if err := json.Unmarshal(gateRec.Body.Bytes(), &gate); err != nil {
		t.Fatalf("decode gate: %v", err)
	}
	if !gate.Allowed {
		t.Fatalf("T0 should allow recharge, got allowed=%v reason=%s", gate.Allowed, gate.ReasonCode)
	}
	if gate.ReasonCode != reasonKycOK {
		t.Fatalf("reason want OK, got %q", gate.ReasonCode)
	}
	if gate.MaxSingleYuan != 1000 {
		t.Fatalf("T0 max_single_yuan want 1000, got %d", gate.MaxSingleYuan)
	}
	if gate.MaxDailyYuan != 5000 {
		t.Fatalf("T0 max_daily_yuan want 5000, got %d", gate.MaxDailyYuan)
	}
}

func TestKycEvaluateVerifiedPhoneToT1WithAudit(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userID, err := createUserWithPhoneLogin("+86", "13800001111", "hash", "")
	if err != nil {
		t.Fatalf("create phone user: %v", err)
	}

	body := strings.NewReader(`{"user_id":"` + userID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/kyc/evaluate/", body)
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	req.Header.Set("X-Request-Id", "req-kyc-eval-1")
	rec := httptest.NewRecorder()
	handleInternalKycEvaluate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("evaluate: %d %s", rec.Code, rec.Body.String())
	}
	var profile kycProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Tier != kycTierT1 || profile.Status != kycStatusApproved {
		t.Fatalf("want T1/approved, got %s/%s", profile.Tier, profile.Status)
	}

	audits, err := listKycAudit(userID, 10)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(audits) < 1 {
		t.Fatal("expected audit entry")
	}
	if audits[0].NewTier != kycTierT1 || audits[0].TriggerSource != kycTriggerEvaluate {
		t.Fatalf("audit=%+v", audits[0])
	}

	gate, err := checkKycRechargeGate(userID, 1000)
	if err != nil {
		t.Fatalf("gate: %v", err)
	}
	if !gate.Allowed || gate.MaxSingleYuan != 4999 {
		t.Fatalf("gate=%+v", gate)
	}
}

func TestKycAdminOverrideToT2(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userID, _, err := createUserWithEmailLogin("kyc-admin@test.com", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	body := strings.NewReader(`{
		"user_id":"` + userID + `",
		"tier":"T2_enhanced",
		"status":"approved",
		"reason_code":"MANUAL_REVIEW",
		"reason_detail":"enhanced after review",
		"actor_id":"admin-1"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/kyc/admin-override/", body)
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalKycAdminOverride(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("override: %d %s", rec.Code, rec.Body.String())
	}
	var profile kycProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Tier != kycTierT2 || profile.Status != kycStatusApproved {
		t.Fatalf("want T2/approved, got %s/%s", profile.Tier, profile.Status)
	}

	audits, err := listKycAudit(userID, 10)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(audits) != 1 || audits[0].TriggerSource != kycTriggerAdmin || audits[0].ActorID != "admin-1" {
		t.Fatalf("audit=%+v", audits)
	}

	gate, err := checkKycRechargeGate(userID, 10000)
	if err != nil {
		t.Fatalf("gate: %v", err)
	}
	if !gate.Allowed || gate.MaxSingleYuan != 50000 {
		t.Fatalf("gate=%+v", gate)
	}
}

func TestKycRechargeGateDeniesAmlReview(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userID, err := createUserWithPhoneLogin("+86", "13800002222", "hash", "")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := evaluateKyc(context.Background(), userID); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	body := strings.NewReader(`{
		"user_id":"` + userID + `",
		"result":"review",
		"provider":"manual",
		"notes":"stub review",
		"actor_id":"aml-ops",
		"screening_ref":"ref-1"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/kyc/aml-screening/", body)
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalKycAmlScreening(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("aml: %d %s", rec.Code, rec.Body.String())
	}

	gateReq := httptest.NewRequest(http.MethodGet, "/api/internal/kyc/recharge-gate/?user_id="+userID+"&amount_yuan=100", nil)
	gateReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	gateRec := httptest.NewRecorder()
	handleInternalKycRechargeGate(gateRec, gateReq)
	if gateRec.Code != http.StatusOK {
		t.Fatalf("gate: %d %s", gateRec.Code, gateRec.Body.String())
	}
	var gate kycRechargeGateResult
	if err := json.Unmarshal(gateRec.Body.Bytes(), &gate); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gate.Allowed || gate.ReasonCode != reasonKycAMLReview {
		t.Fatalf("gate=%+v", gate)
	}
}

func TestKycLimitPolicySeedAndPut(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	req := httptest.NewRequest(http.MethodGet, "/api/internal/kyc/limit-policy/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalKycLimitPolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	results, ok := payload["results"].([]interface{})
	if !ok || len(results) != 3 {
		t.Fatalf("results=%v", payload["results"])
	}

	putBody := strings.NewReader(`{"tier":"T1_basic","max_single_yuan":4000,"max_daily_yuan":15000,"recharge_allowed":true}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/internal/kyc/limit-policy/", putBody)
	putReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	putRec := httptest.NewRecorder()
	handleInternalKycLimitPolicy(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put: %d %s", putRec.Code, putRec.Body.String())
	}
	p, err := getKycLimitPolicy(kycTierT1)
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if p.MaxSingleYuan != 4000 || p.MaxDailyYuan != 15000 {
		t.Fatalf("policy=%+v", p)
	}
}

func TestKycEventTopicConstants(t *testing.T) {
	if eventTopicMap["KYC_TIER_CHANGED"] != "kyc-tier-changed" {
		t.Fatalf("KYC_TIER_CHANGED topic=%q", eventTopicMap["KYC_TIER_CHANGED"])
	}
	if eventTopicMap["KYC_STATUS_CHANGED"] != "kyc-status-changed" {
		t.Fatalf("KYC_STATUS_CHANGED topic=%q", eventTopicMap["KYC_STATUS_CHANGED"])
	}
	if eventTopicMap["AML_SCREENING_RECORDED"] != "aml-screening-recorded" {
		t.Fatalf("AML_SCREENING_RECORDED topic=%q", eventTopicMap["AML_SCREENING_RECORDED"])
	}
}

func TestKycMigrationTablesExist(t *testing.T) {
	setupAuthTestDB(t)
	for _, table := range []string{
		"auth_kyc_profile",
		"auth_kyc_audit_log",
		"auth_aml_screening_record",
		"auth_kyc_limit_policy",
	} {
		var name string
		err := db.QueryRow(
			`SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}
}

// ── Public KYC endpoint tests ────────────────────────────────────────────

// makeStaff promotes a user to staff (is_staff=1).
func makeStaff(userID string) {
	if _, err := db.Exec(`UPDATE auth_user SET is_staff = 1 WHERE id = ?`, userID); err != nil {
		panic("makeStaff: " + err.Error())
	}
}

// makeSuperuser promotes a user to superuser (is_superuser=1, is_staff=1).
func makeSuperuser(userID string) {
	if _, err := db.Exec(`UPDATE auth_user SET is_superuser = 1, is_staff = 1 WHERE id = ?`, userID); err != nil {
		panic("makeSuperuser: " + err.Error())
	}
}

func TestPublicKycMeUnauthenticatedReturns401(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/kyc/me/", nil)
	rec := httptest.NewRecorder()
	handlePublicKycMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicKycMeAuthenticatedReturnsSummary(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("kyc-me@test.com", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/kyc/me/", nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handlePublicKycMe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if summary["user_id"] != userID {
		t.Fatalf("user_id: want %q, got %v", userID, summary["user_id"])
	}
	if summary["tier"] != "T0_unverified" {
		t.Fatalf("tier: want T0_unverified, got %v", summary["tier"])
	}
	if summary["status"] != "none" {
		t.Fatalf("status: want none, got %v", summary["status"])
	}
	if v, ok := summary["max_single_yuan"].(float64); !ok || v != 1000 {
		t.Fatalf("max_single_yuan: want 1000, got %v", summary["max_single_yuan"])
	}
	if v, ok := summary["recharge_allowed"].(bool); !ok || !v {
		t.Fatalf("recharge_allowed: want true, got %v", summary["recharge_allowed"])
	}
}

func TestPublicKycAdminEndpointsRequireStaff(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("kyc-regular@test.com", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	targetID := "some-other-user"
	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/kyc/admin/users/" + targetID + "/", ""},
		{http.MethodPost, "/api/kyc/admin/users/" + targetID + "/override/", `{"tier":"T1_basic","status":"approved"}`},
		{http.MethodPost, "/api/kyc/admin/users/" + targetID + "/aml/", `{"result":"clear","provider":"manual"}`},
		{http.MethodPost, "/api/kyc/admin/users/" + targetID + "/evaluate/", ""},
	}

	for _, ep := range endpoints {
		var req *http.Request
		if ep.body != "" {
			req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		} else {
			req = httptest.NewRequest(ep.method, ep.path, nil)
		}
		req.SetPathValue("user_id", targetID)
		req.Header.Set("Authorization", "Token "+token)
		rec := httptest.NewRecorder()

		switch ep.path {
		case "/api/kyc/admin/users/" + targetID + "/":
			handlePublicKycAdminGet(rec, req)
		case "/api/kyc/admin/users/" + targetID + "/override/":
			handlePublicKycAdminOverride(rec, req)
		case "/api/kyc/admin/users/" + targetID + "/aml/":
			handlePublicKycAdminAml(rec, req)
		case "/api/kyc/admin/users/" + targetID + "/evaluate/":
			handlePublicKycAdminEvaluate(rec, req)
		}

		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s: want 403, got %d body=%s", ep.method, ep.path, rec.Code, rec.Body.String())
		}
	}
}

func TestPublicKycAdminEndpointsUnauthenticatedReturns401(t *testing.T) {
	setupAuthTestDB(t)

	targetID := "some-user"
	endpoints := []struct {
		method string
		path   string
		body   string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{http.MethodGet, "/api/kyc/admin/users/" + targetID + "/", "", handlePublicKycAdminGet},
		{http.MethodPost, "/api/kyc/admin/users/" + targetID + "/override/", `{"tier":"T1_basic","status":"approved"}`, handlePublicKycAdminOverride},
		{http.MethodPost, "/api/kyc/admin/users/" + targetID + "/aml/", `{"result":"clear"}`, handlePublicKycAdminAml},
		{http.MethodPost, "/api/kyc/admin/users/" + targetID + "/evaluate/", "", handlePublicKycAdminEvaluate},
	}

	for _, ep := range endpoints {
		var req *http.Request
		if ep.body != "" {
			req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		} else {
			req = httptest.NewRequest(ep.method, ep.path, nil)
		}
		req.SetPathValue("user_id", targetID)
		rec := httptest.NewRecorder()
		ep.handler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: want 401, got %d body=%s", ep.method, ep.path, rec.Code, rec.Body.String())
		}
	}
}

func TestPublicKycAdminGetAsStaff(t *testing.T) {
	setupAuthTestDB(t)
	adminID, _, err := createUserWithEmailLogin("kyc-staff@test.com", "password123")
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}
	makeStaff(adminID)

	targetID, _, err := createUserWithEmailLogin("kyc-target@test.com", "password123")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	// Give the target a KYC profile by evaluating.
	if _, err := evaluateKyc(context.Background(), targetID); err != nil {
		t.Fatalf("evaluate target: %v", err)
	}

	token, err := getOrCreateToken(adminID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/kyc/admin/users/"+targetID+"/", nil)
	req.SetPathValue("user_id", targetID)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handlePublicKycAdminGet(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var aggregate map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &aggregate); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if aggregate["user_id"] != targetID {
		t.Fatalf("user_id: want %q, got %v", targetID, aggregate["user_id"])
	}
	if aggregate["profile"] == nil {
		t.Fatal("profile should not be nil")
	}
	if aggregate["audit"] == nil {
		t.Fatal("audit should not be nil (empty array)")
	}
	// OPT-20260819-029：超管 GET 附带实时限额策略数组（可为空但不为 null）。
	policies, ok := aggregate["limit_policies"].([]interface{})
	if !ok {
		t.Fatalf("limit_policies should be an array, got %T %v", aggregate["limit_policies"], aggregate["limit_policies"])
	}
	_ = policies
}

func TestPublicKycAdminOverrideAsSuperuser(t *testing.T) {
	setupAuthTestDB(t)
	adminID, _, err := createUserWithEmailLogin("kyc-super@test.com", "password123")
	if err != nil {
		t.Fatalf("create superuser: %v", err)
	}
	makeSuperuser(adminID)

	targetID, _, err := createUserWithEmailLogin("kyc-ov-target@test.com", "password123")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	token, err := getOrCreateToken(adminID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	body := strings.NewReader(`{"tier":"T2_enhanced","status":"approved","reason_code":"MANUAL_REVIEW","reason_detail":"public api test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/kyc/admin/users/"+targetID+"/override/", body)
	req.SetPathValue("user_id", targetID)
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlePublicKycAdminOverride(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var profile kycProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Tier != kycTierT2 || profile.Status != kycStatusApproved {
		t.Fatalf("want T2/approved, got %s/%s", profile.Tier, profile.Status)
	}
}

func TestPublicKycAdminEvaluateAsStaff(t *testing.T) {
	setupAuthTestDB(t)
	adminID, _, err := createUserWithEmailLogin("kyc-eval-staff@test.com", "password123")
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}
	makeStaff(adminID)

	targetID, err := createUserWithPhoneLogin("+86", "13800003333", "hash", "")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	token, err := getOrCreateToken(adminID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/kyc/admin/users/"+targetID+"/evaluate/", nil)
	req.SetPathValue("user_id", targetID)
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("X-Request-Id", "pub-eval-1")
	rec := httptest.NewRecorder()
	handlePublicKycAdminEvaluate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var profile kycProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Tier != kycTierT1 || profile.Status != kycStatusApproved {
		t.Fatalf("phone-verified should become T1/approved, got %s/%s", profile.Tier, profile.Status)
	}
}
