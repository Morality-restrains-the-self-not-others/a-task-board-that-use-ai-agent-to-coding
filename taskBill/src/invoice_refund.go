package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

func unusedRefundableCents(ctx context.Context, exec sqlExecutor, tenantID, orderID, totalYuanCents int64) (int64, error) {
	var granted, remaining int64
	err := exec.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quantity), 0), COALESCE(SUM(remaining), 0)
		FROM billing_resource_grant
		WHERE tenant_id = ? AND order_id = ? AND resource_type = ?`,
		tenantID, orderID, ResourceTypeTaskPost,
	).Scan(&granted, &remaining)
	if err != nil {
		return 0, err
	}
	if granted <= 0 {
		return totalYuanCents, nil
	}
	if remaining <= 0 {
		return 0, nil
	}
	if remaining >= granted {
		return totalYuanCents, nil
	}
	return totalYuanCents * remaining / granted, nil
}

func remainingDealCentsAfterRefund(orderTotal, refundAmount int64) int64 {
	left := orderTotal - refundAmount
	if left < 0 {
		return 0
	}
	return left
}

func cancelPendingInvoiceApplicationsForOrder(ctx context.Context, orderID int64) {
	now := utcNow()
	res, err := db.ExecContext(ctx, `
		UPDATE billing_invoice_application SET status = ?, updated_at = ?
		WHERE order_id = ? AND status = ?`,
		invoiceAppCancelled, now, orderID, invoiceAppPending)
	if err != nil {
		slog.WarnContext(ctx, "invoice_pending_cancel_failed",
			"level", "warn",
			"order_id", formatID(orderID),
			"error", err.Error(),
		)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		slog.InfoContext(ctx, "invoice_pending_cancelled_on_refund",
			"level", "info",
			"order_id", formatID(orderID),
			"count", n,
		)
	}
}

func loadIssuedBlueInvoices(ctx context.Context, orderID int64) ([]Invoice, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, order_id, application_id, related_invoice_id, kind, purpose, status,
		       amount_yuan_cents, fapiao_id, wechat_apply_id, wechat_fapiao_number, buyer_snapshot,
		       fail_reason, created_at, updated_at
		FROM billing_invoice
		WHERE order_id = ? AND kind = ? AND status IN (?, ?)
		ORDER BY id ASC`, orderID, invoiceKindBlue, invoiceStatusIssued, invoiceStatusIssuing)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.OrderID, &inv.ApplicationID, &inv.RelatedInvoiceID,
			&inv.Kind, &inv.Purpose, &inv.Status, &inv.AmountYuanCents, &inv.FapiaoID,
			&inv.WechatApplyID, &inv.WechatFapiaoNumber, &inv.BuyerSnapshot,
			&inv.FailReason, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func parseBuyerSnapshot(raw string) invoiceBuyerInput {
	var m map[string]string
	_ = json.Unmarshal([]byte(raw), &m)
	return invoiceBuyerInput{
		Type:        m["type"],
		Name:        m["name"],
		TaxpayerID:  m["taxpayer_id"],
		Address:     m["address"],
		Telephone:   m["telephone"],
		BankName:    m["bank_name"],
		BankAccount: m["bank_account"],
	}
}

func ReconcileInvoicesAfterRefund(ctx context.Context, orderID, refundAmountCents int64) error {
	cancelPendingInvoiceApplicationsForOrder(ctx, orderID)
	blues, err := loadIssuedBlueInvoices(ctx, orderID)
	if err != nil {
		return err
	}
	if len(blues) == 0 {
		slog.InfoContext(ctx, "invoice_refund_no_blue",
			"level", "info",
			"order_id", formatID(orderID),
		)
		return nil
	}
	var orderTotal int64
	if err := db.QueryRowContext(ctx, `SELECT total_yuan_cents FROM billing_resource_order WHERE id = ?`, orderID).Scan(&orderTotal); err != nil {
		return err
	}
	remain := remainingDealCentsAfterRefund(orderTotal, refundAmountCents)

	for _, blue := range blues {
		if err := reverseIssuedBlueAndMaybeReissue(ctx, blue, remain); err != nil {
			return err
		}
		remain = 0
	}
	return nil
}

