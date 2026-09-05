package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessTokenMatchesIdentifier(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("match@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := activateLoginMethodByEmail("match@test.com"); err != nil {
		t.Fatalf("activate: %v", err)
	}

	ok, err := accessTokenMatchesIdentifier(userID, "match@test.com")
	if err != nil || !ok {
		t.Fatalf("expected match for email, ok=%v err=%v", ok, err)
	}

	ok, err = accessTokenMatchesIdentifier(userID, "other@test.com")
	if err != nil || ok {
		t.Fatalf("expected no match for other email, ok=%v err=%v", ok, err)
	}
}

func TestHandleLoginWithAccessTokenUsernameMatch(t *testing.T) {
	setupAuthTestDB(t)
	email := "plugin@test.com"
	userID, _, err := createUserWithEmailLogin(email, "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := activateLoginMethodByEmail(email); err != nil {
		t.Fatalf("activate: %v", err)
	}

	_, rawToken, err := createAccessToken(userID, "chrome-plugin", "")
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}

	body := map[string]interface{}{
		"username":     email,
		"access_token": rawToken,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/login-with-access-token/", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLoginWithAccessToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["token"] == nil || payload["token"] == "" {
		t.Fatalf("expected session token in response: %+v", payload)
	}
}

func TestHandleLoginWithAccessTokenUsernameRequired(t *testing.T) {
	setupAuthTestDB(t)
	email := "required@test.com"
	userID, _, err := createUserWithEmailLogin(email, "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := activateLoginMethodByEmail(email); err != nil {
		t.Fatalf("activate: %v", err)
	}
	_, rawToken, err := createAccessToken(userID, "test", "")
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}

	body := map[string]interface{}{"access_token": rawToken}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/login-with-access-token/", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLoginWithAccessToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLoginWithAccessTokenUsernameMismatch(t *testing.T) {
	setupAuthTestDB(t)
	userA, _, err := createUserWithEmailLogin("user-a@test.com", "hash")
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	if err := activateLoginMethodByEmail("user-a@test.com"); err != nil {
		t.Fatalf("activate A: %v", err)
	}
	_, _, err = createUserWithEmailLogin("user-b@test.com", "hash")
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}
	if err := activateLoginMethodByEmail("user-b@test.com"); err != nil {
		t.Fatalf("activate B: %v", err)
	}

	_, rawToken, err := createAccessToken(userA, "chrome-plugin", "")
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}

	body := map[string]interface{}{
		"username":     "user-b@test.com",
		"access_token": rawToken,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/login-with-access-token/", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLoginWithAccessToken(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func activateLoginMethodByEmail(email string) error {
	lm, err := findLoginMethodByEmail(email)
	if err != nil || lm == nil {
		return err
	}
	return activateLoginMethod(lm.ID)
}
