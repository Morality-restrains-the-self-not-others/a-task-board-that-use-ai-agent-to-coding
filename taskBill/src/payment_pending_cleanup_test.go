package main

import (
	"context"
	"testing"
)

// insertPendingForOrder seeds a billing_payment_pending row for a given order.
func insertPendingForOrder(t *testing.T, orderID int64, outTradeNo, status string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO billing_payment_pending (
			out_trade_no, user_id, tenant_id, order_id, amount_fen, status, created_at
		) VALUES (?, ?, ?, ?, 55, ?, ?)`,
		outTradeNo, generateSnowflakeID(), 0, orderID, status, utcNow(),
	)
	if err != nil {
		t.Fatalf("insert billing_payment_pending: %v", err)
	}
}

func pendingStatusForOrder(t *testing.T, orderID int64, outTradeNo string) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM billing_payment_pending WHERE order_id = ? AND out_trade_no = ?`, orderID, outTradeNo).Scan(&status); err != nil {
		t.Fatalf("query pending: %v", err)
	}
	return status
}

// OPT-20260820-018: markOrderRefunded 后同订单的 paid pending 行须变 refunded、
// 未支付行须变 cancelled，不得残留 pending。
func TestMarkOrderRefundedClosesPaymentPending(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	now := utcNow()
	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, 0, 'ORD-PENDING-CLEANUP', 'paid', 55, 'wechat', 'wechat:PENDING_CLEANUP', ?, ?)`,
		orderID, now, now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	insertPendingForOrder(t, orderID, "OUT_PAID", "paid")
	insertPendingForOrder(t, orderID, "OUT_PENDING", "pending")

	if err := markOrderRefunded(context.Background(), db, orderID); err != nil {
		t.Fatalf("markOrderRefunded: %v", err)
	}
	if got := pendingStatusForOrder(t, orderID, "OUT_PAID"); got != "refunded" {
		t.Fatalf("paid pending row status=%q want refunded", got)
	}
	if got := pendingStatusForOrder(t, orderID, "OUT_PENDING"); got != "cancelled" {
		t.Fatalf("pending row status=%q want cancelled", got)
	}
}

// OPT-20260820-018: 取消订单路径经 closePaymentPendingForOrder 把 pending 行标 cancelled，
// 已 paid 行保持 paid（不应被误取消）。
func TestClosePaymentPendingForOrderCancel(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	now := utcNow()
	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, 0, 'ORD-PENDING-CANCEL', 'pending', 55, 'wechat', '', ?, ?)`,
		orderID, now, now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	insertPendingForOrder(t, orderID, "OUT_CANCEL", "pending")

	if err := closePaymentPendingForOrder(context.Background(), db, orderID, "paid", "cancelled"); err != nil {
		t.Fatalf("closePaymentPendingForOrder: %v", err)
	}
	if got := pendingStatusForOrder(t, orderID, "OUT_CANCEL"); got != "cancelled" {
		t.Fatalf("pending row status=%q want cancelled", got)
	}
}
