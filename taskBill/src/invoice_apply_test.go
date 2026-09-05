package main

import (
	"context"
	"errors"
	"testing"
)

func insertPaidWechatOrderForInvoice(t *testing.T, tenantID, orderID, cents int64, captureID string) {
	t.Helper()
	now := utcNow()
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	ref := "WXINV" + formatID(orderID)
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', ?, 'wechat', ?, ?, ?)`,
		orderID, tenantID, "ORD-INV-"+formatID(orderID), cents, "wechat:"+ref, now, now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order_item (
			id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, created_at
		) VALUES (?, ?, 'task_post', 1, ?, ?, '', ?)`,
		generateSnowflakeID(), orderID, cents, cents, now,
	); err != nil {
		t.Fatalf("insert item: %v", err)
	}
	order, _, err := loadOrder(tenantID, orderID)
	if err != nil {
		t.Fatalf("load order: %v", err)
	}
	if _, err := ensureOrderPaymentLedger(context.Background(), db, tenantID, acc.ID, order, now); err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE billing_payment_ledger SET provider_capture_id = ?
		WHERE tenant_id = ? AND channel = 'wechat' AND provider_ref = ?`,
		captureID, tenantID, ref,
	); err != nil {
		t.Fatalf("capture: %v", err)
	}
}

func insertPaidWechatOrderRowOnly(t *testing.T, tenantID, orderID, cents int64) {
	t.Helper()
	now := utcNow()
	ref := "WXINV" + formatID(orderID)
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', ?, 'wechat', ?, ?, ?)`,
		orderID, tenantID, "ORD-INV-"+formatID(orderID), cents, "wechat:"+ref, now, now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}
}

func individualBuyer() invoiceBuyerInput {
	return invoiceBuyerInput{Type: buyerTypeIndividual, Name: "张三"}
}

func TestApplyInvoiceUnpaidRejected(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at
		) VALUES (?, ?, ?, 'pending', 55, 'wechat', 'wechat:WXU', ?)`,
		orderID, tenantID, "ORD-UNPAID", now,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if !errors.Is(err, ErrInvoiceOrderNotPaid) {
		t.Fatalf("err=%v want ErrInvoiceOrderNotPaid", err)
	}
}

func TestApplyInvoicePaidFirstAndIdempotentReplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV1")

	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "ik-1")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if app.Status != invoiceAppPending {
		t.Fatalf("status=%s", app.Status)
	}
	_, err = applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if !errors.Is(err, ErrInvoicePendingExists) {
		t.Fatalf("second without key err=%v", err)
	}
	replay, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "ik-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replay.ID != app.ID {
		t.Fatalf("replay id=%d want %d", replay.ID, app.ID)
	}
}

func TestApplyInvoiceOrgMissingTaxID(t *testing.T) {
	_, err := parseInvoiceBuyer(map[string]interface{}{
		"type": "ORGANIZATION",
		"name": "测试公司",
	})
	if !errors.Is(err, ErrInvoiceOrgTaxIDRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestApplyInvoiceZeroAmountRejected(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderRowOnly(t, tenantID, orderID, 0)
	_, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if !errors.Is(err, ErrInvoiceZeroAmount) {
		t.Fatalf("err=%v want ErrInvoiceZeroAmount", err)
	}
	var n int
	if qerr := db.QueryRow(`SELECT COUNT(*) FROM billing_invoice_application WHERE order_id = ?`, orderID).Scan(&n); qerr != nil {
		t.Fatalf("count: %v", qerr)
	}
	if n != 0 {
		t.Fatalf("applications=%d want 0", n)
	}
}

func TestApplyInvoiceZeroAmountRejectedEvenIfNotWechat(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', 0, 'admin_grant', '', ?, ?)`,
		orderID, tenantID, "ORD-GRANT-"+formatID(orderID), now, now,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if !errors.Is(err, ErrInvoiceZeroAmount) {
		t.Fatalf("err=%v want ErrInvoiceZeroAmount ( ahead of wechat-channel gate)", err)
	}
}

func TestApplyInvoiceOneCentAllowed(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 1, "420000INV1C")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if app.Status != invoiceAppPending {
		t.Fatalf("status=%s", app.Status)
	}
}

func TestApplyInvoiceWrongTenantNotFound(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	other := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV2")
	_, err := applyInvoiceApplication(context.Background(), other, orderID, "u1", individualBuyer(), "")
	if err == nil || err.Error() != "订单不存在" {
		t.Fatalf("err=%v want 订单不存在", err)
	}
}
