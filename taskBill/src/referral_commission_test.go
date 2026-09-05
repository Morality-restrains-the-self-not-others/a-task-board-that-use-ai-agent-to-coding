package main

import (
	"context"
	"testing"
	"time"
)

func setupReferralCommissionTestDB(t *testing.T) int64 {
	t.Helper()
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	tenantID := int64(850256677331562501)
	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}
	return tenantID
}

func TestReferralCommissionPointsHalfUp(t *testing.T) {
	if got := referralCommissionPoints(10000); got != 500 {
		t.Fatalf("10000 -> %d", got)
	}
	if got := referralCommissionPoints(10); got != 1 { // 0.5 -> 1 half-up
		t.Fatalf("10 -> %d", got)
	}
	if got := referralCommissionPoints(1); got != 0 {
		t.Fatalf("1 -> %d", got)
	}
}

func TestReferralAccrueSettleAndSummary(t *testing.T) {
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })
	tenantID := setupReferralCommissionTestDB(t)
	referrer := "ref-user-1"
	referred := "downline-user-1"
	boundAt := "2026-01-01 00:00:00.000000"
	if err := upsertReferralEdge(referrer, referred, tenantID, boundAt, "", true); err != nil {
		t.Fatalf("edge: %v", err)
	}

	consumedAt := "2026-01-10 12:00:00.000000"
	if err := accrueReferralFromConsumption(
		t.Context(), referred, "txn-consume-1", 1, 10000, consumedAt,
	); err != nil {
		t.Fatalf("accrue: %v", err)
	}

	sum, err := referralCommissionSummary(referrer)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum["pending_points"] != "500" {
		t.Fatalf("pending=%v", sum["pending_points"])
	}
	if sum["settled_points"] != "0" {
		t.Fatalf("settled=%v", sum["settled_points"])
	}

	// 未满 15 日
	n, err := settleDueReferralCommissions(t.Context(), time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("settle early: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 settled early, got %d", n)
	}

	// 满 15 日：1/10 + 15d = 1/25
	n, err = settleDueReferralCommissions(t.Context(), time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 settled, got %d", n)
	}

	sum, err = referralCommissionSummary(referrer)
	if err != nil {
		t.Fatalf("summary2: %v", err)
	}
	if sum["pending_points"] != "0" || sum["settled_points"] != "500" {
		t.Fatalf("after settle pending=%v settled=%v", sum["pending_points"], sum["settled_points"])
	}

	// 余额充值路径已移除（2026-07-26），推荐佣金改为发放任务帖配额
	// 不再通过 balance 验证，summary 中的 settled_points 已在上方断言
}

func TestReferralAccrueOutsideValidityWindow(t *testing.T) {
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })
	tenantID := setupReferralCommissionTestDB(t)
	referrer := "ref-user-2"
	referred := "downline-user-2"
	boundAt := "2025-01-01 00:00:00.000000"
	if err := upsertReferralEdge(referrer, referred, tenantID, boundAt, "", true); err != nil {
		t.Fatalf("edge: %v", err)
	}
	// 绑定超过一年
	if err := accrueReferralFromConsumption(
		t.Context(), referred, "txn-old-1", 2, 10000, "2026-02-01 00:00:00.000000",
	); err != nil {
		t.Fatalf("accrue: %v", err)
	}
	sum, err := referralCommissionSummary(referrer)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum["pending_points"] != "0" {
		t.Fatalf("pending should be 0, got %v", sum["pending_points"])
	}
}

func TestReferralVoidBeforeSettle(t *testing.T) {
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })
	tenantID := setupReferralCommissionTestDB(t)
	referrer := "ref-user-3"
	referred := "downline-user-3"
	if err := upsertReferralEdge(referrer, referred, tenantID, "2026-01-01 00:00:00.000000", "", true); err != nil {
		t.Fatalf("edge: %v", err)
	}
	if err := accrueReferralFromConsumption(
		t.Context(), referred, "txn-void-1", 3, 10000, "2026-01-10 00:00:00.000000",
	); err != nil {
		t.Fatalf("accrue: %v", err)
	}
	if err := voidReferralAccrualBySourceTxn("txn-void-1", "refund"); err != nil {
		t.Fatalf("void: %v", err)
	}
	n, err := settleDueReferralCommissions(t.Context(), time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	if n != 0 {
		t.Fatalf("voided should not settle, got %d", n)
	}
	sum, _ := referralCommissionSummary(referrer)
	if sum["pending_points"] != "0" || sum["settled_points"] != "0" {
		t.Fatalf("summary after void: %v", sum)
	}
}

