package main

import (
	"context"
	"encoding/json"
	"fmt"

	"tracelog"
)

// creditRecharge 创建一笔充值交易（直接更新账户余额 + 写入交易流水 + 台账）。
// 仅测试使用；用户充值须走资源订单路径。
// 本函数定义在 _test.go 文件中，仅编译到测试二进制，不上生产。
func creditRecharge(
	ctx context.Context,
	tenantID int64,
	userID string,
	points int64,
	providerRef string,
	pointsSourceType string,
	description string,
	captureID string,
) (map[string]interface{}, error) {
	acc, _, err := getOrCreateBillingAccount(tenantID, true)
	if err != nil {
		return nil, fmt.Errorf("creditRecharge: account: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("creditRecharge: begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int64
	if err := tx.QueryRow(`SELECT balance FROM billing_account WHERE id = ?`, acc.ID).Scan(&balance); err != nil {
		return nil, fmt.Errorf("creditRecharge: read balance: %w", err)
	}
	after := balance + points
	now := utcNow()
	if _, err := tx.Exec(`UPDATE billing_account SET balance = ?, updated_at = ? WHERE id = ?`, after, now, acc.ID); err != nil {
		return nil, fmt.Errorf("creditRecharge: update balance: %w", err)
	}

	tid := generateSnowflakeID()
	if _, err := tx.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, user_id, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'recharge', ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		tid, acc.ID, points, balance, after, pointsSourceType, nullStr(userID), description,
		providerRef, now,
	); err != nil {
		return nil, fmt.Errorf("creditRecharge: insert txn: %w", err)
	}

	// billing_payment_ledger 已迁移为 BIGINT（047），始终写入付费台账。
	if rechargeSourceNeedsLedger(pointsSourceType) {
		channel, ref, currency, perr := parsePaymentLedgerMeta(pointsSourceType, providerRef)
		if perr != nil {
			channel = pointsSourceType
			ref = providerRef
			currency = "CNY"
		}
		expiresAt, err := creditLotExpiresAt(now)
		if err != nil {
			return nil, fmt.Errorf("creditRecharge: expires_at: %w", err)
		}
		ledgerID := generateSnowflakeID()
		// expires_at 列 DEFAULT ''；空串会被 expireCreditLots 当成已过期（'' <= now）。
		// 与 ensureOrderPaymentLedger 一致写入 now+12 个月。
		if _, err := tx.Exec(`
			INSERT INTO billing_payment_ledger (
				id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
				points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ledgerID, tenantID, acc.ID, channel, ref, captureID,
			points, points, points, currency, tid, now, expiresAt,
		); err != nil {
			return nil, fmt.Errorf("creditRecharge: insert ledger: %w", err)
		}
	}

	txnPayload := map[string]interface{}{
		"transaction_id":     providerRef,
		"account_id":         formatID(acc.ID),
		"amount_points":      fmt.Sprintf("%d", points),
		"amount_yuan":        pointsToYuanEquivalentStr(points),
		"points_source_type": pointsSourceType,
		"transaction_type":   "recharge",
		"created_at":         now,
	}
	if trid := tracelog.TraceIDFromContext(ctx); trid != "" {
		txnPayload["trace_id"] = trid
	}
	raw, _ := json.Marshal(txnPayload)
	outboxID := generateSnowflakeID()
	if _, err := tx.Exec(`
		INSERT INTO billing_outbox_message (id, billing_transaction_id, event_type, payload, status, created_at)
		VALUES (?, ?, 'BILLING_TRANSACTION_CREATED', ?, 'pending', ?)`,
		outboxID, tid, string(raw), now,
	); err != nil {
		return nil, fmt.Errorf("creditRecharge: outbox: %w", err)
	}

	ikID := generateSnowflakeID()
	if _, err := tx.Exec(`
		INSERT INTO billing_idempotency_key (id, `+"`"+`key`+"`"+`, created_at, transaction_id)
		VALUES (?, ?, ?, ?)`,
		ikID, "credit:"+providerRef, now, tid,
	); err != nil {
		return nil, fmt.Errorf("creditRecharge: idempotency: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("creditRecharge: commit: %w", err)
	}

	// 异步后置任务
	go func() {
		djangoEmitBillingEvent(context.Background(), outboxID, txnPayload)
	}()

	return map[string]interface{}{
		"transaction_db_id": formatID(tid),
		"balance_after":     after,
		"points":            points,
	}, nil
}
