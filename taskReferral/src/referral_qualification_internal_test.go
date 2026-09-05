package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInternalQualificationActiveApproved(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "paytime-qual-ok", time.Now().AddDate(0, 3, 0))

	req := httptest.NewRequest(http.MethodGet, "/api/internal/referral/qualification/active/?user_id=paytime-qual-ok", nil)
	rec := httptest.NewRecorder()
	handleInternalQualificationActive(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		UserID string `json:"user_id"`
		Active bool   `json:"active"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.UserID != "paytime-qual-ok" || !resp.Active {
		t.Fatalf("resp=%+v want active approved", resp)
	}
}

func TestInternalQualificationActiveRevokedAndMissing(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "paytime-qual-revoked", time.Now().AddDate(0, 3, 0))
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "paytime-qual-revoked").Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if _, err := revokeReferralQualification(context.Background(), appID, "admin-1", "违反推荐政策取消资格", "ik-paytime-rev"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/referral/qualification/active/?user_id=paytime-qual-revoked", nil)
	rec := httptest.NewRecorder()
	handleInternalQualificationActive(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["active"] != false {
		t.Fatalf("revoked must be inactive, got %v", resp["active"])
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/referral/qualification/active/?user_id=nobody", nil)
	rec2 := httptest.NewRecorder()
	handleInternalQualificationActive(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("missing status=%d", rec2.Code)
	}
	var resp2 map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatal(err)
	}
	if resp2["active"] != false {
		t.Fatalf("missing user must be inactive, got %v", resp2["active"])
	}
}

func TestInternalQualificationActiveBatchApprovedAndMissing(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "batch-qual-yes", time.Now().AddDate(0, 3, 0))
	insertApprovedReferral(t, "batch-qual-expired", time.Now().AddDate(0, -1, 0))

	body := `{"user_ids":["batch-qual-yes","batch-qual-expired","batch-qual-none","batch-qual-yes"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/referral/qualification/active/batch/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalQualificationActiveBatch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Qualifications map[string]bool `json:"qualifications"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Qualifications["batch-qual-yes"] {
		t.Fatalf("approved unexpired must be true, got %v", resp.Qualifications)
	}
	if resp.Qualifications["batch-qual-expired"] {
		t.Fatalf("expired must be false, got %v", resp.Qualifications)
	}
	if resp.Qualifications["batch-qual-none"] {
		t.Fatalf("missing code must be false, got %v", resp.Qualifications)
	}
}

func TestInternalQualificationActiveBatchEmptyUserIDs(t *testing.T) {
	setupTestReferralDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/referral/qualification/active/batch/", strings.NewReader(`{"user_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalQualificationActiveBatch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalQualificationActiveRequiresUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/referral/qualification/active/", nil)
	rec := httptest.NewRecorder()
	handleInternalQualificationActive(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}
