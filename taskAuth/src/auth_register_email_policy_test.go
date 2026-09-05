package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEmailRegisterRejectedWithoutInviteToken verifies that email registration
// without a valid invite_token is rejected (invite-only email registration).
func TestEmailRegisterRejectedWithoutInviteToken(t *testing.T) {
	setupAuthTestDB(t)

	body := strings.NewReader(`{"email":"rejected@test.com","password":"password123"}`)
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
	if !strings.Contains(payload["error"], "邀请") {
		t.Fatalf("expected invite-related error, got %q", payload["error"])
	}
}

