package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func insertEmailInviteForExpire(t *testing.T, id int64, email, token, status, expiresSQL string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO auth_email_registration_invite
		(id, email, token, inviter_user_id, status, expires_at, created_at,
		 email_sent_at, email_send_attempts, delivery_status, delivery_error)
		VALUES (?, ?, ?, '0', ?, `+expiresSQL+`, NOW(), NOW(), 0, 'pending', '')`,
		id, email, token, status)
	if err != nil {
		t.Fatalf("insert invite %d: %v", id, err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM auth_email_registration_invite WHERE id = ?`, id)
	})
}

func inviteStatus(t *testing.T, id int64) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM auth_email_registration_invite WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("status %d: %v", id, err)
	}
	return status
}

func TestCleanupExpiredEmailInvitesMarksOnlyDuePending(t *testing.T) {
	setupAuthTestDB(t)
	dueID := int64(99999111)
	freshID := int64(99999112)
	acceptedID := int64(99999113)
	insertEmailInviteForExpire(t, dueID, "due-expire@example.com", "tok-due-expire", "pending", "DATE_SUB(NOW(), INTERVAL 1 DAY)")
	insertEmailInviteForExpire(t, freshID, "fresh-expire@example.com", "tok-fresh-expire", "pending", "DATE_ADD(NOW(), INTERVAL 7 DAY)")
	insertEmailInviteForExpire(t, acceptedID, "accepted-expire@example.com", "tok-accepted-expire", "accepted", "DATE_SUB(NOW(), INTERVAL 1 DAY)")

	n, err := cleanupExpiredEmailInvites()
	if err != nil {
		t.Fatalf("cleanupExpiredEmailInvites: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count=%d want 1", n)
	}
	if got := inviteStatus(t, dueID); got != "expired" {
		t.Fatalf("due status=%s want expired", got)
	}
	if got := inviteStatus(t, freshID); got != "pending" {
		t.Fatalf("fresh status=%s want pending", got)
	}
	if got := inviteStatus(t, acceptedID); got != "accepted" {
		t.Fatalf("accepted status=%s want accepted", got)
	}

	n2, err := cleanupExpiredEmailInvites()
	if err != nil {
		t.Fatalf("second cleanup: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("idempotent expired count=%d want 0", n2)
	}
}

func TestInternalEmailInvitesExpireDueEndpoint(t *testing.T) {
	setupAuthTestDB(t)
	dueID := int64(99999121)
	insertEmailInviteForExpire(t, dueID, "ep-expire@example.com", "tok-ep-expire", "pending", "DATE_SUB(NOW(), INTERVAL 2 HOUR)")

	origSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-internal-secret"
	t.Cleanup(func() { cfg.InternalSecret = origSecret })

	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/email-invites/expire-due/", nil)
	rec := httptest.NewRecorder()
	handleInternalEmailInvitesExpireDue(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without secret, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/internal/taskauth/email-invites/expire-due/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec = httptest.NewRecorder()
	handleInternalEmailInvitesExpireDue(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 GET, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/email-invites/expire-due/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec = httptest.NewRecorder()
	handleInternalEmailInvitesExpireDue(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	expired, ok := body["expired"].(float64)
	if !ok || expired < 1 {
		t.Fatalf("expected expired >= 1, got %v", body["expired"])
	}
	if got := inviteStatus(t, dueID); got != "expired" {
		t.Fatalf("due status=%s want expired", got)
	}
}
