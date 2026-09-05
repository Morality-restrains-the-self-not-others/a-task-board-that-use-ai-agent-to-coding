package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// ErrNoRefundableAmount is returned when neither balance nor a paid refundable order applies.
var ErrNoRefundableAmount = errors.New("无可退金额")

// ErrOrderNotRefundable is returned for unpaid / grant / zero-amount / wrong-tenant orders.
var ErrOrderNotRefundable = errors.New("订单不可退款")

// ErrOrderAlreadyRefunded is returned when the order already has a pending/approved refund.
var ErrOrderAlreadyRefunded = errors.New("订单已有退款申请")

type orderRefundItem struct {
	resourceType string
	quantity     int64
	region       string
}

type sqlQuerier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func orderRefundChannel(paymentMethod, paymentRef string) (channel, providerRef string, ok bool) {
	method := strings.ToLower(strings.TrimSpace(paymentMethod))
	ref := strings.TrimSpace(paymentRef)
	switch method {
	case "wechat":
		channel = "wechat"
		providerRef = strings.TrimPrefix(ref, "wechat:")
	case "paypal":
		channel = "paypal"
		providerRef = strings.TrimPrefix(ref, "paypal:")
	default:
		if strings.HasPrefix(ref, "wechat:") {
			channel = "wechat"
			providerRef = strings.TrimPrefix(ref, "wechat:")
		} else if strings.HasPrefix(ref, "paypal:") {
			channel = "paypal"
			providerRef = strings.TrimPrefix(ref, "paypal:")
		} else {
			return "", "", false
		}
	}
	providerRef = strings.TrimSpace(providerRef)
	if providerRef == "" {
		return "", "", false
	}
	return channel, providerRef, true
}

