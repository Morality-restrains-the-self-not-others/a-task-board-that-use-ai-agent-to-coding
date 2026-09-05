package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// seedEmailVerificationCode 插入一条邮箱验证码（is_used=0, expires_at 未来）。
func seedEmailVerificationCode(t *testing.T, email, code, userID string) {
	t.Helper()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	future := time.Now().UTC().Add(5 * time.Minute).Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_sms_verification_code (email, user_id, code, created_at, expires_at, is_used)
		VALUES (?, ?, ?, ?, ?, 0)`, email, userID, code, now, future)
	if err != nil {
		t.Fatalf("seed email code: %v", err)
	}
}

func TestBindEmail_RequiresLogin(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"vendor@example.com","code":"123456"}`))
	rec := httptest.NewRecorder()
	handleBindEmail(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", rec.Code)
	}
}

func TestBindEmail_InvalidCode(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-email-bind-01")
	seedEmailVerificationCode(t, "vendor@example.com", "654321", userID)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"vendor@example.com","code":"000000"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userID))
	rec := httptest.NewRecorder()
	handleBindEmail(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid code, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBindEmail_RejectSyntheticEmail(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-email-bind-02")
	// 合成邮箱（sso-<id>@sso.invalid）即使验证码存在也不允许绑定
	seedEmailVerificationCode(t, "sso-123@sso.invalid", "111111", userID)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"sso-123@sso.invalid","code":"111111"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userID))
	rec := httptest.NewRecorder()
	handleBindEmail(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 synthetic email, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBindEmail_SuccessAndTakenConflict(t *testing.T) {
	setupAuthTestDB(t)
	userA := mustCreateWechatTestUser(t, "u-email-bind-03")
	userB := mustCreateWechatTestUser(t, "u-email-bind-04")

	// A 绑定邮箱（验证码正确）→ 成功，profile 返回 has_email=true
	seedEmailVerificationCode(t, "vendor@example.com", "111111", userA)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"vendor@example.com","code":"111111"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userA))
	rec := httptest.NewRecorder()
	handleBindEmail(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	profile, err := buildUserProfileJSON(userA)
	if err != nil {
		t.Fatalf("build profile: %v", err)
	}
	if profile["has_email"] != true || profile["email"] != "vendor@example.com" {
		t.Fatalf("expected has_email=true/email=vendor@example.com, got %v", profile)
	}

	// B 绑定同一邮箱（验证码正确）→ 409 已被占用
	seedEmailVerificationCode(t, "vendor@example.com", "222222", userB)
	req2 := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"vendor@example.com","code":"222222"}`))
	req2.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userB))
	rec2 := httptest.NewRecorder()
	handleBindEmail(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 taken, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestBindEmail_ReplaceOwnEmail(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-email-bind-05")

	// 首次绑定
	seedEmailVerificationCode(t, "old@example.com", "111111", userID)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"old@example.com","code":"111111"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userID))
	rec := httptest.NewRecorder()
	handleBindEmail(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 first bind, got %d body=%s", rec.Code, rec.Body.String())
	}

	// 同人换绑新邮箱 → 成功且 identifier 更新
	seedEmailVerificationCode(t, "new@example.com", "222222", userID)
	req2 := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_email/",
		bytes.NewBufferString(`{"email":"new@example.com","code":"222222"}`))
	req2.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userID))
	rec2 := httptest.NewRecorder()
	handleBindEmail(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 replace, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	profile, err := buildUserProfileJSON(userID)
	if err != nil {
		t.Fatalf("build profile: %v", err)
	}
	if profile["email"] != "new@example.com" {
		t.Fatalf("expected replaced email new@example.com, got %v", profile["email"])
	}
}
