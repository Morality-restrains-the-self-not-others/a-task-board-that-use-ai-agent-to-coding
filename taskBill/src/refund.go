package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

const (
	refundStatusPending  = "pending"
	refundStatusApproved = "approved"
	refundStatusRejected = "rejected"
)

// ErrPendingRefundApplication is returned when a tenant already has a pending refund application.
var ErrPendingRefundApplication = errors.New("pending refund application already exists")

type RefundApplication struct {
	ID                int64
	TenantID          int64
	AccountID         int64
	ApplicantUserID   string
	FrozenPoints      int64
	Status            string
	Reason            string
	ReviewerUserID    string
	ReviewNote        string
	PaymentRefundRefs string
	OrderID           sql.NullInt64
	ReviewedAt        sql.NullString
	CreatedAt         string
	UpdatedAt         string
}

type paymentLedgerRow struct {
	ID              int64
	Channel         string
	ProviderRef     string
	CaptureID       string
	Points          int64
	RemainingPoints int64
	AmountMinor     int64
	Currency        string
}

func beginImmediateConn(ctx context.Context) (*sql.Conn, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "START TRANSACTION"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func commitImmediateConn(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return nil
	}
	_, err := conn.ExecContext(ctx, "COMMIT")
	_ = conn.Close()
	return err
}

func rollbackImmediateConn(ctx context.Context, conn *sql.Conn) {
	if conn == nil {
		return
	}
	_, _ = conn.ExecContext(ctx, "ROLLBACK")
	_ = conn.Close()
}

func rechargeSourceNeedsLedger(pointsSourceType string) bool {
	s := strings.TrimSpace(pointsSourceType)
	if s == "" {
		return false
	}
	switch s {
	case paypalPointsSourceType, wechatPointsSourceType,
		"user_recharge_admin", "admin_grant", "user_recharge":
		return true
	default:
		return strings.HasPrefix(s, "user_recharge") || strings.Contains(s, "grant")
	}
}

func parsePaymentLedgerMeta(pointsSourceType, txnID string) (channel, providerRef, currency string, err error) {
	txnID = strings.TrimSpace(txnID)
	switch pointsSourceType {
	case paypalPointsSourceType:
		channel = "paypal"
		currency = "USD"
		if strings.HasPrefix(txnID, "paypal:") {
			providerRef = strings.TrimPrefix(txnID, "paypal:")
		} else {
			providerRef = txnID
		}
	case wechatPointsSourceType:
		channel = "wechat"
		currency = "CNY"
		if strings.HasPrefix(txnID, "wechat:") {
			providerRef = strings.TrimPrefix(txnID, "wechat:")
		} else {
			providerRef = txnID
		}
	case "user_recharge_admin":
		channel = "admin"
		currency = "CNY"
		providerRef = txnID
	case "admin_grant":
		channel = "grant"
		currency = "CNY"
		providerRef = txnID
	case "user_recharge":
		channel = "other"
		currency = "CNY"
		providerRef = txnID
	default:
		if strings.HasPrefix(pointsSourceType, "user_recharge") || strings.Contains(pointsSourceType, "grant") {
			channel = "other"
			currency = "CNY"
			providerRef = txnID
		} else {
			return "", "", "", fmt.Errorf("unsupported recharge source %q", pointsSourceType)
		}
	}
	if providerRef == "" {
		return "", "", "", fmt.Errorf("missing provider_ref in transaction_id")
	}
	return channel, providerRef, currency, nil
}

