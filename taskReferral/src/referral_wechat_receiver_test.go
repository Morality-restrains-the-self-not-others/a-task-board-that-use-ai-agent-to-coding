package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEnsureWechatProfitSharingReceiverLive_PostsUserID(t *testing.T) {
	var gotUser string
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/taskbill/profit-sharing/receivers/ensure/" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		gotSecret = r.Header.Get("X-TaskBill-Internal-Secret")
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("json: %v", err)
		}
		gotUser = body["user_id"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"registered","user_id":"` + gotUser + `"}`))
	}))
	t.Cleanup(srv.Close)

	prevURL, prevSec := cfg.BillServiceURL, cfg.BillInternalSecret
	cfg.BillServiceURL = srv.URL
	cfg.BillInternalSecret = "bill-secret"
	t.Cleanup(func() {
		cfg.BillServiceURL = prevURL
		cfg.BillInternalSecret = prevSec
	})

	got := ensureWechatProfitSharingReceiverLive("user-recv-1")
	if got.Status != "registered" {
		t.Fatalf("got=%+v", got)
	}
	if gotUser != "user-recv-1" {
		t.Fatalf("posted user=%s", gotUser)
	}
	if gotSecret != "bill-secret" {
		t.Fatalf("secret=%s", gotSecret)
	}
}

func TestApproveReferralApplication_EnsuresWechatReceiver(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")
	res, err := applyReferralWithConsent("user-approve-wx", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Status != "pending" {
		t.Fatalf("apply status=%s", res.Status)
	}
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "user-approve-wx").Scan(&appID); err != nil {
		t.Fatalf("id: %v", err)
	}

	var called string
	orig := ensureWechatProfitSharingReceiver
	ensureWechatProfitSharingReceiver = func(uid string) wechatReceiverStatus {
		called = uid
		return wechatReceiverStatus{Status: "registered"}
	}
	t.Cleanup(func() { ensureWechatProfitSharingReceiver = orig })

	if _, err := approveReferralApplication(context.Background(), appID, "admin-1", "申请人介绍充分予以通过", ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if called != "user-approve-wx" {
		t.Fatalf("ensure called with %q", called)
	}
}

func TestOpenModeApply_EnsuresWechatReceiver(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "open", "")
	var called string
	orig := ensureWechatProfitSharingReceiver
	ensureWechatProfitSharingReceiver = func(uid string) wechatReceiverStatus {
		called = uid
		return wechatReceiverStatus{Status: "registered"}
	}
	t.Cleanup(func() { ensureWechatProfitSharingReceiver = orig })

	res, err := applyReferralWithConsent("user-open-wx", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Status != "approved" {
		t.Fatalf("status=%s", res.Status)
	}
	if called != "user-open-wx" {
		t.Fatalf("ensure called with %q", called)
	}
}

func TestGetReferralCodeStatus_EnsuresReceiverWhenActive(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "user-status-wx", time.Now().AddDate(0, 1, 0))

	var called string
	orig := ensureWechatProfitSharingReceiver
	ensureWechatProfitSharingReceiver = func(uid string) wechatReceiverStatus {
		called = uid
		return wechatReceiverStatus{Status: "pending_openid", Reason: "no_wechat_openid"}
	}
	t.Cleanup(func() { ensureWechatProfitSharingReceiver = orig })

	st, err := getReferralCodeStatus("user-status-wx")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if called != "user-status-wx" {
		t.Fatalf("ensure called with %q", called)
	}
	if st.WechatReceiverStatus != "pending_openid" {
		t.Fatalf("receiver status=%s", st.WechatReceiverStatus)
	}
}

func TestGetReferralCodeStatus_DoesNotEnsureWhenInactive(t *testing.T) {
	setupTestReferralDB(t)
	called := false
	orig := ensureWechatProfitSharingReceiver
	ensureWechatProfitSharingReceiver = func(string) wechatReceiverStatus {
		called = true
		return wechatReceiverStatus{Status: "registered"}
	}
	t.Cleanup(func() { ensureWechatProfitSharingReceiver = orig })

	if _, err := getReferralCodeStatus("user-none-wx"); err != nil {
		t.Fatalf("status: %v", err)
	}
	if called {
		t.Fatal("must not register WeChat receiver without qualification")
	}
}

func TestDeleteWechatProfitSharingReceiverLive_PostsUserID(t *testing.T) {
	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/taskbill/profit-sharing/receivers/delete/" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("json: %v", err)
		}
		gotUser = body["user_id"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"deleted","user_id":"` + gotUser + `"}`))
	}))
	t.Cleanup(srv.Close)

	prevURL, prevSec := cfg.BillServiceURL, cfg.BillInternalSecret
	cfg.BillServiceURL = srv.URL
	cfg.BillInternalSecret = "bill-secret"
	t.Cleanup(func() {
		cfg.BillServiceURL = prevURL
		cfg.BillInternalSecret = prevSec
	})

	got := deleteWechatProfitSharingReceiverLive("user-del-recv-1")
	if got.Status != "deleted" {
		t.Fatalf("got=%+v", got)
	}
	if gotUser != "user-del-recv-1" {
		t.Fatalf("posted user=%s", gotUser)
	}
}

func TestRevokeReferralQualification_DeletesWechatReceiver(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "user-revoke-del", time.Now().AddDate(0, 3, 0))
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "user-revoke-del").Scan(&appID); err != nil {
		t.Fatal(err)
	}

	var called string
	orig := deleteWechatProfitSharingReceiver
	deleteWechatProfitSharingReceiver = func(uid string) wechatReceiverStatus {
		called = uid
		return wechatReceiverStatus{Status: "deleted"}
	}
	t.Cleanup(func() { deleteWechatProfitSharingReceiver = orig })

	res, err := revokeReferralQualification(context.Background(), appID, "admin-1", "违反推荐政策取消资格", "ik-revoke-del")
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if res["status"] != "revoked" {
		t.Fatalf("status=%v", res["status"])
	}
	if called != "user-revoke-del" {
		t.Fatalf("delete receiver called with %q, want user-revoke-del", called)
	}
}

func TestRevokeReferralQualification_DeleteReceiverFailureDoesNotBlock(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "user-revoke-del-fail", time.Now().AddDate(0, 3, 0))
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "user-revoke-del-fail").Scan(&appID); err != nil {
		t.Fatal(err)
	}

	var called string
	orig := deleteWechatProfitSharingReceiver
	deleteWechatProfitSharingReceiver = func(uid string) wechatReceiverStatus {
		called = uid
		return wechatReceiverStatus{Status: "failed", Reason: "bill_unreachable"}
	}
	t.Cleanup(func() { deleteWechatProfitSharingReceiver = orig })

	res, err := revokeReferralQualification(context.Background(), appID, "admin-1", "违反推荐政策取消资格", "ik-revoke-del-fail")
	if err != nil {
		t.Fatalf("revoke must not be blocked by delete receiver failure: %v", err)
	}
	if res["status"] != "revoked" {
		t.Fatalf("status=%v", res["status"])
	}
	if called != "user-revoke-del-fail" {
		t.Fatalf("delete receiver called with %q", called)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM referral_code WHERE id=?`, appID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "revoked" {
		t.Fatalf("db status=%s", status)
	}
}
