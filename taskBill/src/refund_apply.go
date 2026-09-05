package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func applyRefundApplication(ctx context.Context, tenantID int64, applicantUserID, reason string, orderID int64, idempotencyKey string) (*RefundApplication, error) {
	enabled, err := getRefundPolicyEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrRefundDisabled
	}
	if orderID <= 0 {
		return nil, fmt.Errorf("缺少订单 ID")
	}
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		return nil, err
	}
	if existing, err := replayActiveRefundIfKeyed(ctx, db, tenantID, orderID, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	if pending, err := tenantHasPendingRefund(ctx, db, tenantID); err != nil {
		return nil, err
	} else if pending {
		return nil, ErrPendingRefundApplication
	}

	conn, err := beginImmediateConn(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackImmediateConn(ctx, conn)

	if existing, err := replayActiveRefundIfKeyed(ctx, conn, tenantID, orderID, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	if pending, err := tenantHasPendingRefund(ctx, conn, tenantID); err != nil {
		return nil, err
	} else if pending {
		return nil, ErrPendingRefundApplication
	}
	if active, err := orderHasActiveRefund(ctx, conn, orderID); err != nil {
		return nil, err
	} else if active {
		return nil, ErrOrderAlreadyRefunded
	}

	order, err := loadOrderForRefund(ctx, conn, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	now := utcNow()
	if _, err := ensureOrderPaymentLedger(ctx, conn, tenantID, acc.ID, order, now); err != nil {
		return nil, err
	}

	// Direct-paid resource orders never credited balance; do not freeze unrelated余额.
	// frozen_points records the unused (refundable) amount: full when no consumption lots.
	frozenPoints, err := unusedRefundableCents(ctx, conn, tenantID, orderID, order.TotalYuanCents)
	if err != nil {
		return nil, err
	}
	if frozenPoints <= 0 {
		return nil, ErrNoRefundableAmount
	}

	appID := generateSnowflakeID()
	_, err = conn.ExecContext(ctx, `
		INSERT INTO billing_refund_application (
			id, tenant_id, account_id, applicant_user_id, frozen_points, status, reason,
			reviewer_user_id, review_note, payment_refund_refs, order_id, reviewed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, '', '', '[]', ?, NULL, ?, ?)`,
		appID, tenantID, acc.ID, applicantUserID, frozenPoints, refundStatusPending, reason,
		orderID, now, now,
	)
	if err != nil {
		return nil, err
	}
	if err := commitImmediateConn(ctx, conn); err != nil {
		return nil, err
	}
	conn = nil

	app := &RefundApplication{
		ID:              appID,
		TenantID:        tenantID,
		AccountID:       acc.ID,
		ApplicantUserID: applicantUserID,
		FrozenPoints:    frozenPoints,
		Status:          refundStatusPending,
		Reason:          reason,
		OrderID:         sql.NullInt64{Int64: orderID, Valid: true},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	payload := map[string]interface{}{
		"application_id":    formatID(appID),
		"tenant_id":         formatID(tenantID),
		"account_id":        formatID(acc.ID),
		"applicant_user_id": applicantUserID,
		"frozen_points":     frozenPoints,
		"order_id":          formatID(orderID),
		"reason":            reason,
		"created_at":        now,
	}
	go djangoEmitEvent(ctx, "BILLING_REFUND_APPLICATION_SUBMITTED", payload)
	slog.InfoContext(ctx, "refund_application_submitted",
		"level", "info",
		"tenant_id", formatID(tenantID),
		"application_id", formatID(appID),
		"order_id", formatID(orderID),
		"frozen_points", frozenPoints,
		"applicant_user_id", applicantUserID,
	)
	return app, nil
}

func loadActiveRefundApplicationByOrder(ctx context.Context, exec sqlExecutor, tenantID, orderID int64) (*RefundApplication, error) {
	var app RefundApplication
	err := exec.QueryRowContext(ctx, `
		SELECT id, tenant_id, account_id, applicant_user_id, frozen_points, status, reason,
		       reviewer_user_id, review_note, payment_refund_refs, order_id, reviewed_at, created_at, updated_at
		FROM billing_refund_application
		WHERE tenant_id = ? AND order_id = ? AND status IN (?, ?)
		ORDER BY id ASC LIMIT 1`,
		tenantID, orderID, refundStatusPending, refundStatusApproved,
	).Scan(
		&app.ID, &app.TenantID, &app.AccountID, &app.ApplicantUserID, &app.FrozenPoints, &app.Status,
		&app.Reason, &app.ReviewerUserID, &app.ReviewNote, &app.PaymentRefundRefs,
		&app.OrderID, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// replayActiveRefundIfKeyed returns the existing active application when Idempotency-Key is set.
// Empty key keeps the historical 409 conflict for a second submit.
func replayActiveRefundIfKeyed(ctx context.Context, exec sqlExecutor, tenantID, orderID int64, idempotencyKey string) (*RefundApplication, error) {
	existing, err := loadActiveRefundApplicationByOrder(ctx, exec, tenantID, orderID)
	if err != nil || existing == nil {
		return existing, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, ErrOrderAlreadyRefunded
	}
	slog.InfoContext(ctx, "refund_application_idempotent_replay",
		"level", "info",
		"tenant_id", formatID(tenantID),
		"application_id", formatID(existing.ID),
		"order_id", formatID(orderID),
	)
	return existing, nil
}
