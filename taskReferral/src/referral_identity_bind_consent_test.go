package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApplyReferralCodeRequiresIdentityBindConsent(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	_, err := applyReferralCode("user-consent-missing", testValidPersonalIntro, testValidLegalName, false)
	if !errors.Is(err, errReferralIdentityBindConsentRequired) {
		t.Fatalf("missing consent err=%v", err)
	}

	result, err := applyReferralCode("user-consent-ok", testValidPersonalIntro, testValidLegalName, true)
	if err != nil {
		t.Fatalf("consented apply: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("status=%s", result.Status)
	}

	var consentedAt time.Time
	if err := db.QueryRow(
		`SELECT identity_bind_consented_at FROM referral_code WHERE user_id = ?`,
		"user-consent-ok",
	).Scan(&consentedAt); err != nil {
		t.Fatalf("query consent: %v", err)
	}
	if consentedAt.IsZero() {
		t.Fatal("identity_bind_consented_at empty")
	}
}

func TestHandleReferralCodeApplyRequiresIdentityBindConsent(t *testing.T) {
	mux := setupTestMux(t)

	body, _ := json.Marshal(map[string]string{
		"personal_intro": testValidPersonalIntro,
		"legal_name":     testValidLegalName,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/referral-codes/apply/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withReferralUser(req, "user-http-consent")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing consent code=%d body=%s", rec.Code, rec.Body.String())
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v", err)
	}
	if parsed["error"] != "identity_bind_consent_required" {
		t.Fatalf("error=%s", parsed["error"])
	}

	okReq := newReferralApplyRequest("user-http-consent", testValidPersonalIntro)
	okRec := httptest.NewRecorder()
	mux.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("consented apply code=%d body=%s", okRec.Code, okRec.Body.String())
	}
}
