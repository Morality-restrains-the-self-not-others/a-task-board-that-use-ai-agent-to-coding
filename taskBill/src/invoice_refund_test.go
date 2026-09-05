package main

import (
	"context"
	"os"
	"testing"
)

func TestReconcileInvoicesNoBlueSkipsReverse(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	reversed := 0
	orig := reverseWechatFapiaoFn
	reverseWechatFapiaoFn = func(ctx context.Context, req wechatFapiaoReverseRequest) error {
		reversed++
		return nil
	}
	t.Cleanup(func() { reverseWechatFapiaoFn = orig })

	orderID := generateSnowflakeID()
	if err := ReconcileInvoicesAfterRefund(context.Background(), orderID, 55); err != nil {
		t.Fatal(err)
	}
	if reversed != 0 {
		t.Fatalf("reverse called %d", reversed)
	}
}

func TestReconcileInvoicesFullRefundReverseNoReissue(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	stubWechatFapiaoOK(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV6")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin", "ok", ""); err != nil {
		t.Fatal(err)
	}
	issued := 0
	origI := issueWechatFapiaoFn
	issueWechatFapiaoFn = func(ctx context.Context, req wechatFapiaoIssueRequest) (wechatFapiaoIssueResult, error) {
		issued++
		return wechatFapiaoIssueResult{ApplyID: req.FapiaoApplyID, FapiaoID: req.FapiaoID}, nil
	}
	t.Cleanup(func() { issueWechatFapiaoFn = origI })

	if err := ReconcileInvoicesAfterRefund(context.Background(), orderID, 55); err != nil {
		t.Fatal(err)
	}
	if issued != 0 {
		t.Fatalf("reissue count=%d want 0", issued)
	}
	invoices, err := listInvoicesForOrder(context.Background(), orderID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, inv := range invoices {
		kinds[inv["kind"].(string)]++
		if inv["kind"] == invoiceKindBlue && inv["purpose"] == invoicePurposeOriginal {
			if inv["status"] != invoiceStatusReversePending {
				t.Fatalf("original blue status=%v want reverse_pending", inv["status"])
			}
		}
		if inv["kind"] == invoiceKindRed {
			if inv["status"] != invoiceStatusReversePending {
				t.Fatalf("red status=%v want reverse_pending", inv["status"])
			}
			if inv["reverse_confirm_hours"] != invoiceReverseConfirmHours {
				t.Fatalf("hours=%v", inv["reverse_confirm_hours"])
			}
			if inv["reverse_confirm_deadline"] == "" || inv["reverse_confirm_deadline"] == nil {
				t.Fatal("missing reverse_confirm_deadline")
			}
		}
	}
	if kinds[invoiceKindRed] != 1 {
		t.Fatalf("kinds=%v", kinds)
	}
	attached := map[string]interface{}{}
	attachOrderInvoices(attached, orderID)
	conf, ok := attached["invoice_reverse_confirm"].(map[string]interface{})
	if !ok {
		t.Fatal("missing invoice_reverse_confirm")
	}
	if conf["hours"] != invoiceReverseConfirmHours {
		t.Fatalf("confirm hours=%v", conf["hours"])
	}
	if conf["message"] != invoiceReverseConfirmHint {
		t.Fatalf("message=%v", conf["message"])
	}
}

func TestReconcileInvoicesPartialRefundReissues(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	stubWechatFapiaoOK(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 100, "420000INV7")
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_grant (
			id, tenant_id, resource_type, quantity, remaining, reason, created_at, source_kind, order_id
		) VALUES (?, ?, 'task_post', 2, 1, 'paid', ?, 'purchase', ?)`,
		generateSnowflakeID(), tenantID, now, orderID,
	); err != nil {
		t.Fatalf("grant: %v", err)
	}
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin", "ok", ""); err != nil {
		t.Fatal(err)
	}

	reissueAmt := int64(0)
	origI := issueWechatFapiaoFn
	issueWechatFapiaoFn = func(ctx context.Context, req wechatFapiaoIssueRequest) (wechatFapiaoIssueResult, error) {
		if req.Purpose == invoicePurposeReissue {
			reissueAmt = req.TotalAmountFen
		}
		return wechatFapiaoIssueResult{ApplyID: req.FapiaoApplyID, FapiaoID: req.FapiaoID}, nil
	}
	t.Cleanup(func() { issueWechatFapiaoFn = origI })

	unused, err := unusedRefundableCents(context.Background(), db, tenantID, orderID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if unused != 50 {
		t.Fatalf("unused=%d want 50", unused)
	}
	if err := ReconcileInvoicesAfterRefund(context.Background(), orderID, unused); err != nil {
		t.Fatal(err)
	}
	if reissueAmt != 50 {
		t.Fatalf("reissue=%d want 50", reissueAmt)
	}
}

func TestRefundOrderStillFullWhenNoConsumption(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	_ = os.Setenv("TASKBILL_REFUND_PROVIDER", "mock")
	t.Cleanup(func() { _ = os.Unsetenv("TASKBILL_REFUND_PROVIDER") })
	stubWechatFapiaoOK(t)

	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV8")
	app, err := applyRefundApplication(context.Background(), tenantID, "user-1", "体验不满意", orderID, "")
	if err != nil {
		t.Fatalf("apply refund: %v", err)
	}
	if app.FrozenPoints != 55 {
		t.Fatalf("frozen=%d want 55", app.FrozenPoints)
	}
}

func TestApplyWechatFapiaoNotifyMarksIssued(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	stubWechatFapiaoOK(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV9")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin", "ok", ""); err != nil {
		t.Fatal(err)
	}
	invoices, _ := listInvoicesForOrder(context.Background(), orderID)
	fapiaoID := invoices[0]["fapiao_id"].(string)
	payload := []byte(`{"event_type":"FAPIAO.ISSUED","resource":{"fapiao_apply_id":"420000INV9","fapiao_information":[{"fapiao_id":"` + fapiaoID + `","fapiao_number":"12345678","status":"ISSUED"}]}}`)
	if err := applyWechatFapiaoNotify(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	invoices, _ = listInvoicesForOrder(context.Background(), orderID)
	if invoices[0]["status"] != invoiceStatusIssued {
		t.Fatalf("status=%v", invoices[0]["status"])
	}
	if invoices[0]["wechat_fapiao_number"] != "12345678" {
		t.Fatalf("number=%v", invoices[0]["wechat_fapiao_number"])
	}
}

func TestApplyWechatFapiaoNotifyMarksReversed(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	stubWechatFapiaoOK(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV10")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin", "ok", ""); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileInvoicesAfterRefund(context.Background(), orderID, 55); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"event_type":"FAPIAO.REVERSED","resource":{"fapiao_apply_id":"420000INV10","fapiao_information":[{"fapiao_id":"x","status":"REVERSED"}]}}`)
	if err := applyWechatFapiaoNotify(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	invoices, err := listInvoicesForOrder(context.Background(), orderID)
	if err != nil {
		t.Fatal(err)
	}
	var sawBlueReversed, sawRedIssued bool
	for _, inv := range invoices {
		if inv["kind"] == invoiceKindBlue && inv["purpose"] == invoicePurposeOriginal {
			if inv["status"] != invoiceStatusReversed {
				t.Fatalf("blue status=%v", inv["status"])
			}
			sawBlueReversed = true
		}
		if inv["kind"] == invoiceKindRed {
			if inv["status"] != invoiceStatusIssued {
				t.Fatalf("red status=%v", inv["status"])
			}
			if _, ok := inv["reverse_confirm_deadline"]; ok {
				t.Fatalf("red still has confirm after REVERSED: %v", inv)
			}
			sawRedIssued = true
		}
	}
	if !sawBlueReversed || !sawRedIssued {
		t.Fatalf("sawBlue=%v sawRed=%v invoices=%v", sawBlueReversed, sawRedIssued, invoices)
	}
	attached := map[string]interface{}{}
	attachOrderInvoices(attached, orderID)
	if _, ok := attached["invoice_reverse_confirm"]; ok {
		t.Fatalf("confirm still attached after REVERSED: %v", attached["invoice_reverse_confirm"])
	}
}