func insertPaymentLedger(ctx context.Context, exec sqlExecutor, tenantID, accountID, txnDBID, points int64, pointsSourceType, txnID, now, providerCaptureID string) error {
	channel, providerRef, currency, err := parsePaymentLedgerMeta(pointsSourceType, txnID)
	if err != nil {
		return err
	}
	expiresAt, err := creditLotExpiresAt(now)
	if err != nil {
		return err
	}
	ledgerID := generateSnowflakeID()
	_, err = exec.ExecContext(ctx, `
		INSERT INTO billing_payment_ledger (
			id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
			points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ledgerID, tenantID, accountID, channel, providerRef, strings.TrimSpace(providerCaptureID),
		points, points, points, currency, txnDBID, now, expiresAt,
	)
	return err
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func tenantHasPendingRefund(ctx context.Context, exec sqlExecutor, tenantID int64) (bool, error) {
	var exists int
	err := exec.QueryRowContext(ctx, `
		SELECT 1 FROM billing_refund_application
		WHERE tenant_id = ? AND status = ? LIMIT 1`, tenantID, refundStatusPending).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func loadRefundApplication(ctx context.Context, exec sqlExecutor, appID int64) (*RefundApplication, error) {
	var app RefundApplication
	err := exec.QueryRowContext(ctx, `
		SELECT id, tenant_id, account_id, applicant_user_id, frozen_points, status, reason,
		       reviewer_user_id, review_note, payment_refund_refs, order_id, reviewed_at, created_at, updated_at
		FROM billing_refund_application WHERE id = ?`, appID).Scan(
		&app.ID, &app.TenantID, &app.AccountID, &app.ApplicantUserID, &app.FrozenPoints, &app.Status,
		&app.Reason, &app.ReviewerUserID, &app.ReviewNote, &app.PaymentRefundRefs,
		&app.OrderID, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refund application not found")
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func rejectRefundApplication(ctx context.Context, appID int64, reviewerUserID, note string) (*RefundApplication, error) {
	conn, err := beginImmediateConn(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackImmediateConn(ctx, conn)

	app, err := loadRefundApplication(ctx, conn, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != refundStatusPending {
		return nil, fmt.Errorf("refund application is not pending")
	}

	var balance, frozen int64
	err = conn.QueryRowContext(ctx, `
		SELECT balance, frozen_balance FROM billing_account WHERE id = ?`, app.AccountID).
		Scan(&balance, &frozen)
	if err != nil {
		return nil, err
	}

	now := utcNow()
	// Order-direct refunds never froze balance; only unfreeze when frozen covers the amount.
	if frozen >= app.FrozenPoints && app.FrozenPoints > 0 {
		newBalance := balance + app.FrozenPoints
		newFrozen := frozen - app.FrozenPoints
		_, err = conn.ExecContext(ctx, `
			UPDATE billing_account SET balance = ?, frozen_balance = ?, updated_at = ? WHERE id = ?`,
			newBalance, newFrozen, now, app.AccountID,
		)
		if err != nil {
			return nil, err
		}
	} else if !app.OrderID.Valid || app.OrderID.Int64 <= 0 {
		return nil, fmt.Errorf("frozen balance insufficient")
	}

	_, err = conn.ExecContext(ctx, `
		UPDATE billing_refund_application
		SET status = ?, reviewer_user_id = ?, review_note = ?, reviewed_at = ?, updated_at = ?
		WHERE id = ?`,
		refundStatusRejected, reviewerUserID, note, now, now, appID,
	)
	if err != nil {
		return nil, err
	}
	if err := commitImmediateConn(ctx, conn); err != nil {
		return nil, err
	}
	conn = nil

	app.Status = refundStatusRejected
	app.ReviewerUserID = reviewerUserID
	app.ReviewNote = note
	app.ReviewedAt = sql.NullString{String: now, Valid: true}
	app.UpdatedAt = now

	payload := map[string]interface{}{
		"application_id":   formatID(appID),
		"tenant_id":        formatID(app.TenantID),
		"reviewer_user_id": reviewerUserID,
		"review_note":      note,
		"unfrozen_points":  app.FrozenPoints,
		"reviewed_at":      now,
	}
	go djangoEmitEvent(ctx, "BILLING_REFUND_APPLICATION_REJECTED", payload)
	slog.InfoContext(ctx, "refund_application_rejected",
		"level", "info",
		"application_id", formatID(appID),
		"tenant_id", formatID(app.TenantID),
		"reviewer_user_id", reviewerUserID,
		"unfrozen_points", app.FrozenPoints,
	)
	return app, nil
}

// refundListFilter 退款申请列表过滤条件（OPT-20260823-045：管理端表头列过滤）。
type refundListFilter struct {
	TenantID int64
	OrderID  int64
	Status   string
}

func listRefundApplications(ctx context.Context, f refundListFilter) ([]map[string]interface{}, error) {
	query := `
		SELECT id, tenant_id, account_id, applicant_user_id, frozen_points, status, reason,
		       reviewer_user_id, review_note, payment_refund_refs, order_id, reviewed_at, created_at, updated_at
		FROM billing_refund_application`
	args := []interface{}{}
	where := []string{}
	if f.TenantID > 0 {
		where = append(where, "tenant_id = ?")
		args = append(args, f.TenantID)
	}
	if f.OrderID > 0 {
		where = append(where, "order_id = ?")
		args = append(args, f.OrderID)
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		where = append(where, "status = ?")
		args = append(args, s)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var app RefundApplication
		if err := rows.Scan(
			&app.ID, &app.TenantID, &app.AccountID, &app.ApplicantUserID, &app.FrozenPoints, &app.Status,
			&app.Reason, &app.ReviewerUserID, &app.ReviewNote, &app.PaymentRefundRefs,
			&app.OrderID, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, refundApplicationJSON(&app))
	}
	return out, rows.Err()
}

func int64FromInterface(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case string:
		id, _ := strconv.ParseInt(n, 10, 64)
		return id
	default:
		return 0
	}
}

func refundApplicationJSON(app *RefundApplication) map[string]interface{} {
	m := map[string]interface{}{
		"id":                formatID(app.ID),
		"tenant_id":         formatID(app.TenantID),
		"account_id":        formatID(app.AccountID),
		"applicant_user_id": app.ApplicantUserID,
		"frozen_points":     app.FrozenPoints,
		"frozen_yuan":       centsToYuanStr(app.FrozenPoints),
		"status":            app.Status,
		"reason":            app.Reason,
		"reviewer_user_id":  app.ReviewerUserID,
		"review_note":       app.ReviewNote,
		"created_at":        app.CreatedAt,
		"updated_at":        app.UpdatedAt,
	}
	if app.ReviewedAt.Valid {
		m["reviewed_at"] = app.ReviewedAt.String
	}
	if app.OrderID.Valid {
		m["order_id"] = formatID(app.OrderID.Int64)
	}
	if strings.TrimSpace(app.PaymentRefundRefs) != "" {
		var refs []interface{}
		if json.Unmarshal([]byte(app.PaymentRefundRefs), &refs) == nil {
			m["payment_refund_refs"] = refs
		} else {
			m["payment_refund_refs"] = app.PaymentRefundRefs
		}
	}
	return m
}
