package main

import (
	"context"
	"strings"
	"testing"
)

// OPT-20260821-035: 分账请求必须用真实微信支付单号（回调回写 provider_capture_id），
// 禁止 "wechat:ORD...TODO" 假值；找不到单号则失败，不静默当成功。

func seedProfitSharingOrderWithLedger(t *testing.T, tenantID, orderID int64, paymentRef, txnCaptureID string) {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, payment_method, payment_ref, created_at)
		VALUES (?, ?, ?, 'paid', 1000, 'wechat', ?, ?)`,
		orderID, tenantID, "ORD"+formatID(orderID), paymentRef, now); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at) VALUES (?, ?, 0, ?, ?)`,
		generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_payment_ledger (id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
			points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at)
		VALUES (?, ?, (SELECT id FROM billing_account WHERE tenant_id = ?), 'wechat', ?, ?, 1000, 1000, 1000, 'CNY', NULL, ?, ?)`,
		generateSnowflakeID(), tenantID, tenantID, strings.TrimPrefix(paymentRef, "wechat:"), txnCaptureID, now, now); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}
}

func seedProfitSharingRecord(t *testing.T, orderID, tenantID int64) int64 {
	t.Helper()
	now := utcNow()
	outNo := "PS" + formatID(generateSnowflakeID())
	if _, err := db.Exec(`
		INSERT INTO billing_profit_sharing (out_profit_sharing_no, order_id, order_number, tenant_id,
			referrer_user_id, referrer_openid, total_yuan_cents, commission_yuan_cents, status, settle_after, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'referrer-1', 'openid-1', 1000, 50, 'pending', ?, ?, ?)`,
		outNo, orderID, "ORD"+formatID(orderID), tenantID, now, now, now); err != nil {
		t.Fatalf("seed profit sharing: %v", err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM billing_profit_sharing WHERE out_profit_sharing_no = ?`, outNo).Scan(&id); err != nil {
		t.Fatalf("load ps id: %v", err)
	}
	return id
}

func TestLookupWechatTransactionIDFromLedger(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 94001
	orderID := int64(2000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX123", "42000011112222")

	got, err := lookupWechatTransactionIDForOrder(orderID)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got != "42000011112222" {
		t.Fatalf("txn id=%q want 42000011112222", got)
	}
}

func TestExecuteProfitSharingUsesRealTransactionID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 94002
	orderID := int64(2000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX456", "42000033334444")
	psID := seedProfitSharingRecord(t, orderID, tenantID)

	var capturedTxnID string
	orig := createProfitSharingOrder
	createProfitSharingOrder = func(_ context.Context, _ string, wechatTransactionID string, _ string, _ []profitSharingReceiver) (string, error) {
		capturedTxnID = wechatTransactionID
		return "wx-ps-order-1", nil
	}
	t.Cleanup(func() { createProfitSharingOrder = orig })

	record := profitSharingRecord{
		ID:                  psID,
		OutProfitSharingNo:  "PS" + formatID(psID),
		OrderID:             orderID,
		OrderNumber:         "ORD" + formatID(orderID),
		TenantID:            tenantID,
		ReferrerOpenid:      "openid-1",
		CommissionYuanCents: 50,
	}
	if err := executeProfitSharing(context.Background(), record); err != nil {
		t.Fatalf("executeProfitSharing: %v", err)
	}
	if capturedTxnID != "42000033334444" {
		t.Fatalf("createProfitSharingOrder txn id=%q want 42000033334444", capturedTxnID)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&status); err != nil {
		t.Fatalf("load status: %v", err)
	}
	if status != psStatusFinished {
		t.Fatalf("status=%q want finished", status)
	}
}

func TestExecuteProfitSharingFailsWithoutTransactionID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 94003
	orderID := int64(2000 + tenantID)
	// 旧支付未回写真实单号：provider_capture_id 为空
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX789", "")
	psID := seedProfitSharingRecord(t, orderID, tenantID)

	called := false
	orig := createProfitSharingOrder
	createProfitSharingOrder = func(_ context.Context, _, _, _ string, _ []profitSharingReceiver) (string, error) {
		called = true
		return "", nil
	}
	t.Cleanup(func() { createProfitSharingOrder = orig })

	record := profitSharingRecord{
		ID:                 psID,
		OutProfitSharingNo: "PS" + formatID(psID),
		OrderID:            orderID,
		OrderNumber:        "ORD" + formatID(orderID),
		TenantID:           tenantID,
		ReferrerOpenid:     "openid-1",
	}
	if err := executeProfitSharing(context.Background(), record); err == nil {
		t.Fatal("expected error when no transaction_id")
	}
	if called {
		t.Fatal("createProfitSharingOrder should not be called without real transaction_id")
	}
}
