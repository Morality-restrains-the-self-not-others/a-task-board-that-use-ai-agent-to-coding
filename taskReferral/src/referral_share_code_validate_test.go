package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postShareCodeValidate(t *testing.T, body string, withSecret bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/referral/share-code/validate/", strings.NewReader(body))
	if withSecret {
		req.Header.Set("X-TaskReferral-Internal-Secret", "test-secret")
	}
	rec := httptest.NewRecorder()
	handleInternalShareCodeValidate(rec, req)
	return rec
}

func insertShareCodeForTest(t *testing.T, code, userID, status string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO referral_share_code (code, user_id, channel_name, is_default, status)
		VALUES (?, ?, '默认', 1, ?)
		ON DUPLICATE KEY UPDATE user_id = VALUES(user_id), status = VALUES(status)`,
		code, userID, status)
	if err != nil {
		t.Fatalf("insert share code: %v", err)
	}
}

func TestShareCodeValidateActiveCode(t *testing.T) {
	setupTestReferralDB(t)
	ensureShareCodeTable(t)
	insertShareCodeForTest(t, "AbC12345xY", "user-owner-1", "active")

	rec := postShareCodeValidate(t, `{"code":"AbC12345xY"}`, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["valid"] != true {
		t.Fatalf("valid=%v want true body=%s", out["valid"], rec.Body.String())
	}
	if out["owner_user_id"] != "user-owner-1" {
		t.Fatalf("owner_user_id=%v", out["owner_user_id"])
	}
}

func TestShareCodeValidateUnknownCode(t *testing.T) {
	setupTestReferralDB(t)
	ensureShareCodeTable(t)

	rec := postShareCodeValidate(t, `{"code":"NOPE000000"}`, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["valid"] != false {
		t.Fatalf("valid=%v want false", out["valid"])
	}
	if out["reason"] != "unknown_code" {
		t.Fatalf("reason=%v", out["reason"])
	}
}

func TestShareCodeValidateDisabledCode(t *testing.T) {
	setupTestReferralDB(t)
	ensureShareCodeTable(t)
	insertShareCodeForTest(t, "Disabled0xY", "user-owner-2", "disabled")

	rec := postShareCodeValidate(t, `{"code":"Disabled0xY"}`, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["valid"] != false {
		t.Fatalf("valid=%v want false body=%s", out["valid"], rec.Body.String())
	}
}

func TestShareCodeValidateEmptyCode(t *testing.T) {
	setupTestReferralDB(t)
	ensureShareCodeTable(t)

	rec := postShareCodeValidate(t, `{}`, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["valid"] != false {
		t.Fatalf("valid=%v want false", out["valid"])
	}
}

func TestShareCodeValidateRequiresInternalSecret(t *testing.T) {
	t.Setenv("TASK_REFERRAL_INTERNAL_SECRET", "test-secret")
	setupTestReferralDB(t)
	ensureShareCodeTable(t)
	rec := postShareCodeValidate(t, `{"code":"AbC12345xY"}`, false)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("without secret expected 403, got %d", rec.Code)
	}
	rec = postShareCodeValidate(t, `{"code":"AbC12345xY"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("with secret expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestShareCodeValidateMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/referral/share-code/validate/", nil)
	rec := httptest.NewRecorder()
	handleInternalShareCodeValidate(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
