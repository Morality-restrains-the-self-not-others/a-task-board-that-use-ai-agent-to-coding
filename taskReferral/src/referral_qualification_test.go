package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testReviewReason = "申请人介绍充分予以通过"

func TestValidateReviewReason(t *testing.T) {
	if _, err := validateReviewReason("短"); !errors.Is(err, errReferralReasonInvalid) {
		t.Fatalf("short reason: %v", err)
	}
	if _, err := validateReviewReason(""); !errors.Is(err, errReferralReasonInvalid) {
		t.Fatalf("empty reason: %v", err)
	}
	got, err := validateReviewReason("  " + testReviewReason + "  ")
	if err != nil {
		t.Fatalf("valid reason: %v", err)
	}
	if got != testReviewReason {
		t.Fatalf("trimmed=%q", got)
	}
}

func TestApproveRejectRequireReasonAndWriteAudit(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")
	if _, err := applyReferralWithConsent("user-audit-a", testValidPersonalIntro); err != nil {
		t.Fatal(err)
	}
	app := getPendingReferralCode("user-audit-a")
	if _, err := approveReferralApplication(context.Background(), app.ID, "admin-1", "短", ""); !errors.Is(err, errReferralReasonInvalid) {
		t.Fatalf("approve short: %v", err)
	}
	if _, err := approveReferralApplication(context.Background(), app.ID, "admin-1", testReviewReason, ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	items, total, err := listQualificationAudit(app.ID, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || items[0].Action != "approve" || items[0].Reason != testReviewReason {
		t.Fatalf("audit=%+v total=%d", items, total)
	}
}

func TestRevokeReferralQualification(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "user-revoke-1", time.Now().AddDate(0, 3, 0))
	var appID int
	if err := db.QueryRow(`SELECT id FROM referral_code WHERE user_id=?`, "user-revoke-1").Scan(&appID); err != nil {
		t.Fatal(err)
	}

	if _, err := revokeReferralQualification(context.Background(), appID, "admin-1", "短理由", "ik-revoke-1"); !errors.Is(err, errReferralReasonInvalid) {
		t.Fatalf("short: %v", err)
	}

	res, err := revokeReferralQualification(context.Background(), appID, "admin-1", "违反推荐政策取消资格", "ik-revoke-1")
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if res["status"] != "revoked" {
		t.Fatalf("status=%v", res["status"])
	}

	res2, err := revokeReferralQualification(context.Background(), appID, "admin-1", "违反推荐政策取消资格", "ik-revoke-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if res2["message"] != "already_revoked" {
		t.Fatalf("replay message=%v", res2["message"])
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM referral_code WHERE id=?`, appID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "revoked" {
		t.Fatalf("db status=%s", status)
	}
	if getActiveReferralCode("user-revoke-1") != nil {
		t.Fatal("active qualification should be gone")
	}
	items, total, err := listQualificationAudit(appID, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || items[0].Action != "revoke" {
		t.Fatalf("expected 1 revoke audit, got total=%d items=%+v", total, items)
	}

	st, err := getReferralCodeStatus("user-revoke-1")
	if err != nil {
		t.Fatal(err)
	}
	if !st.CanApply {
		t.Fatal("revoked user should be able to reapply immediately")
	}
}

func TestRevokePendingNotAllowed(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")
	if _, err := applyReferralWithConsent("user-revoke-pending", testValidPersonalIntro); err != nil {
		t.Fatal(err)
	}
	app := getPendingReferralCode("user-revoke-pending")
	if _, err := revokeReferralQualification(context.Background(), app.ID, "admin-1", "违反推荐政策取消资格", ""); !errors.Is(err, errReferralAppNotRevocable) {
		t.Fatalf("pending revoke: %v", err)
	}
}

func TestListApplicationsCanRevoke(t *testing.T) {
	setupTestReferralDB(t)
	insertApprovedReferral(t, "user-list-active", time.Now().AddDate(0, 3, 0))
	items, _, err := listReferralApplications("approved", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].CanRevoke || !items[0].IsActive {
		t.Fatalf("expected revocable approved row, got %+v", items)
	}
	if items[0].StatusDisplay != "已通过" {
		t.Fatalf("display=%s", items[0].StatusDisplay)
	}
}
