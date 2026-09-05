package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAuthVerifyRejectsMissingAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/verify", nil)
	rec := httptest.NewRecorder()
	handleAuthVerify(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAuthVerifyAcceptsToken(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/verify", nil)
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleAuthVerify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), userID) {
		t.Fatalf("expected body to contain user id %s, got %s", userID, rec.Body.String())
	}
}