func TestReferralBackfillHistoricalConsumption(t *testing.T) {
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })
	tenantID := setupReferralCommissionTestDB(t)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	referrer := "ref-late"
	referred := "downline-late"
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			user_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', 10000, 10000, 0, ?, 1, 'hist', 'txn-hist-late', ?)
	`, generateSnowflakeID(), acc.ID, referred, now); err != nil {
		t.Fatalf("insert txn: %v", err)
	}
	if err := upsertReferralEdge(referrer, referred, tenantID, "2026-01-01 00:00:00.000000", "", true); err != nil {
		t.Fatalf("edge: %v", err)
	}
	n, err := backfillReferralAccrualsForReferredUser(t.Context(), referred)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if n < 1 {
		t.Fatalf("backfilled %d want >=1", n)
	}
	sum, err := referralCommissionSummary(referrer)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum["pending_points"] != "500" {
		t.Fatalf("pending=%v want 500", sum["pending_points"])
	}
}

func TestReferralAccrueWhenSnapshotInactiveButPaytimeActive(t *testing.T) {
	// 快照 commission_eligible=0 + 现查活跃 → 仍计提（ADR-0033 支付时刻资格口径）
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool {
		return referrerUserID == "ref-late-qual"
	})
	tenantID := setupReferralCommissionTestDB(t)
	if err := upsertReferralEdge("ref-late-qual", "down-late-qual", tenantID, "2026-01-01 00:00:00.000000", "WXCODE", false); err != nil {
		t.Fatalf("edge: %v", err)
	}
	if err := accrueReferralFromConsumption(
		t.Context(), "down-late-qual", "txn-late-qual-1", 9, 10000, "2026-01-10 00:00:00.000000",
	); err != nil {
		t.Fatalf("accrue: %v", err)
	}
	sum, err := referralCommissionSummary("ref-late-qual")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum["pending_points"] != "500" {
		t.Fatalf("snapshot-inactive + paytime-active must accrue, pending=%v", sum["pending_points"])
	}
}

func TestReferralAccrueSkippedWhenPaytimeInactive(t *testing.T) {
	// 快照 commission_eligible=1 + 现查不活跃 → 不计提
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return false })
	tenantID := setupReferralCommissionTestDB(t)
	if err := upsertReferralEdge("ref-noqual", "down-noqual", tenantID, "2026-01-01 00:00:00.000000", "WXCODE", true); err != nil {
		t.Fatalf("edge: %v", err)
	}
	if err := accrueReferralFromConsumption(
		t.Context(), "down-noqual", "txn-noqual-1", 9, 10000, "2026-01-10 00:00:00.000000",
	); err != nil {
		t.Fatalf("accrue: %v", err)
	}
	sum, err := referralCommissionSummary("ref-noqual")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum["pending_points"] != "0" {
		t.Fatalf("paytime-inactive must not accrue, pending=%v", sum["pending_points"])
	}
}

func TestReferralEdgeSnapshotNotOverwritten(t *testing.T) {
	tenantID := setupReferralCommissionTestDB(t)
	if err := upsertReferralEdge("r1", "d1", tenantID, "2026-01-01 00:00:00.000000", "CODEA", true); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := upsertReferralEdge("r1", "d1", tenantID, "2026-02-01 00:00:00.000000", "CODEB", false); err != nil {
		t.Fatalf("second: %v", err)
	}
	var code string
	var eligible int
	if err := db.QueryRow(`SELECT channel_code, commission_eligible FROM billing_referral_edge WHERE referred_user_id=?`, "d1").Scan(&code, &eligible); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if code != "CODEA" || eligible != 1 {
		t.Fatalf("snapshot overwritten code=%s eligible=%d", code, eligible)
	}
}