func orderHasActiveRefund(ctx context.Context, exec sqlExecutor, orderID int64) (bool, error) {
	var exists int
	err := exec.QueryRowContext(ctx, `
		SELECT 1 FROM billing_refund_application
		WHERE order_id = ? AND status IN (?, ?) LIMIT 1`,
		orderID, refundStatusPending, refundStatusApproved,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ensureOrderPaymentLedger upserts a payment-ledger row for a direct-paid resource order
// so approve can refund via WeChat/PayPal. Idempotent on (channel, provider_ref).
func ensureOrderPaymentLedger(ctx context.Context, exec sqlExecutor, tenantID, accountID int64, order *ResourceOrder, now string) (int64, error) {
	channel, providerRef, ok := orderRefundChannel(order.PaymentMethod, order.PaymentRef)
	if !ok {
		return 0, ErrOrderNotRefundable
	}
	var ledgerID int64
	err := exec.QueryRowContext(ctx, `
		SELECT id FROM billing_payment_ledger
		WHERE tenant_id = ? AND channel = ? AND provider_ref = ?
		ORDER BY id ASC LIMIT 1`,
		tenantID, channel, providerRef,
	).Scan(&ledgerID)
	if err == nil && ledgerID > 0 {
		return ledgerID, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	expiresAt, err := creditLotExpiresAt(now)
	if err != nil {
		return 0, err
	}
	currency := "CNY"
	if channel == "paypal" {
		currency = "USD"
	}
	points := order.TotalYuanCents
	if points <= 0 {
		return 0, ErrNoRefundableAmount
	}
	ledgerID = generateSnowflakeID()
	_, err = exec.ExecContext(ctx, `
		INSERT INTO billing_payment_ledger (
			id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
			points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, NULL, ?, ?)`,
		ledgerID, tenantID, accountID, channel, providerRef,
		points, points, points, currency, now, expiresAt,
	)
	if err != nil {
		return 0, fmt.Errorf("ensure order payment ledger: %w", err)
	}
	slog.InfoContext(ctx, "order_payment_ledger_ensured",
		"level", "info",
		"order_id", formatID(order.ID),
		"ledger_id", formatID(ledgerID),
		"channel", channel,
		"provider_ref", providerRef,
		"points", points,
	)
	return ledgerID, nil
}

func loadOrderForRefund(ctx context.Context, exec sqlExecutor, tenantID, orderID int64) (*ResourceOrder, error) {
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
		return nil, ErrOrderNotRefundable
	}
	if err != nil {
		return nil, fmt.Errorf("查询关联订单失败: %w", err)
	}
	o.PaidAt = paidAt
	o.CancelledAt = cancelledAt
	if o.TenantID != tenantID {
		return nil, ErrOrderNotRefundable
	}
	if o.Status != OrderStatusPaid {
		return nil, ErrOrderNotRefundable
	}
	if o.TotalYuanCents <= 0 {
		return nil, ErrNoRefundableAmount
	}
	if _, _, ok := orderRefundChannel(o.PaymentMethod, o.PaymentRef); !ok {
		return nil, ErrOrderNotRefundable
	}
	return &o, nil
}

func loadOrderRefundItems(ctx context.Context, exec sqlExecutor, orderID int64) ([]orderRefundItem, error) {
	q, ok := exec.(sqlQuerier)
	if !ok {
		return nil, fmt.Errorf("revoke order resources: executor does not support Query")
	}
	rows, err := q.QueryContext(ctx, `
		SELECT resource_type, quantity, COALESCE(region,'')
		FROM billing_resource_order_item WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []orderRefundItem
	for rows.Next() {
		var it orderRefundItem
		if err := rows.Scan(&it.resourceType, &it.quantity, &it.region); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// revokeOrderResourcesOnRefund rolls back quotas granted by markOrderPaid (floor at 0).
func revokeOrderResourcesOnRefund(ctx context.Context, exec sqlExecutor, orderID, tenantID int64, now string) error {
	items, err := loadOrderRefundItems(ctx, exec, orderID)
	if err != nil {
		return err
	}
	for _, it := range items {
		switch it.resourceType {
		case ResourceTypeTaskPost:
			_, err := exec.ExecContext(ctx, `
				UPDATE billing_account
				SET task_post_quota = GREATEST(0, task_post_quota - ?), updated_at = ?
				WHERE tenant_id = ?`, it.quantity, now, tenantID)
			if err != nil {
				return err
			}
			if _, err := exec.ExecContext(ctx, `
				UPDATE billing_resource_grant
				SET remaining = 0
				WHERE order_id = ? AND tenant_id = ? AND resource_type = ?`,
				orderID, tenantID, ResourceTypeTaskPost); err != nil {
				return err
			}
		case ResourceTypeGitlabDisk:
			region := strings.TrimSpace(it.region)
			if region == "" {
				continue
			}
			_, err := exec.ExecContext(ctx, `
				UPDATE billing_tenant_gitlab_resource
				SET disk_gb = GREATEST(0, disk_gb - ?), updated_at = ?
				WHERE tenant_id = ? AND region = ?`, it.quantity, now, tenantID, region)
			if err != nil {
				return err
			}
		case ResourceTypeGitlabTraffic:
			region := strings.TrimSpace(it.region)
			if region == "" {
				continue
			}
			_, err := exec.ExecContext(ctx, `
				UPDATE billing_tenant_gitlab_resource
				SET traffic_prepaid_gb = GREATEST(0, traffic_prepaid_gb - ?), updated_at = ?
				WHERE tenant_id = ? AND region = ?`, it.quantity, now, tenantID, region)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// markOrderRefunded sets order status after a successful refund approval.
func markOrderRefunded(ctx context.Context, exec sqlExecutor, orderID int64) error {
	if _, err := exec.ExecContext(ctx, `
		UPDATE billing_resource_order SET status = ? WHERE id = ? AND status = ?`,
		OrderStatusRefunded, orderID, OrderStatusPaid,
	); err != nil {
		return err
	}
	// OPT-20260820-018: 已支付二维码行同步标 refunded，未支付行标 cancelled，
	// 避免残留 pending 行继续占表、干扰对账。
	return closePaymentPendingForOrder(ctx, exec, orderID, "refunded", "cancelled")
}

// closePaymentPendingForOrder resolves leftover billing_payment_pending rows after an
// order transitions to a terminal state: paid rows → paidTo, still-pending rows → pendingTo.
func closePaymentPendingForOrder(ctx context.Context, exec sqlExecutor, orderID int64, paidTo, pendingTo string) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE billing_payment_pending
		SET status = CASE WHEN status = 'paid' THEN ? ELSE ? END
		WHERE order_id = ? AND status IN ('pending', 'paid')`,
		paidTo, pendingTo, orderID,
	)
	return err
}

// executeOrderRefundAllocations refunds only the ledger tied to the order payment.
func executeOrderRefundAllocations(ctx context.Context, app *RefundApplication, order *ResourceOrder) ([]refundChannelAllocation, []map[string]interface{}, error) {
	channel, providerRef, ok := orderRefundChannel(order.PaymentMethod, order.PaymentRef)
	if !ok {
		return nil, nil, ErrOrderNotRefundable
	}
	var row paymentLedgerRow
	err := db.QueryRowContext(ctx, `
		SELECT id, channel, provider_ref, provider_capture_id, points, remaining_points, amount_minor, currency
		FROM billing_payment_ledger
		WHERE tenant_id = ? AND channel = ? AND provider_ref = ?
		ORDER BY id ASC LIMIT 1`,
		app.TenantID, channel, providerRef,
	).Scan(
		&row.ID, &row.Channel, &row.ProviderRef, &row.CaptureID,
		&row.Points, &row.RemainingPoints, &row.AmountMinor, &row.Currency,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("order payment ledger not found: %w", err)
	}

	prior := parseRefundProviderProgress(app.PaymentRefundRefs)
	alloc := app.FrozenPoints
	if alloc > row.RemainingPoints {
		alloc = row.RemainingPoints
	}
	if alloc <= 0 {
		return nil, nil, ErrNoRefundableAmount
	}
	outNo := refundOutRefundNo(app.ID, row.ID)
	lidKey := ledgerIDKey(row.ID)
	ref := ""
	if prev, ok := prior[lidKey]; ok {
		ref = strings.TrimSpace(fmt.Sprint(prev["refund_ref"]))
		if ref == "<nil>" {
			ref = ""
		} else if pts, ok := asInt64(prev["points"]); ok && pts > 0 {
			alloc = pts
		}
	}
	if ref == "" {
		ref, err = executeProviderRefundFn(ctx, providerRefundRequest{
			Channel:        row.Channel,
			ProviderRef:    row.ProviderRef,
			CaptureID:      row.CaptureID,
			Currency:       row.Currency,
			RefundPoints:   alloc,
			OriginalPoints: row.Points,
			AmountMinor:    row.AmountMinor,
			OutRefundNo:    outNo,
		})
		if err != nil {
			return nil, nil, err
		}
	}
	item := map[string]interface{}{
		"ledger_id":     formatID(row.ID),
		"channel":       row.Channel,
		"provider_ref":  row.ProviderRef,
		"points":        alloc,
		"currency":      row.Currency,
		"refund_ref":    ref,
		"out_refund_no": outNo,
		"order_id":      formatID(order.ID),
	}
	if err := persistRefundProviderProgress(ctx, app.ID, []map[string]interface{}{item}); err != nil {
		slog.WarnContext(ctx, "provider_refund_progress_persist_failed",
			"level", "warn",
			"application_id", formatID(app.ID),
			"error", err.Error(),
		)
	}
	return []refundChannelAllocation{{
		LedgerID: row.ID, Points: alloc, Channel: row.Channel,
		ProviderRef: row.ProviderRef, Currency: row.Currency, RefundRef: ref,
		OutRefundNo: outNo,
	}}, []map[string]interface{}{item}, nil
}
