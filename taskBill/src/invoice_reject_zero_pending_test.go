package main

import (
	"context"
	"testing"
)

func TestRejectPendingZeroAmountInvoiceApplicationsClearsPending(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	zeroOrder := generateSnowflakeID()
	paidOrder := generateSnowflakeID()
	insertPaidWechatOrderRowOnly(t, tenantID, zeroOrder, 0)
	insertPaidWechatOrderForInvoice(t, tenantID, paidOrder, 55, "420000INVZ")
	now := utcNow()
	zeroApp := generateSnowflakeID()
	paidApp := generateSnowflakeID()
	for _, row := range []struct {
		appID   int64
		orderID int64
	}{
		{zeroApp, zeroOrder},
		{paidApp, paidOrder},
	} {
		if _, err := db.Exec(`
			INSERT INTO billing_invoice_application (
				id, tenant_id, order_id, applicant_user_id, invoice_type, buyer_type, buyer_name, taxpayer_id,
				address, telephone, bank_name, bank_account, status, reviewer_user_id, review_note,
				reviewed_at, created_at, updated_at
			) VALUES (?, ?, ?, 'u1', 'general', 'INDIVIDUAL', '张三', '', '', '', '', '', 'pending', '', '', NULL, ?, ?)`,
			row.appID, tenantID, row.orderID, now, now,
		); err != nil {
			t.Fatalf("insert pending app %d: %v", row.appID, err)
		}
	}

	ids, err := pendingZeroAmountInvoiceAppIDs(context.Background())
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if len(ids) != 1 || ids[0] != zeroApp {
		t.Fatalf("inventory ids=%v want [%d]", ids, zeroApp)
	}

	n, err := rejectPendingZeroAmountInvoiceApplications(context.Background(), "opt-20260829-008")
	if err != nil {
		t.Fatalf("reject batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("rejected=%d want 1", n)
	}

	ids, err = pendingZeroAmountInvoiceAppIDs(context.Background())
	if err != nil {
		t.Fatalf("inventory after: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("pending zero-amount after reject=%v want empty", ids)
	}

	app, err := loadInvoiceApplication(context.Background(), db, zeroApp)
	if err != nil {
		t.Fatalf("load zero app: %v", err)
	}
	if app.Status != invoiceAppRejected {
		t.Fatalf("zero app status=%q want %q", app.Status, invoiceAppRejected)
	}
	if app.ReviewNote != zeroAmountInvoiceRejectNote {
		t.Fatalf("note=%q want %q", app.ReviewNote, zeroAmountInvoiceRejectNote)
	}

	paid, err := loadInvoiceApplication(context.Background(), db, paidApp)
	if err != nil {
		t.Fatalf("load paid app: %v", err)
	}
	if paid.Status != invoiceAppPending {
		t.Fatalf("paid app status=%q want pending", paid.Status)
	}
}
