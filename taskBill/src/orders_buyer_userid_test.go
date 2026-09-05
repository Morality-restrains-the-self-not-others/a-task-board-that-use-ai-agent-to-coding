package main

import (
	"context"
	"testing"
)

// OPT-20260821-032: resource_purchase 交易流水必须写入真实买家 user_id，
// 禁止订单 user_id=0 时流水 user_id 为 NULL/空。

func TestCreateOrderPersistsBuyerUserID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000031
	const userID = "9300000032"

	order, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	}, "", userID)
	if err != nil {
		t.Fatalf("createOrderWithNote: %v", err)
	}

	// 订单创建即写入买家
	var gotUserID string
	if err := db.QueryRow(`SELECT user_id FROM billing_resource_order WHERE id = ?`, order.ID).Scan(&gotUserID); err != nil {
		t.Fatalf("load order user_id: %v", err)
	}
	if gotUserID != userID {
		t.Fatalf("order.user_id=%q, want %q", gotUserID, userID)
	}

	if err := markOrderPaid(context.Background(), order.ID, "wechat", "mock_ref", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}

	// 付款后交易流水 user_id 同步写入，禁止 resource_purchase 空 user_id
	var txnUserID string
	if err := db.QueryRow(`SELECT user_id FROM billing_transaction WHERE related_order_id = ? AND points_source_type = 'resource_purchase'`, order.ID).Scan(&txnUserID); err != nil {
		t.Fatalf("load txn user_id: %v", err)
	}
	if txnUserID != userID {
		t.Fatalf("transaction.user_id=%q, want %q", txnUserID, userID)
	}
}

func TestMarkOrderPaidBackfillsBuyerFromPending(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000033
	const userID = "9300000034"

	// 旧路径：订单创建未写 user_id（遗留订单的复现）
	order, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	}, "", "")
	if err != nil {
		t.Fatalf("createOrderWithNote: %v", err)
	}
	var gotUserID string
	if err := db.QueryRow(`SELECT user_id FROM billing_resource_order WHERE id = ?`, order.ID).Scan(&gotUserID); err != nil {
		t.Fatalf("load order user_id: %v", err)
	}
	if gotUserID != "0" && gotUserID != "" {
		t.Fatalf("fixture order.user_id=%q, want 0/empty", gotUserID)
	}

	// 支付会话已记录买家（旧支付流程写入 billing_payment_pending）
	outTradeNo := "WX" + formatID(generateSnowflakeID())
	if _, err := db.Exec(`
		INSERT INTO billing_payment_pending (out_trade_no, user_id, tenant_id, order_id, amount_fen, status, created_at)
		VALUES (?, ?, ?, ?, 100, 'pending', ?)`,
		outTradeNo, userID, tenantID, order.ID, utcNow()); err != nil {
		t.Fatalf("seed payment pending: %v", err)
	}

	if err := markOrderPaid(context.Background(), order.ID, "wechat", "mock_ref", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}

	var txnUserID string
	if err := db.QueryRow(`SELECT user_id FROM billing_transaction WHERE related_order_id = ? AND points_source_type = 'resource_purchase'`, order.ID).Scan(&txnUserID); err != nil {
		t.Fatalf("load txn user_id: %v", err)
	}
	if txnUserID != userID {
		t.Fatalf("transaction.user_id=%q, want %q", txnUserID, userID)
	}
	// 订单行也被回填，后续 loadOrder 可见买家
	var orderUserID string
	if err := db.QueryRow(`SELECT user_id FROM billing_resource_order WHERE id = ?`, order.ID).Scan(&orderUserID); err != nil {
		t.Fatalf("load order user_id: %v", err)
	}
	if orderUserID != userID {
		t.Fatalf("backfilled order.user_id=%q, want %q", orderUserID, userID)
	}
}
