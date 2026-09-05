package main

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestApplyRefundApplicationOrderDirectPayZeroBalance(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	_ = os.Setenv("TASKBILL_REFUND_PROVIDER", "mock")
	t.Cleanup(func() { _ = os.Unsetenv("TASKBILL_REFUND_PROVIDER") })

	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	itemID := generateSnowflakeID()
	now := utcNow()

	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	if acc.Balance != 0 {
		t.Fatalf("expected zero balance, got %d", acc.Balance)
	}

	_, err = db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', 55, 'wechat', 'wechat:WX_TEST_REFUND_1', ?, ?)`,
		orderID, tenantID, "ORD-REFUND-TEST-1", now, now,
	)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO billing_resource_order_item (
			id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, created_at
		) VALUES (?, ?, 'task_post', 1, 55, 55, '', ?)`,
		itemID, orderID, now,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	_, err = db.Exec(`UPDATE billing_account SET task_post_quota = 1, updated_at = ? WHERE id = ?`, now, acc.ID)
	if err != nil {
		t.Fatalf("set quota: %v", err)
	}

	app, err := applyRefundApplication(context.Background(), tenantID, "user-1", "体验不满意", orderID, "")
	if err != nil {
		t.Fatalf("applyRefundApplication: %v", err)
	}
	if app.FrozenPoints != 55 {
		t.Fatalf("frozen_points=%d want 55", app.FrozenPoints)
	}
	if !app.OrderID.Valid || app.OrderID.Int64 != orderID {
		t.Fatalf("order_id=%v", app.OrderID)
	}

	var balance, frozen int64
	if err := db.QueryRow(`SELECT balance, frozen_balance FROM billing_account WHERE id = ?`, acc.ID).
		Scan(&balance, &frozen); err != nil {
		t.Fatalf("reload account: %v", err)
	}
	if balance != 0 || frozen != 0 {
		t.Fatalf("order-direct refund must not freeze balance: balance=%d frozen=%d", balance, frozen)
	}

	var ledgerRemaining int64
	if err := db.QueryRow(`
		SELECT remaining_points FROM billing_payment_ledger
		WHERE tenant_id = ? AND provider_ref = 'WX_TEST_REFUND_1'`, tenantID,
	).Scan(&ledgerRemaining); err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if ledgerRemaining != 55 {
		t.Fatalf("ledger remaining=%d want 55", ledgerRemaining)
	}

	approved, err := approveRefundApplication(context.Background(), app.ID, "admin-1", "ok")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != refundStatusApproved {
		t.Fatalf("status=%s", approved.Status)
	}

	var orderStatus string
	if err := db.QueryRow(`SELECT status FROM billing_resource_order WHERE id = ?`, orderID).Scan(&orderStatus); err != nil {
		t.Fatalf("order status: %v", err)
	}
	if orderStatus != OrderStatusRefunded {
		t.Fatalf("order status=%s want refunded", orderStatus)
	}

	var quota int64
	if err := db.QueryRow(`SELECT task_post_quota FROM billing_account WHERE id = ?`, acc.ID).Scan(&quota); err != nil {
		t.Fatalf("quota: %v", err)
	}
	if quota != 0 {
		t.Fatalf("quota after refund=%d want 0", quota)
	}

	_, err = applyRefundApplication(context.Background(), tenantID, "user-1", "again", orderID, "")
	if !errors.Is(err, ErrOrderAlreadyRefunded) && !errors.Is(err, ErrOrderNotRefundable) {
		t.Fatalf("second apply err=%v", err)
	}
}

func TestApplyRefundApplicationAdminGrantRejected(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	now := utcNow()
	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', 0, 'admin_grant', '', ?, ?)`,
		orderID, tenantID, "ORD-GRANT-1", now, now,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err = applyRefundApplication(context.Background(), tenantID, "u", "want refund", orderID, "")
	if !errors.Is(err, ErrOrderNotRefundable) && !errors.Is(err, ErrNoRefundableAmount) {
		t.Fatalf("want not refundable, got %v", err)
	}
}

func TestOrderRefundChannel(t *testing.T) {
	ch, ref, ok := orderRefundChannel("wechat", "wechat:WX123")
	if !ok || ch != "wechat" || ref != "WX123" {
		t.Fatalf("got %s %s %v", ch, ref, ok)
	}
	_, _, ok = orderRefundChannel("admin_grant", "")
	if ok {
		t.Fatal("admin_grant must not be refundable channel")
	}
}

func TestApplyRefundApplicationIdempotentReplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	now := utcNow()
	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', 55, 'wechat', 'wechat:WX_REPLAY_1', ?, ?)`,
		orderID, tenantID, "ORD-REPLAY-1", now, now,
	)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	first, err := applyRefundApplication(context.Background(), tenantID, "user-1", "timeout retry", orderID, "ik-refund-1")
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	second, err := applyRefundApplication(context.Background(), tenantID, "user-1", "timeout retry", orderID, "ik-refund-1")
	if err != nil {
		t.Fatalf("replay apply: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("replay id=%d want %d", second.ID, first.ID)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_refund_application WHERE tenant_id = ?`, tenantID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("applications=%d want 1", n)
	}
}
