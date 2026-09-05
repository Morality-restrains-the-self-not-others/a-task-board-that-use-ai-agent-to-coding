package main

import (
	"context"
	"testing"
	"time"
)

func TestDisableReferrerCommissionEligibility(t *testing.T) {
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })
	tenantID := setupReferralCommissionTestDB(t)
	referrer := "ref-disable-1"
	referred := "down-disable-1"
	if err := upsertReferralEdge(referrer, referred, tenantID, "2026-01-01 00:00:00.000000", "", true); err != nil {
		t.Fatal(err)
	}
	if err := accrueReferralFromConsumption(
		t.Context(), referred, "txn-disable-1", 11, 10000, "2026-01-10 12:00:00.000000",
	); err != nil {
		t.Fatal(err)
	}
	// settle one copy by advancing clock
	if _, err := settleDueReferralCommissions(t.Context(), time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := upsertReferralEdge(referrer, "down-disable-2", tenantID, "2026-01-02 00:00:00.000000", "", true); err != nil {
		t.Fatal(err)
	}
	if err := accrueReferralFromConsumption(
		t.Context(), "down-disable-2", "txn-disable-2", 12, 10000, "2026-01-11 12:00:00.000000",
	); err != nil {
		t.Fatal(err)
	}

	edges, voided, err := disableReferrerCommissionEligibility(referrer, "qualification_revoked")
	if err != nil {
		t.Fatal(err)
	}
	if edges < 1 {
		t.Fatalf("expected edges updated, got %d", edges)
	}
	if voided != 1 {
		t.Fatalf("expected 1 pending voided, got %d", voided)
	}

	var eligible int
	if err := db.QueryRow(`SELECT commission_eligible FROM billing_referral_edge WHERE referred_user_id=?`, referred).Scan(&eligible); err != nil {
		t.Fatal(err)
	}
	if eligible != 0 {
		t.Fatalf("edge still eligible=%d", eligible)
	}

	sum, err := referralCommissionSummary(referrer)
	if err != nil {
		t.Fatal(err)
	}
	if sum["pending_points"] != "0" {
		t.Fatalf("pending should be 0 after void, got %v", sum["pending_points"])
	}
	if sum["settled_points"] != "500" {
		t.Fatalf("settled must remain, got %v", sum["settled_points"])
	}

	edges2, voided2, err := disableReferrerCommissionEligibility(referrer, "qualification_revoked")
	if err != nil {
		t.Fatal(err)
	}
	if edges2 != 0 || voided2 != 0 {
		t.Fatalf("replay should no-op edges=%d voided=%d", edges2, voided2)
	}
}
