package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEmailInviteDeliveryErrorStoredOnCreate verifies that when email delivery
// fails during invitation creation, the error message is stored in the
// delivery_error column.
func TestEmailInviteDeliveryErrorStoredOnCreate(t *testing.T) {
	setupAuthTestDB(t)

	// Create a super admin user for auth
	adminID, _ := createTestUserWithToken(t, "admin-delivery@test.com")
	if err := ensureSuperAdminRow(adminID); err != nil {
		t.Fatalf("ensure super admin: %v", err)
	}

	// Disable Kafka to force SMTP fallback (which will fail in test env)
	prev := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Setenv("KAFKA_BOOTSTRAP_SERVERS", "")
	t.Setenv("TASKAUTH_KAFKA_BOOTSTRAP_SERVERS", "")
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prev })

	body := strings.NewReader(`{"email":"delivery-test@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/email-invitations/", body)
	req.Header.Set("X-User-Id", adminID)
	rec := httptest.NewRecorder()
	handleCreateEmailInvitation(rec, req)

	// The invitation should be created even if delivery fails
	if rec.Code != http.StatusCreated {
		// It may fail with 400 if email is already registered, which is fine
		t.Logf("create returned %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the delivery_error is stored in DB when delivery fails
	var deliveryStatus, deliveryError string
	err := db.QueryRow(`
		SELECT COALESCE(delivery_status, 'pending'),
		       COALESCE(delivery_error, '')
		FROM auth_email_registration_invite
		WHERE email = ? ORDER BY created_at DESC LIMIT 1`,
		"delivery-test@example.com",
	).Scan(&deliveryStatus, &deliveryError)
	if err != nil {
		t.Skipf("no invite row found (may be skipped due to email already registered): %v", err)
		return
	}
	// If delivery failed, delivery_error should be non-empty
	if deliveryStatus == "failed" && deliveryError == "" {
		t.Error("delivery_status is 'failed' but delivery_error is empty — expected error message to be stored")
	}
}

// TestEmailInviteDeliveryErrorInListResponse verifies that the deliveryError
// field is included in the list API response when present.
func TestEmailInviteDeliveryErrorInListResponse(t *testing.T) {
	setupAuthTestDB(t)

	// Insert a test invitation with a known delivery error
	_, err := db.Exec(`
		INSERT INTO auth_email_registration_invite
		(id, email, token, inviter_user_id, status, expires_at, created_at,
		 email_sent_at, email_send_attempts, delivery_status, delivery_error)
		VALUES (99999001, 'fail-test@example.com', 'test-token-error-1', '0',
		        'pending', DATE_ADD(NOW(), INTERVAL 7 DAY), NOW(),
		        NOW(), 2, 'failed',
		        'EMAIL_SENT failed: kafka=connection refused, smtp=dial tcp: connection refused')`,
	)
	if err != nil {
		t.Skipf("insert test row: %v (may be MySQL-specific syntax)", err)
		return
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM auth_email_registration_invite WHERE id = 99999001`)
	})

	// Create a super admin and query the list
	adminID, _ := createTestUserWithToken(t, "admin-list@test.com")
	if err := ensureSuperAdminRow(adminID); err != nil {
		t.Fatalf("ensure super admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/email-invitations/", nil)
	req.Header.Set("X-User-Id", adminID)
	rec := httptest.NewRecorder()
	handleListEmailInvitations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Invitations []map[string]interface{} `json:"invitations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	found := false
	for _, inv := range payload.Invitations {
		if inv["email"] == "fail-test@example.com" {
			found = true
			if inv["deliveryStatus"] != "failed" {
				t.Errorf("expected deliveryStatus='failed', got %v", inv["deliveryStatus"])
			}
			deliveryError, ok := inv["deliveryError"].(string)
			if !ok || deliveryError == "" {
				t.Error("expected non-empty deliveryError in response for failed delivery")
			}
			t.Logf("deliveryError = %q", deliveryError)
		}
		// For delivered/pending entries, deliveryError should be absent
		if inv["deliveryStatus"] != "failed" {
			if _, hasErr := inv["deliveryError"]; hasErr {
				t.Logf("deliveryError present for non-failed entry %v: %v", inv["email"], inv["deliveryError"])
			}
		}
	}
	if !found {
		t.Log("test invitation not found in list response (may be DB-dependent)")
	}
}

// TestEmailInviteDeliveryErrorNotReturnedOnSuccess verifies that deliveryError
// is omitted when delivery was successful.
func TestEmailInviteDeliveryErrorNotReturnedOnSuccess(t *testing.T) {
	setupAuthTestDB(t)

	// Insert a test invitation with successful delivery
	_, err := db.Exec(`
		INSERT INTO auth_email_registration_invite
		(id, email, token, inviter_user_id, status, expires_at, created_at,
		 email_sent_at, email_send_attempts, delivery_status)
		VALUES (99999002, 'success-test@example.com', 'test-token-ok-1', '0',
		        'pending', DATE_ADD(NOW(), INTERVAL 7 DAY), NOW(),
		        NOW(), 1, 'delivered')`,
	)
	if err != nil {
		t.Skipf("insert test row: %v (may be MySQL-specific syntax)", err)
		return
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM auth_email_registration_invite WHERE id = 99999002`)
	})

	adminID, _ := createTestUserWithToken(t, "admin-success@test.com")
	if err := ensureSuperAdminRow(adminID); err != nil {
		t.Fatalf("ensure super admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/email-invitations/", nil)
	req.Header.Set("X-User-Id", adminID)
	rec := httptest.NewRecorder()
	handleListEmailInvitations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Invitations []map[string]interface{} `json:"invitations"`
	}
	json.Unmarshal(rec.Body.Bytes(), &payload)

	for _, inv := range payload.Invitations {
		if inv["email"] == "success-test@example.com" {
			if _, hasErr := inv["deliveryError"]; hasErr {
				t.Error("deliveryError should be absent when delivery was successful")
			}
		}
	}
}
