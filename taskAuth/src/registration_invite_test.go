package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setRegistrationInvitePolicyForTest(t *testing.T, enabled bool, dailyQuota int) {
	t.Helper()
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	if _, err := db.Exec(`
		UPDATE auth_registration_invite_policy
		SET enabled = ?, daily_quota = ?, updated_at = NOW()
		WHERE singleton_key = ?`, enabledInt, dailyQuota, registrationInvitePolicyKey); err != nil {
		t.Fatalf("set policy: %v", err)
	}
}

func createTestUserWithToken(t *testing.T, email string) (userID, token string) {
	t.Helper()
	var err error
	userID, _, err = createUserWithEmailLogin(email, "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err = getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return userID, token
}

func insertUnusedInviteCode(t *testing.T, code, issuerUserID string) {
	t.Helper()
	day := shanghaiCalendarDay(time.Now())
	id := "test-invite-" + code
	_, err := db.Exec(`
		INSERT INTO auth_registration_invite_code (
			id, code, issuer_user_id, status, issued_day, created_at
		) VALUES (?, ?, ?, 'unused', ?, NOW())`,
		id, code, issuerUserID, day,
	)
	if err != nil {
		t.Fatalf("insert invite code: %v", err)
	}
}

// TestRegistrationInviteEmailRegisterRejected verifies email registration is
// invite-only — without invite_token it is rejected regardless of policy.
func TestRegistrationInviteEmailRegisterRejected(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, false, 0)

	body := strings.NewReader(`{"email":"invite-off@test.com","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/email_register/", body)
	rec := httptest.NewRecorder()
	handleEmailRegister(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (email invite-only), got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRegistrationInviteEnabledWithoutCodeRejected(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, true, 5)

	body := strings.NewReader(`{"email":"need-code@test.com","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/email_register/", body)
	rec := httptest.NewRecorder()
	handleEmailRegister(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["error"] != "邮箱注册需要邀请，请使用手机号注册或联系管理员获取邀请" {
		t.Fatalf("error=%q", payload["error"])
	}
}

func TestRegistrationInviteInvalidCodeRejected(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, true, 5)

	body := strings.NewReader(`{"email":"bad-code@test.com","password":"password123","invite_token":"ZZZZZZZZ"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/email_register/", body)
	rec := httptest.NewRecorder()
	handleEmailRegister(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["error"] != "邀请链接无效或已过期" {
		t.Fatalf("error=%q", payload["error"])
	}
}

// TestRegistrationInviteEmailRegisterWithCodeRejected verifies email registration
// requires email invite token — invite codes alone are not sufficient.
func TestRegistrationInviteEmailRegisterWithCodeRejected(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, true, 5)
	issuerID, _ := createTestUserWithToken(t, "issuer@test.com")
	insertUnusedInviteCode(t, "ABCD2345", issuerID)

	body := strings.NewReader(`{"email":"redeem@test.com","password":"password123","invite_code":"ABCD2345"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/email_register/", body)
	rec := httptest.NewRecorder()
	handleEmailRegister(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (needs email invite token), got %d body=%s", rec.Code, rec.Body.String())
	}

	// Old invite code should NOT have been consumed
	status, err := lookupInviteCodeStatus("ABCD2345")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if status != "unused" {
		t.Fatalf("expected unused (should not consume), got %q", status)
	}
}

func TestRegistrationInviteQuotaExhaustedOnApply(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, true, 1)
	userID, token := createTestUserWithToken(t, "apply@test.com")

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/registration-invite-codes/apply/", nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleApplyRegistrationInviteCode(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first apply expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/accounts/users/registration-invite-codes/apply/", nil)
	req2.Header.Set("Authorization", "Token "+token)
	rec2 := httptest.NewRecorder()
	handleApplyRegistrationInviteCode(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("second apply expected 400, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["error"] != "daily_quota_exhausted" {
		t.Fatalf("error=%q user=%s", payload["error"], userID)
	}
}

func TestRegistrationInviteNonSuperuserPutPolicyForbidden(t *testing.T) {
	setupAuthTestDB(t)
	_, token := createTestUserWithToken(t, "regular@test.com")

	body := strings.NewReader(`{"enabled":true,"daily_quota":10}`)
	req := httptest.NewRequest(http.MethodPut, "/api/system-admin/registration-invite-policy/", body)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleSystemAdminRegistrationInvitePolicy(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRegistrationInviteSuperuserPutPolicyViaXUserID(t *testing.T) {
	setupAuthTestDB(t)
	adminID, _ := createTestUserWithToken(t, "admin@test.com")
	if err := ensureSuperAdminRow(adminID); err != nil {
		t.Fatalf("ensure super admin: %v", err)
	}

	body := strings.NewReader(`{"enabled":true,"daily_quota":3}`)
	req := httptest.NewRequest(http.MethodPut, "/api/system-admin/registration-invite-policy/", body)
	req.Header.Set("X-User-Id", adminID)
	rec := httptest.NewRecorder()
	handleSystemAdminRegistrationInvitePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["enabled"] != true {
		t.Fatalf("enabled=%v", payload["enabled"])
	}
	if payload["daily_quota"].(float64) != 3 {
		t.Fatalf("daily_quota=%v", payload["daily_quota"])
	}
}

func TestResolveUserIDFromRequestPrefersXUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Id", "gateway-user-1")
	req.Header.Set("Authorization", "Token should-not-use")
	uid, ok := resolveUserIDFromRequest(req)
	if !ok || uid != "gateway-user-1" {
		t.Fatalf("got uid=%q ok=%v", uid, ok)
	}
}

func TestRedeemInviteCodeErrors(t *testing.T) {
	setupAuthTestDB(t)
	issuerID, _ := createTestUserWithToken(t, "issuer2@test.com")
	insertUnusedInviteCode(t, "WXYZ5678", issuerID)

	newUserID, _, _ := createUserWithEmailLogin("newuser@test.com", "password123")
	if err := redeemInviteCode(context.Background(), "WXYZ5678", newUserID); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if err := redeemInviteCode(context.Background(), "WXYZ5678", newUserID); !errors.Is(err, errInviteCodeUsed) {
		t.Fatalf("expected used, got %v", err)
	}
}

func TestApplyRegistrationInviteCodeDisabled(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, false, 5)
	_, token := createTestUserWithToken(t, "apply-disabled@test.com")

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/registration-invite-codes/apply/", nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleApplyRegistrationInviteCode(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["error"] != "invite_feature_disabled" {
		t.Fatalf("error=%q", payload["error"])
	}
}

func TestPublicRegistrationInvitePolicy(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, true, 2)

	req := httptest.NewRequest(http.MethodGet, "/api/public/registration-invite-policy/", nil)
	rec := httptest.NewRecorder()
	handlePublicRegistrationInvitePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["enabled"] != true {
		t.Fatalf("enabled=%v", payload["enabled"])
	}
	if payload["remaining_today"].(float64) != 2 {
		t.Fatalf("remaining_today=%v", payload["remaining_today"])
	}
}

func TestGenerateRegistrationInviteCodeAlphabet(t *testing.T) {
	for i := 0; i < 32; i++ {
		code, err := generateRegistrationInviteCode()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if len(code) != 8 {
			t.Fatalf("len=%d code=%q", len(code), code)
		}
		for _, ch := range code {
			if !strings.ContainsRune(inviteCodeAlphabet, ch) {
				t.Fatalf("invalid char %q in %q", ch, code)
			}
		}
	}
}
