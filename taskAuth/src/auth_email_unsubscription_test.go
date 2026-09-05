package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskAuth/domain"
)

func TestPublicUnsubscribeRoundTrip(t *testing.T) {
	setupAuthTestDB(t)
	prev := cfg.UnsubscribeHmacSecret
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	cfg.FrontendBase = "http://localhost:4000"
	t.Cleanup(func() { cfg.UnsubscribeHmacSecret = prev })

	tok, err := domain.SignUnsubscribeToken(cfg.UnsubscribeHmacSecret, "Skip.Me@Example.com")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/public/email-unsubscribe/?token="+tok, nil)
	rec := httptest.NewRecorder()
	handlePublicEmailUnsubscribe(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/auth/unsubscribe/?ok=1") {
		t.Fatalf("location=%s", loc)
	}
	if !isEmailUnsubscribed("skip.me@example.com") {
		t.Fatal("expected unsubscribed")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/public/email-unsubscribe/?token="+tok, nil)
	rec2 := httptest.NewRecorder()
	handlePublicEmailUnsubscribe(rec2, req2)
	if rec2.Code != http.StatusFound {
		t.Fatalf("repeat status=%d", rec2.Code)
	}
}

func TestPublicUnsubscribeOneClickPOST(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	tok, err := domain.SignUnsubscribeToken(cfg.UnsubscribeHmacSecret, "oneclick@example.com")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/public/email-unsubscribe/?token="+tok,
		strings.NewReader("List-Unsubscribe=One-Click"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handlePublicEmailUnsubscribe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) && !strings.Contains(rec.Body.String(), `"ok": true`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if !isEmailUnsubscribed("oneclick@example.com") {
		t.Fatal("expected unsubscribed")
	}
}

func TestPublicUnsubscribeInvalidToken(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	req := httptest.NewRequest(http.MethodGet, "/api/public/email-unsubscribe/?token=v1.bad.bad", nil)
	rec := httptest.NewRecorder()
	handlePublicEmailUnsubscribe(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "ok=0") {
		t.Fatalf("location=%s", rec.Header().Get("Location"))
	}
}

func TestCreateEmailInvitationSkipsWhenUnsubscribed(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	cfg.FrontendBase = "http://localhost:4000"
	if _, err := upsertEmailUnsubscription(httptest.NewRequest(http.MethodGet, "/", nil).Context(),
		"blocked-invite@example.com", domain.UnsubscribeSourceInvite); err != nil {
		t.Fatal(err)
	}
	adminID, _ := createTestUserWithToken(t, "admin-unsub@test.com")
	if err := ensureSuperAdminRow(adminID); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader(`{"email":"blocked-invite@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/email-invitations/", body)
	req.Header.Set("X-User-Id", adminID)
	rec := httptest.NewRecorder()
	handleCreateEmailInvitation(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "email_skipped") {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.EmailUnsubscribedHint) {
		t.Fatalf("missing hint: %s", rec.Body.String())
	}
	var status string
	err := db.QueryRow(`SELECT delivery_status FROM auth_email_registration_invite WHERE email = ?`,
		"blocked-invite@example.com").Scan(&status)
	if err != nil {
		t.Fatal(err)
	}
	if status != "skipped_unsubscribed" {
		t.Fatalf("status=%s", status)
	}
}

func TestInternalEmailUnsubscription(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	cfg.InternalSecret = ""
	req := httptest.NewRequest(http.MethodGet, "/api/internal/email-unsubscription/?email=nobody@example.com", nil)
	rec := httptest.NewRecorder()
	handleInternalEmailUnsubscription(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"unsubscribed":false`) && !strings.Contains(rec.Body.String(), `"unsubscribed": false`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestPublicEmailResubscribeRoundTrip(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	tok, err := domain.SignUnsubscribeToken(cfg.UnsubscribeHmacSecret, "Resubscribe@Example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := upsertEmailUnsubscription(httptest.NewRequest(http.MethodGet, "/", nil).Context(),
		"resubscribe@example.com", domain.UnsubscribeSourceInvite); err != nil {
		t.Fatal(err)
	}
	if !isEmailUnsubscribed("resubscribe@example.com") {
		t.Fatal("precondition: expected unsubscribed")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/public/email-resubscribe/?token="+tok, nil)
	req.Header.Set("Idempotency-Key", "ik-1")
	rec := httptest.NewRecorder()
	handlePublicEmailResubscribe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"resubscribed":true`) && !strings.Contains(rec.Body.String(), `"resubscribed": true`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if isEmailUnsubscribed("resubscribe@example.com") {
		t.Fatal("expected unsubscribed row removed")
	}
	// 幂等：重复提交仍 ok
	req2 := httptest.NewRequest(http.MethodPost, "/api/public/email-resubscribe/?token="+tok, nil)
	req2.Header.Set("Idempotency-Key", "ik-2")
	rec2 := httptest.NewRecorder()
	handlePublicEmailResubscribe(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("repeat status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	if isEmailUnsubscribed("resubscribe@example.com") {
		t.Fatal("repeat must remain resubscribed")
	}
}

func TestPublicEmailResubscribeRequiresIdempotencyKey(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	tok, err := domain.SignUnsubscribeToken(cfg.UnsubscribeHmacSecret, "no-idem@example.com")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/public/email-resubscribe/?token="+tok, nil)
	rec := httptest.NewRecorder()
	handlePublicEmailResubscribe(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicEmailResubscribeInvalidToken(t *testing.T) {
	setupAuthTestDB(t)
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	req := httptest.NewRequest(http.MethodPost, "/api/public/email-resubscribe/?token=v1.bad.bad", nil)
	req.Header.Set("Idempotency-Key", "ik-1")
	rec := httptest.NewRecorder()
	handlePublicEmailResubscribe(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicEmailResubscribeMethodNotAllowed(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/public/email-resubscribe/?token=x", nil)
	rec := httptest.NewRecorder()
	handlePublicEmailResubscribe(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestUnsubscribeRedirectCarriesResubscribeToken(t *testing.T) {
	setupAuthTestDB(t)
	prev := cfg.UnsubscribeHmacSecret
	cfg.UnsubscribeHmacSecret = "test-unsub-hmac-not-for-prod"
	cfg.FrontendBase = "http://localhost:4000"
	t.Cleanup(func() { cfg.UnsubscribeHmacSecret = prev })
	tok, err := domain.SignUnsubscribeToken(cfg.UnsubscribeHmacSecret, "carry-token@example.com")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/public/email-unsubscribe/?token="+tok, nil)
	rec := httptest.NewRecorder()
	handlePublicEmailUnsubscribe(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/auth/unsubscribe/?ok=1") {
		t.Fatalf("location=%s", loc)
	}
	if !strings.Contains(loc, "token=") {
		t.Fatalf("location must carry resubscribe token: %s", loc)
	}
}
