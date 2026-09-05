package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAdminRevokeAndAudit(t *testing.T) {
	mux := setupTestMux(t)
	insertApprovedReferral(t, "revoke-http-1", time.Now().AddDate(0, 3, 0))
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "revoke-http-1").Scan(&appID); err != nil {
		t.Fatal(err)
	}

	body := `{"reason":"违反推荐政策取消资格"}`
	req := withReferralUser(httptest.NewRequest(http.MethodPost,
		"/api/system-admin/referral/applications/"+itoa(appID)+"/revoke/",
		bytes.NewBufferString(body)), "super1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "ik-http-revoke-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("revoke code=%d body=%s", rec.Code, rec.Body.String())
	}

	auditReq := withReferralUser(httptest.NewRequest(http.MethodGet,
		"/api/system-admin/referral/applications/"+itoa(appID)+"/audit/", nil), "super1")
	auditRec := httptest.NewRecorder()
	mux.ServeHTTP(auditRec, auditReq)
	if auditRec.Code != 200 {
		t.Fatalf("audit code=%d body=%s", auditRec.Code, auditRec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(auditRec.Body.Bytes(), &out)
	if out["total"].(float64) < 1 {
		t.Fatalf("expected audit rows, got %v", out)
	}

	deny := withReferralUser(httptest.NewRequest(http.MethodPost,
		"/api/system-admin/referral/applications/"+itoa(appID)+"/revoke/",
		bytes.NewBufferString(body)), "user1")
	denyRec := httptest.NewRecorder()
	mux.ServeHTTP(denyRec, deny)
	if denyRec.Code != 403 {
		t.Fatalf("expected 403, got %d", denyRec.Code)
	}
}

func TestAdminApproveIdempotentReplay(t *testing.T) {
	mux := setupTestMux(t)
	setReferralPolicyForTest(t, "approval", "")
	applyReq := newReferralApplyRequest("approve-idem-1", testValidPersonalIntro)
	applyRec := httptest.NewRecorder()
	mux.ServeHTTP(applyRec, applyReq)
	if applyRec.Code != 200 {
		t.Fatalf("apply code=%d body=%s", applyRec.Code, applyRec.Body.String())
	}
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "approve-idem-1").Scan(&appID); err != nil {
		t.Fatal(err)
	}
	body := `{"reason":"资料充分予以通过"}`
	doApprove := func() (*httptest.ResponseRecorder, map[string]interface{}) {
		req := withReferralUser(httptest.NewRequest(http.MethodPost,
			"/api/system-admin/referral/applications/"+itoa(appID)+"/approve/",
			bytes.NewBufferString(body)), "super1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "ik-approve-replay-1")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var out map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}
	rec1, out1 := doApprove()
	if rec1.Code != 200 || out1["status"] != "approved" {
		t.Fatalf("first approve code=%d body=%s", rec1.Code, rec1.Body.String())
	}
	rec2, out2 := doApprove()
	if rec2.Code != 200 || out2["status"] != "approved" || out2["message"] != "already_approved" {
		t.Fatalf("replay code=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var total float64
	if err := db.QueryRow(`SELECT COUNT(*) FROM referral_qualification_audit WHERE application_id=?`, appID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("expected 1 audit row, got %v", total)
	}
}

func TestAdminRejectIdempotentReplay(t *testing.T) {
	mux := setupTestMux(t)
	setReferralPolicyForTest(t, "approval", "")
	applyReq := newReferralApplyRequest("reject-idem-1", testValidPersonalIntro)
	applyRec := httptest.NewRecorder()
	mux.ServeHTTP(applyRec, applyReq)
	if applyRec.Code != 200 {
		t.Fatalf("apply code=%d body=%s", applyRec.Code, applyRec.Body.String())
	}
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "reject-idem-1").Scan(&appID); err != nil {
		t.Fatal(err)
	}
	body := `{"reason":"申请人资料不充分，不符合推荐人准入标准"}`
	doReject := func() (*httptest.ResponseRecorder, map[string]interface{}) {
		req := withReferralUser(httptest.NewRequest(http.MethodPost,
			"/api/system-admin/referral/applications/"+itoa(appID)+"/reject/",
			bytes.NewBufferString(body)), "super1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "ik-reject-replay-1")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var out map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}
	rec1, out1 := doReject()
	if rec1.Code != 200 || out1["status"] != "rejected" {
		t.Fatalf("first reject code=%d body=%s", rec1.Code, rec1.Body.String())
	}
	rec2, out2 := doReject()
	if rec2.Code != 200 || out2["status"] != "rejected" || out2["message"] != "already_rejected" {
		t.Fatalf("replay code=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var total float64
	if err := db.QueryRow(`SELECT COUNT(*) FROM referral_qualification_audit WHERE application_id=?`, appID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("expected 1 audit row, got %v", total)
	}
}
