package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// OPT-20260819-004: 回调入账前核对 pending.amount_fen 与订单 total_yuan_cents。
// 旧二维码（amount_fen=100）扫到 55 分订单时必须拒绝入账，且订单保持 pending。

func TestWechatCreditFromPendingAmountMismatch(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = 9000000030
	const userID = "9000000031"
	const outTradeNo = "ORD-20260818-003-mismatch"

	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at, user_id)
		VALUES (?, ?, ?, 'pending', 55, ?, ?)`,
		orderID, tenantID, "ORD-20260818-003", utcNow(), userID,
	); err != nil {
		t.Fatalf("seed order: %v", err)
	}

	storeWechatPending(wechatPendingOrder{
		OutTradeNo: outTradeNo,
		TenantID:   tenantID,
		UserID:     userID,
		AmountYuan: 1,
		AmountFen:  100,
		OrderID:    orderID,
		CreatedAt:  time.Now(),
	})

	_, err := wechatCreditFromPending(context.Background(), outTradeNo, 100)
	if err == nil {
		t.Fatal("expected amount mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "mismatch") && !strings.Contains(err.Error(), "!=") {
		t.Fatalf("error = %v, want amount mismatch", err)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM billing_resource_order WHERE id = ?`, orderID).Scan(&status); err != nil {
		t.Fatalf("query order status: %v", err)
	}
	if status != "pending" {
		t.Fatalf("order status = %q, want pending (拒绝入账)", status)
	}
}

func TestOutTradeNoFingerprintStable(t *testing.T) {
	a := outTradeNoFingerprint("ORD-20260818-003")
	b := outTradeNoFingerprint("ORD-20260818-003")
	if a == "" || a != b {
		t.Fatalf("fingerprint %q not stable", a)
	}
	if len(a) != 8 {
		t.Fatalf("fingerprint length = %d, want 8", len(a))
	}
}
