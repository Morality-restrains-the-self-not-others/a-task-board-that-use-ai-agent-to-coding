package main

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func seedPaidOrderForProfitSharing(t *testing.T, orderTenant, orderID, buyerUserID int64, totalCents int64) {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, user_id, created_at, paid_at)
		VALUES (?, ?, ?, 'paid', ?, ?, ?, ?)`,
		orderID, orderTenant, "ORD-"+formatID(orderID), totalCents, buyerUserID, now, now); err != nil {
		t.Fatalf("seed order: %v", err)
	}
}

func TestMarkOrderForProfitSharingRecordsWhenOrderTenantDiffers(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })

	referrerTenant := generateSnowflakeID()
	orderTenant := generateSnowflakeID()
	orderID := generateSnowflakeID()
	buyerID := generateSnowflakeID()
	referrerID := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, orderTenant, orderID, buyerID, 55)
	if err := upsertReferralEdge(formatID(referrerID), formatID(buyerID), referrerTenant, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}

	if err := markOrderForProfitSharing(context.Background(), orderID, orderTenant); err != nil {
		t.Fatalf("mark: %v", err)
	}

	var receiver string
	var cents int64
	if err := db.QueryRow(`
		SELECT referrer_user_id, commission_yuan_cents FROM billing_profit_sharing WHERE order_id = ?`,
		orderID).Scan(&receiver, &cents); err != nil {
		t.Fatalf("load row: %v", err)
	}
	if receiver != formatID(referrerID) {
		t.Fatalf("receiver=%s want %s", receiver, formatID(referrerID))
	}
	if cents != 3 {
		t.Fatalf("commission cents=%d want 3 (5%% of 55)", cents)
	}

	if err := markOrderForProfitSharing(context.Background(), orderID, orderTenant); err != nil {
		t.Fatalf("idempotent mark: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 row after replay, got %d", n)
	}
}

func TestMarkOrderForProfitSharingRecordsWithoutOpenid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })

	referrerTenant := generateSnowflakeID()
	orderTenant := generateSnowflakeID()
	orderID := generateSnowflakeID()
	buyerID := generateSnowflakeID()
	referrerID := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, orderTenant, orderID, buyerID, 1000)
	if err := upsertReferralEdge(formatID(referrerID), formatID(buyerID), referrerTenant, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}

	if err := markOrderForProfitSharing(context.Background(), orderID, orderTenant); err != nil {
		t.Fatalf("mark: %v", err)
	}
	var openid string
	if err := db.QueryRow(`SELECT referrer_openid FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&openid); err != nil {
		t.Fatalf("load: %v", err)
	}
	if openid != "" {
		t.Fatalf("openid=%q want empty", openid)
	}
}

func TestMarkOrderForProfitSharingSkipsIneligibleEdge(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return false })

	referrerTenant := generateSnowflakeID()
	orderTenant := generateSnowflakeID()
	orderID := generateSnowflakeID()
	buyerID := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, orderTenant, orderID, buyerID, 55)
	if err := upsertReferralEdge("r1", formatID(buyerID), referrerTenant, utcNow(), "CH", false); err != nil {
		t.Fatal(err)
	}
	if err := markOrderForProfitSharing(context.Background(), orderID, orderTenant); err != nil {
		t.Fatalf("mark: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("ineligible edge should not record, got %d", n)
	}
}

func TestMarkOrderForProfitSharingRecordsWhenPaytimeQualifiedDespiteIneligibleSnapshot(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })

	referrerTenant := generateSnowflakeID()
	orderTenant := generateSnowflakeID()
	orderID := generateSnowflakeID()
	buyerID := generateSnowflakeID()
	referrerID := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, orderTenant, orderID, buyerID, 1000)
	if err := upsertReferralEdge(formatID(referrerID), formatID(buyerID), referrerTenant, utcNow(), "CH", false); err != nil {
		t.Fatal(err)
	}
	if err := markOrderForProfitSharing(context.Background(), orderID, orderTenant); err != nil {
		t.Fatalf("mark: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("paytime qualified must record despite bind snapshot 0, got %d", n)
	}
}

func TestMarkOrderForProfitSharingSkipsWhenPaytimeUnqualifiedEvenIfSnapshotEligible(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return false })

	referrerTenant := generateSnowflakeID()
	orderTenant := generateSnowflakeID()
	orderID := generateSnowflakeID()
	buyerID := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, orderTenant, orderID, buyerID, 1000)
	if err := upsertReferralEdge("r-revoked", formatID(buyerID), referrerTenant, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	if err := markOrderForProfitSharing(context.Background(), orderID, orderTenant); err != nil {
		t.Fatalf("mark: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("paytime unqualified must not record, got %d", n)
	}
}

func TestMarkOrderForProfitSharingNilDBSkips(t *testing.T) {
	// nightly 060001 复现：markOrderPaid 以异步 goroutine 调用 markOrderForProfitSharing，
	// 测试 cleanup 把全局 db 置 nil 后 goroutine 才执行，导致 db.QueryRow nil 解引用 panic。
	// 修复：函数入口捕获一次 db 并判空，nil 时静默跳过分账标记（与 getReferralConfig 等 nil 兜底一致）。
	saved := db
	db = nil
	defer func() { db = saved }()

	if err := markOrderForProfitSharing(context.Background(), 1, 1); err != nil {
		t.Fatalf("nil db should skip silently, got err=%v", err)
	}
}

func TestFindReferrerForOrderSQLForcesUnicodeCollate(t *testing.T) {
	src, err := os.ReadFile("wechat_profit_sharing.go")
	if err != nil {
		t.Fatal(err)
	}
	const needle = "CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci"
	if !bytes.Contains(src, []byte(needle)) {
		t.Fatalf("findReferrerForOrder JOIN must coerce CAST collation; missing %q", needle)
	}
}
