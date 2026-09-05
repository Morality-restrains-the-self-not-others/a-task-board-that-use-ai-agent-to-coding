package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func applyInvoiceApplication(ctx context.Context, tenantID, orderID int64, applicantUserID string, buyer invoiceBuyerInput, idempotencyKey string) (*InvoiceApplication, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("缺少订单 ID")
	}
	if existing, err := replayPendingInvoiceIfKeyed(ctx, db, tenantID, orderID, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	conn, err := beginImmediateConn(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackImmediateConn(ctx, conn)

	if existing, err := replayPendingInvoiceIfKeyed(ctx, conn, tenantID, orderID, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	order, err := loadOrderForInvoice(ctx, conn, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != OrderStatusPaid {
		return nil, ErrInvoiceOrderNotPaid
	}
	if order.TotalYuanCents <= 0 {
		slog.WarnContext(ctx, "invoice_application_rejected_zero_amount",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"order_id", formatID(orderID),
			"total_yuan_cents", order.TotalYuanCents,
			"payment_method", order.PaymentMethod,
		)
		return nil, ErrInvoiceZeroAmount
	}
	channel, _, ok := orderRefundChannel(order.PaymentMethod, order.PaymentRef)
	if !ok || channel != "wechat" {
		return nil, ErrInvoiceOrderNotPaid
	}

	pending, err := loadPendingInvoiceApplicationByOrder(ctx, conn, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, ErrInvoicePendingExists
	}
	active, err := orderHasActiveBlueInvoice(ctx, conn, orderID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, ErrInvoiceAlreadyIssued
	}

	now := utcNow()
	appID := generateSnowflakeID()
	_, err = conn.ExecContext(ctx, `
		INSERT INTO billing_invoice_application (
			id, tenant_id, order_id, applicant_user_id, invoice_type, buyer_type, buyer_name, taxpayer_id,
			address, telephone, bank_name, bank_account, status, reviewer_user_id, review_note,
			reviewed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', NULL, ?, ?)`,
		appID, tenantID, orderID, applicantUserID, buyer.InvoiceType, buyer.Type, buyer.Name, buyer.TaxpayerID,
		buyer.Address, buyer.Telephone, buyer.BankName, buyer.BankAccount, invoiceAppPending, now, now,
	)
	if err != nil {
		return nil, err
	}
	if err := commitImmediateConn(ctx, conn); err != nil {
		return nil, err
	}
	conn = nil

	app := &InvoiceApplication{
		ID: appID, TenantID: tenantID, OrderID: orderID, ApplicantUserID: applicantUserID,
		InvoiceType: buyer.InvoiceType, BuyerType: buyer.Type, BuyerName: buyer.Name, TaxpayerID: buyer.TaxpayerID,
		Address: buyer.Address, Telephone: buyer.Telephone, BankName: buyer.BankName,
		BankAccount: buyer.BankAccount, Status: invoiceAppPending, CreatedAt: now, UpdatedAt: now,
	}
	go djangoEmitEvent(ctx, "BILLING_INVOICE_APPLICATION_SUBMITTED", map[string]interface{}{
		"application_id":    formatID(appID),
		"tenant_id":         formatID(tenantID),
		"order_id":          formatID(orderID),
		"applicant_user_id": applicantUserID,
		"buyer_type":        buyer.Type,
		"created_at":        now,
	})
	slog.InfoContext(ctx, "invoice_application_submitted",
		"level", "info",
		"tenant_id", formatID(tenantID),
		"application_id", formatID(appID),
		"order_id", formatID(orderID),
		"buyer_type", buyer.Type,
		"applicant_user_id", applicantUserID,
	)
	return app, nil
}

func replayPendingInvoiceIfKeyed(ctx context.Context, exec sqlExecutor, tenantID, orderID int64, idempotencyKey string) (*InvoiceApplication, error) {
	existing, err := loadPendingInvoiceApplicationByOrder(ctx, exec, tenantID, orderID)
	if err != nil || existing == nil {
		return existing, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, ErrInvoicePendingExists
	}
	slog.InfoContext(ctx, "invoice_application_idempotent_replay",
		"level", "info",
		"tenant_id", formatID(tenantID),
		"application_id", formatID(existing.ID),
		"order_id", formatID(orderID),
	)
	return existing, nil
}

func loadOrderForInvoice(ctx context.Context, exec sqlExecutor, tenantID, orderID int64) (*ResourceOrder, error) {
	var o ResourceOrder
	var paidAt, cancelledAt sql.NullString
	err := exec.QueryRowContext(ctx, `
		SELECT id, tenant_id, order_number, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(payment_ref,''), COALESCE(user_id,''),
		       created_at, paid_at, cancelled_at
		FROM billing_resource_order WHERE tenant_id = ? AND id = ?`, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &o.Status, &o.TotalYuanCents,
		&o.PaymentMethod, &o.PaymentRef, &o.UserID,
		&o.CreatedAt, &paidAt, &cancelledAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("订单不存在")
	}
	if err != nil {
		return nil, err
	}
	o.PaidAt = paidAt
	o.CancelledAt = cancelledAt
	return &o, nil
}