func reverseIssuedBlueAndMaybeReissue(ctx context.Context, blue Invoice, remainCents int64) error {
	now := utcNow()
	if err := reverseWechatFapiaoFn(ctx, wechatFapiaoReverseRequest{
		FapiaoApplyID: blue.WechatApplyID,
		FapiaoID:      blue.FapiaoID,
		FapiaoNumber:  blue.WechatFapiaoNumber,
		ReverseReason: "订单退款全额红冲",
	}); err != nil {
		slog.ErrorContext(ctx, "invoice_reverse_failed",
			"level", "error",
			"invoice_id", formatID(blue.ID),
			"order_id", formatID(blue.OrderID),
			"error", err.Error(),
		)
		return err
	}
	redID := generateSnowflakeID()
	redFapiaoID := newMerchantFapiaoID()
	_, err := db.ExecContext(ctx, `
		INSERT INTO billing_invoice (
			id, tenant_id, order_id, application_id, related_invoice_id, kind, purpose, status,
			amount_yuan_cents, fapiao_id, wechat_apply_id, wechat_fapiao_number, buyer_snapshot,
			fail_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?, ?)`,
		redID, blue.TenantID, blue.OrderID, blue.ApplicationID, blue.ID,
		invoiceKindRed, invoicePurposeReverse, invoiceStatusReversePending,
		blue.AmountYuanCents, redFapiaoID, blue.WechatApplyID, blue.BuyerSnapshot, now, now,
	)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		UPDATE billing_invoice SET status = ?, updated_at = ? WHERE id = ?`,
		invoiceStatusReversePending, now, blue.ID)
	if err != nil {
		return err
	}
	deadline := ""
	if t, ok := parseInvoiceTimeUTC(now); ok {
		deadline = t.Add(time.Duration(invoiceReverseConfirmHours) * time.Hour).UTC().Format(time.RFC3339)
	}
	go djangoEmitEvent(ctx, "BILLING_INVOICE_REVERSE_PENDING", map[string]interface{}{
		"order_id":         formatID(blue.OrderID),
		"invoice_id":       formatID(blue.ID),
		"red_invoice_id":   formatID(redID),
		"confirm_deadline": deadline,
		"confirm_hours":    invoiceReverseConfirmHours,
	})
	slog.InfoContext(ctx, "invoice_reverse_pending",
		"level", "info",
		"order_id", formatID(blue.OrderID),
		"invoice_id", formatID(blue.ID),
		"red_invoice_id", formatID(redID),
		"confirm_deadline", deadline,
	)
	if remainCents <= 0 {
		return nil
	}
	return reissueBlueInvoice(ctx, blue, remainCents)
}

func reissueBlueInvoice(ctx context.Context, previous Invoice, remainCents int64) error {
	buyer := parseBuyerSnapshot(previous.BuyerSnapshot)
	if buyer.Name == "" {
		buyer = invoiceBuyerInput{Type: buyerTypeIndividual, Name: "购买方"}
	}
	now := utcNow()
	invID := generateSnowflakeID()
	fapiaoID := newMerchantFapiaoID()
	_, err := db.ExecContext(ctx, `
		INSERT INTO billing_invoice (
			id, tenant_id, order_id, application_id, related_invoice_id, kind, purpose, status,
			amount_yuan_cents, fapiao_id, wechat_apply_id, wechat_fapiao_number, buyer_snapshot,
			fail_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?, ?)`,
		invID, previous.TenantID, previous.OrderID, previous.ApplicationID, previous.ID,
		invoiceKindBlue, invoicePurposeReissue, invoiceStatusIssuing,
		remainCents, fapiaoID, previous.WechatApplyID, previous.BuyerSnapshot, now, now,
	)
	if err != nil {
		return err
	}
	_, err = issueWechatFapiaoFn(ctx, wechatFapiaoIssueRequest{
		OrderID:        previous.OrderID,
		FapiaoApplyID:  previous.WechatApplyID,
		FapiaoID:       fapiaoID,
		TotalAmountFen: remainCents,
		Buyer:          buyer,
		Purpose:        invoicePurposeReissue,
	})
	if err != nil {
		_, _ = db.ExecContext(ctx, `
			UPDATE billing_invoice SET status = ?, fail_reason = ?, updated_at = ? WHERE id = ?`,
			invoiceStatusFailed, truncateRunes(err.Error(), 500), utcNow(), invID)
		return err
	}
	go djangoEmitEvent(ctx, "BILLING_INVOICE_REISSUED", map[string]interface{}{
		"order_id":     formatID(previous.OrderID),
		"invoice_id":   formatID(invID),
		"amount_cents": remainCents,
		"fapiao_id":    fapiaoID,
	})
	slog.InfoContext(ctx, "invoice_reissued",
		"level", "info",
		"order_id", formatID(previous.OrderID),
		"invoice_id", formatID(invID),
		"amount_cents", remainCents,
	)
	return nil
}
