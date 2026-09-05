package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

type refundChannelAllocation struct {
	LedgerID    int64
	Points      int64
	Channel     string
	ProviderRef string
	Currency    string
	RefundRef   string
	OutRefundNo string
}

// executeRefundChannelAllocations calls payment channels with deterministic out_refund_no,
// persists progress outside the final local txn, and skips ledgers already refunded on retry.
func executeRefundChannelAllocations(ctx context.Context, app *RefundApplication) ([]refundChannelAllocation, []map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, channel, provider_ref, provider_capture_id, points, remaining_points, amount_minor, currency
		FROM billing_payment_ledger
		WHERE tenant_id = ? AND remaining_points > 0
		ORDER BY created_at ASC`, app.TenantID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var ledgerRows []paymentLedgerRow
	for rows.Next() {
		var row paymentLedgerRow
		if err := rows.Scan(
			&row.ID, &row.Channel, &row.ProviderRef, &row.CaptureID,
			&row.Points, &row.RemainingPoints, &row.AmountMinor, &row.Currency,
		); err != nil {
			return nil, nil, err
		}
		ledgerRows = append(ledgerRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	prior := parseRefundProviderProgress(app.PaymentRefundRefs)
	remaining := app.FrozenPoints
	var allocations []refundChannelAllocation
	var refundRefs []map[string]interface{}

	for _, row := range ledgerRows {
		if remaining <= 0 {
			break
		}
		alloc := row.RemainingPoints
		if alloc > remaining {
			alloc = remaining
		}
		outNo := refundOutRefundNo(app.ID, row.ID)
		lidKey := ledgerIDKey(row.ID)
		ref := ""
		if prev, ok := prior[lidKey]; ok {
			ref = strings.TrimSpace(fmt.Sprint(prev["refund_ref"]))
			if ref == "" || ref == "<nil>" {
				ref = ""
			} else if pts, ok := asInt64(prev["points"]); ok && pts > 0 {
				alloc = pts
			}
			if ref != "" {
				slog.InfoContext(ctx, "provider_refund_reuse_progress",
					"level", "info",
					"application_id", formatID(app.ID),
					"ledger_id", lidKey,
					"out_refund_no", outNo,
					"refund_ref", ref,
				)
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
				slog.WarnContext(ctx, "provider_refund_failed",
					"level", "warn",
					"application_id", formatID(app.ID),
					"ledger_id", formatID(row.ID),
					"out_refund_no", outNo,
					"error", err.Error(),
				)
				_ = persistRefundProviderProgress(ctx, app.ID, refundRefs)
				return nil, nil, err
			}
		}
		allocations = append(allocations, refundChannelAllocation{
			LedgerID: row.ID, Points: alloc, Channel: row.Channel,
			ProviderRef: row.ProviderRef, Currency: row.Currency, RefundRef: ref,
			OutRefundNo: outNo,
		})
		item := map[string]interface{}{
			"ledger_id":     formatID(row.ID),
			"channel":       row.Channel,
			"provider_ref":  row.ProviderRef,
			"points":        alloc,
			"currency":      row.Currency,
			"refund_ref":    ref,
			"out_refund_no": outNo,
		}
		refundRefs = append(refundRefs, item)
		prior[lidKey] = item
		if err := persistRefundProviderProgress(ctx, app.ID, refundRefs); err != nil {
			slog.WarnContext(ctx, "provider_refund_progress_persist_failed",
				"level", "warn",
				"application_id", formatID(app.ID),
				"error", err.Error(),
			)
		}
		remaining -= alloc
	}
	return allocations, refundRefs, nil
}

func approveRefundApplication(ctx context.Context, appID int64, reviewerUserID, note string) (*RefundApplication, error) {
	// Phase 1: load pending app + execute channel refunds (idempotent out_refund_no) with
	// durable progress, outside the exclusive local txn.
	app, err := loadRefundApplication(ctx, db, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != refundStatusPending {
		return nil, fmt.Errorf("refund application is not pending")
	}

	var allocations []refundChannelAllocation
	var refundRefs []map[string]interface{}
	if app.OrderID.Valid && app.OrderID.Int64 > 0 {
		order, _, err := loadOrder(app.TenantID, app.OrderID.Int64)
		if err != nil {
			return nil, err
		}
		allocations, refundRefs, err = executeOrderRefundAllocations(ctx, app, order)
		if err != nil {
			return nil, err
		}
	} else {
		allocations, refundRefs, err = executeRefundChannelAllocations(ctx, app)
		if err != nil {
			return nil, err
		}
	}

	// Phase 2: local ledger/account/application commit.
	conn, err := beginImmediateConn(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackImmediateConn(ctx, conn)

	app, err = loadRefundApplication(ctx, conn, appID)
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
	for _, a := range allocations {
		_, err = conn.ExecContext(ctx, `
			UPDATE billing_payment_ledger
			SET remaining_points = remaining_points - ?
			WHERE id = ? AND remaining_points >= ?`,
			a.Points, a.LedgerID, a.Points,
		)
		if err != nil {
			return nil, err
		}
	}

	// Order-direct refunds never froze balance; only clear frozen when it covers the amount.
	if frozen >= app.FrozenPoints && app.FrozenPoints > 0 {
		newFrozen := frozen - app.FrozenPoints
		_, err = conn.ExecContext(ctx, `
			UPDATE billing_account SET frozen_balance = ?, updated_at = ? WHERE id = ?`,
			newFrozen, now, app.AccountID,
		)
		if err != nil {
			return nil, err
		}
	} else if !app.OrderID.Valid || app.OrderID.Int64 <= 0 {
		return nil, fmt.Errorf("frozen balance insufficient")
	}

	if app.OrderID.Valid && app.OrderID.Int64 > 0 {
		if err := revokeOrderResourcesOnRefund(ctx, conn, app.OrderID.Int64, app.TenantID, now); err != nil {
			return nil, err
		}
		if err := markOrderRefunded(ctx, conn, app.OrderID.Int64); err != nil {
			return nil, err
		}
	}

	refundTxnID := generateSnowflakeID()
	extTxnID := fmt.Sprintf("refund:%d", appID)
	_, err = conn.ExecContext(ctx, `
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, user_id, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'refund', ?, ?, ?, 'refund', ?, ?, ?, ?, 0)`,
		refundTxnID, app.AccountID, -app.FrozenPoints, balance, balance,
		nullStr(reviewerUserID), note, extTxnID, now,
	)
	if err != nil {
		return nil, err
	}

	refsJSON, _ := json.Marshal(refundRefs)
	_, err = conn.ExecContext(ctx, `
		UPDATE billing_refund_application
		SET status = ?, reviewer_user_id = ?, review_note = ?, payment_refund_refs = ?,
		    reviewed_at = ?, updated_at = ?
		WHERE id = ?`,
		refundStatusApproved, reviewerUserID, note, string(refsJSON), now, now, appID,
	)
	if err != nil {
		return nil, err
	}

	txPayload := map[string]interface{}{
		"transaction_id":     extTxnID,
		"account_id":         formatID(app.AccountID),
		"amount_points":      fmt.Sprintf("%d", app.FrozenPoints),
		"amount_yuan":        pointsToYuanEquivalentStr(app.FrozenPoints),
		"points_source_type": "refund",
		"transaction_type":   "refund",
		"created_at":         now,
	}
	txRaw, _ := json.Marshal(txPayload)
	outboxID := generateSnowflakeID()
	_, err = conn.ExecContext(ctx, `
		INSERT INTO billing_outbox_message (id, billing_transaction_id, event_type, payload, status, created_at)
		VALUES (?, ?, 'BILLING_TRANSACTION_CREATED', ?, 'pending', ?)`,
		outboxID, refundTxnID, string(txRaw), now,
	)
	if err != nil {
		return nil, err
	}

	if err := commitImmediateConn(ctx, conn); err != nil {
		return nil, err
	}
	conn = nil

	if app.OrderID.Valid && app.OrderID.Int64 > 0 {
		if verr := voidReferralAccrualForOrderID(app.OrderID.Int64); verr != nil {
			slog.WarnContext(ctx, "referral_void_on_refund_failed",
				"order_id", app.OrderID.Int64, "error", verr.Error())
		}
	}

	app.Status = refundStatusApproved
	app.ReviewerUserID = reviewerUserID
	app.ReviewNote = note
	app.PaymentRefundRefs = string(refsJSON)
	app.ReviewedAt = sql.NullString{String: now, Valid: true}
	app.UpdatedAt = now

	completedPayload := map[string]interface{}{
		"application_id":        formatID(appID),
		"tenant_id":             formatID(app.TenantID),
		"reviewer_user_id":      reviewerUserID,
		"review_note":           note,
		"frozen_points":         app.FrozenPoints,
		"payment_refund_refs":   refundRefs,
		"reviewed_at":           now,
		"refund_transaction_id": extTxnID,
	}
	go djangoEmitEvent(ctx, "BILLING_REFUND_COMPLETED", completedPayload)
	// 退款后取消该租户的待分账记录（最佳努力，失败不阻塞退款）
	go cancelPendingProfitSharingsForTenant(ctx, app.TenantID, app.FrozenPoints)
	if app.OrderID.Valid && app.OrderID.Int64 > 0 {
		if ierr := ReconcileInvoicesAfterRefund(ctx, app.OrderID.Int64, app.FrozenPoints); ierr != nil {
			slog.ErrorContext(ctx, "invoice_reconcile_after_refund_failed",
				"level", "error",
				"order_id", formatID(app.OrderID.Int64),
				"application_id", formatID(appID),
				"error", ierr.Error(),
			)
		}
	}
	go djangoEmitBillingEvent(ctx, outboxID, txPayload)
	slog.InfoContext(ctx, "refund_application_approved",
		"level", "info",
		"application_id", formatID(appID),
		"tenant_id", formatID(app.TenantID),
		"reviewer_user_id", reviewerUserID,
		"frozen_points", app.FrozenPoints,
	)
	return app, nil
}

func asInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case int:
		return int64(t), true
	default:
		return 0, false
	}
}
