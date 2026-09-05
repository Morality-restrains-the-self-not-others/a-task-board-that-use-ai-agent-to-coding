package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"
)

// maxWechatFapiaoNumberRunes 与 billing_invoice.wechat_fapiao_number VARCHAR(32) 对齐。
const maxWechatFapiaoNumberRunes = 32

var ErrInvoiceFapiaoNumberTooLong = errors.New("微信发票号码过长")

func approveInvoiceApplication(ctx context.Context, appID int64, reviewerUserID, note, fapiaoNumber string) (*InvoiceApplication, error) {
	fapiaoNumber = strings.TrimSpace(fapiaoNumber)
	if utf8.RuneCountInString(fapiaoNumber) > maxWechatFapiaoNumberRunes {
		return nil, ErrInvoiceFapiaoNumberTooLong
	}
	app, err := loadInvoiceApplication(ctx, db, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != invoiceAppPending {
		return nil, ErrInvoiceApplicationNotPend
	}
	order, _, err := loadOrder(app.TenantID, app.OrderID)
	if err != nil {
		return nil, err
	}
	if order.TotalYuanCents <= 0 {
		slog.WarnContext(ctx, "invoice_approve_rejected_zero_amount",
			"level", "warn",
			"application_id", formatID(appID),
			"tenant_id", formatID(app.TenantID),
			"order_id", formatID(app.OrderID),
			"total_yuan_cents", order.TotalYuanCents,
		)
		return nil, ErrInvoiceZeroAmount
	}
	txnID, err := lookupWechatTransactionIDForOrder(app.OrderID)
	if err != nil {
		slog.WarnContext(ctx, "invoice_approve_missing_wechat_txn",
			"level", "warn",
			"application_id", formatID(appID),
			"order_id", formatID(app.OrderID),
			"error", err.Error(),
		)
		return nil, err
	}

	now := utcNow()
	invID := generateSnowflakeID()
	fapiaoID := newMerchantFapiaoID()
	buyer := buyerFromApplication(app)
	// 手动开具模式（issue_mode=manual）：不再调用微信开票 API。
	// 管理员在微信商户平台手动开具发票后，在此面板点击「已开具」登记，
	// 发票记录直接落 issued，杜绝审批后无回调卡在 issuing。
	_, err = db.ExecContext(ctx, `
		INSERT INTO billing_invoice (
			id, tenant_id, order_id, application_id, related_invoice_id, kind, purpose, status,
			amount_yuan_cents, fapiao_id, wechat_apply_id, wechat_fapiao_number, invoice_file_path,
			buyer_snapshot, fail_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?)`,
		invID, app.TenantID, app.OrderID, app.ID, invoiceKindBlue, invoicePurposeOriginal,
		invoiceStatusIssued, order.TotalYuanCents, fapiaoID, txnID, fapiaoNumber, app.InvoiceFilePath, buyer.snapshotJSON(), now, now,
	)
	if err != nil {
		return nil, err
	}

	_, err = db.ExecContext(ctx, `
		UPDATE billing_invoice_application
		SET status = ?, reviewer_user_id = ?, review_note = ?, reviewed_at = ?, updated_at = ?
		WHERE id = ? AND status = ?`,
		invoiceAppApproved, reviewerUserID, note, now, now, appID, invoiceAppPending,
	)
	if err != nil {
		return nil, err
	}

	app.Status = invoiceAppApproved
	app.ReviewerUserID = reviewerUserID
	app.ReviewNote = note
	app.ReviewedAt = sql.NullString{String: now, Valid: true}
	app.UpdatedAt = now

	go djangoEmitEvent(ctx, "BILLING_INVOICE_ISSUE_ACCEPTED", map[string]interface{}{
		"application_id": formatID(appID),
		"invoice_id":     formatID(invID),
		"tenant_id":      formatID(app.TenantID),
		"order_id":       formatID(app.OrderID),
		"fapiao_id":      fapiaoID,
		"amount_cents":   order.TotalYuanCents,
		"issue_mode":     "manual",
	})
	slog.InfoContext(ctx, "invoice_application_approved",
		"level", "info",
		"issue_mode", "manual",
		"application_id", formatID(appID),
		"invoice_id", formatID(invID),
		"tenant_id", formatID(app.TenantID),
		"order_id", formatID(app.OrderID),
		"reviewer_user_id", reviewerUserID,
	)
	return app, nil
}

func rejectInvoiceApplication(ctx context.Context, appID int64, reviewerUserID, note string) (*InvoiceApplication, error) {
	app, err := loadInvoiceApplication(ctx, db, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != invoiceAppPending {
		return nil, ErrInvoiceApplicationNotPend
	}
	now := utcNow()
	_, err = db.ExecContext(ctx, `
		UPDATE billing_invoice_application
		SET status = ?, reviewer_user_id = ?, review_note = ?, reviewed_at = ?, updated_at = ?
		WHERE id = ? AND status = ?`,
		invoiceAppRejected, reviewerUserID, note, now, now, appID, invoiceAppPending,
	)
	if err != nil {
		return nil, err
	}
	app.Status = invoiceAppRejected
	app.ReviewerUserID = reviewerUserID
	app.ReviewNote = note
	app.ReviewedAt = sql.NullString{String: now, Valid: true}
	app.UpdatedAt = now
	slog.InfoContext(ctx, "invoice_application_rejected",
		"level", "info",
		"application_id", formatID(appID),
		"tenant_id", formatID(app.TenantID),
		"order_id", formatID(app.OrderID),
		"reviewer_user_id", reviewerUserID,
	)
	return app, nil
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
