package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"tracelog"
)

// getTaskPostQuotaRemaining 获取任务帖剩余配额
func getTaskPostQuotaRemaining(tenantID int64) (int64, error) {
	var quota int64
	err := db.QueryRow(
		`SELECT COALESCE(task_post_quota, 0) FROM billing_account WHERE tenant_id = ?`, tenantID,
	).Scan(&quota)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return quota, err
}

// consumeTaskPostQuota 消耗一个任务帖配额（优先消耗赠送资源，FEFO）。
func consumeTaskPostQuota(ctx context.Context, tenantID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := consumeTaskPostQuotaLotTx(ctx, tx, tenantID); err != nil {
		return err
	}
	return tx.Commit()
}

// consumeTaskPostQuotaTx 在给定事务中消耗一个任务帖配额（优先消耗赠送资源，FEFO），与消费流水记录合并到同一事务。
func consumeTaskPostQuotaTx(ctx context.Context, tx *sql.Tx, tenantID int64) error {
	_, err := consumeTaskPostQuotaLotTx(ctx, tx, tenantID)
	return err
}

// consumeAndRecordTaskPostQuota 消耗一个任务帖配额并记录消费流水。
// 在同一事务中完成：配额扣减 + billing_transaction + billing_usage + outbox。
// 由任务创建流程调用，确保配额消耗可审计。
func consumeAndRecordTaskPostQuota(ctx context.Context, tenantID int64, taskID, workspaceID, userID, projectID, idempotencyKey string) (map[string]interface{}, error) {
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		return nil, err
	}

	// 幂等性检查：如果提供了 idempotency_key 且已存在，直接返回成功
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("task_post_quota:%s", taskID)
	}
	txnID := fmt.Sprintf("task_post_quota:%s", taskID)

	// ensureBillingUnit 必须在事务外执行，避免 MaxOpenConns(1) 下死锁
	now := utcNow()
	unitID, err := ensureBillingUnit("task_post_quota", "任务帖配额消耗", 0)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 幂等性：检查 idempotency_key 是否已存在
	var existingTxn int64
	err = tx.QueryRow(`SELECT transaction_id FROM billing_idempotency_key WHERE `+"`"+`key`+"`"+` = ?`, idempotencyKey).Scan(&existingTxn)
	if err == nil {
		// 已存在，返回幂等成功响应
		_ = tx.Commit()
		return map[string]interface{}{
			"success":        true,
			"idempotent":     true,
			"transaction_id": txnID,
			"account_id":     formatID(acc.ID),
			"cost_cents":     0,
			"quota_consumed": 0,
			"expires_at":     taskPostExpiresAt(now, ""), // 幂等重放：按当前时点给出存续期
		}, nil
	}

	// Step 1: 原子消耗 1 个任务帖配额
	hit, err := consumeTaskPostQuotaLotTx(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}

	// Step 2: 记录消费流水（cost=0，因为配额已预购支付）
	desc := fmt.Sprintf("创建任务帖消耗配额 (task_id=%s)", taskID)
	txnID = fmt.Sprintf("task_post_quota:%s", taskID)
	tid := generateSnowflakeID()

	_, err = tx.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, project_id, user_id, workspace_id, task_id,
			billing_unit_id, usage_amount, description, transaction_id, created_at,
			related_order_id, source_grant_id
		) VALUES (?, (SELECT id FROM billing_account WHERE tenant_id = ?), 'consumption', 0,
			(SELECT balance FROM billing_account WHERE tenant_id = ?),
			(SELECT balance FROM billing_account WHERE tenant_id = ?),
			'quota_consumption', ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?)`,
		tid, tenantID, tenantID, tenantID,
		nullStr(projectID), nullStr(userID), nullStr(workspaceID), nullStr(taskID),
		unitID, desc, txnID, now, nullInt64(hit.OrderID), nullInt64(hit.GrantID),
	)
	if err != nil {
		return nil, fmt.Errorf("记录任务帖配额消费流水失败: %w", err)
	}

	// Step 3: 记录 billing_usage
	usageID := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO billing_usage (
			id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
			description, usage_time
		) VALUES (?, (SELECT id FROM billing_account WHERE tenant_id = ?), ?, 1,
			?, ?, ?, ?, ?, ?)`,
		usageID, tenantID, unitID,
		nullStr(projectID), nullStr(userID), nullStr(workspaceID), nullStr(taskID),
		desc, now,
	)
	if err != nil {
		return nil, fmt.Errorf("记录任务帖配额用量失败: %w", err)
	}

	// Step 4: 记录 outbox 消息（用于 Kafka 事件发布）
	payload := map[string]interface{}{
		"transaction_id":     txnID,
		"account_id":         fmt.Sprintf("%d", acc.ID),
		"amount_points":      "0",
		"amount_yuan":        "0.00",
		"points_source_type": "quota_consumption",
		"transaction_type":   "consumption",
		"resource_type":      "task_post_quota",
		"task_id":            taskID,
		"source_kind":        hit.SourceKind,
		"order_id":           formatID(hit.OrderID),
		"source_grant_id":    formatID(hit.GrantID),
		"created_at":         now,
	}
	if trid := tracelog.TraceIDFromContext(ctx); trid != "" {
		payload["trace_id"] = trid
	}
	raw, _ := json.Marshal(payload)
	outboxID := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO billing_outbox_message (id, billing_transaction_id, event_type, payload, status, created_at)
		VALUES (?, ?, 'BILLING_TRANSACTION_CREATED', ?, 'pending', ?)`,
		outboxID, tid, string(raw), now)
	if err != nil {
		return nil, fmt.Errorf("记录 outbox 失败: %w", err)
	}

	// Step 5: 记录幂等性键，防重复消耗
	ikID := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO billing_idempotency_key (id, `+"`"+`key`+"`"+`, created_at, transaction_id)
		VALUES (?, ?, ?, ?)`, ikID, idempotencyKey, now, tid)
	if err != nil {
		return nil, fmt.Errorf("记录幂等键失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交配额消耗事务失败: %w", err)
	}

	tracelog.LogForwardStage(ctx, "task_post_quota_consumed", map[string]any{
		"tenant_id":   formatID(tenantID),
		"task_id":     taskID,
		"account_id":  formatID(acc.ID),
		"cost_cents":  0,
		"quota_after": getTaskPostQuotaAfter(tenantID),
	})

	// 异步发送 outbox 到 Django → Kafka
	go djangoEmitBillingEvent(ctx, outboxID, payload)

	return map[string]interface{}{
		"success":           true,
		"transaction_id":    txnID,
		"account_id":        formatID(acc.ID),
		"cost_cents":        0,
		"quota_consumed":    1,
		"transaction_db_id": formatID(tid),
		// 🆕 帖子存续期: 创建帖消耗 1 创建帖次数 → 12 个月存续期
		"expires_at": taskPostExpiresAt(now, ""),
	}, nil
}

// getTaskPostQuotaAfter 查询当前剩余配额（事务提交后调用）
func getTaskPostQuotaAfter(tenantID int64) int64 {
	var quota int64
	if err := db.QueryRow(
		`SELECT COALESCE(task_post_quota, 0) FROM billing_account WHERE tenant_id = ?`, tenantID,
	).Scan(&quota); err != nil {
		return -1
	}
	return quota
}

// getTenantCumulativeConsumptionCents 查询租户累计核销消费金额（分）
// 计算 billing_transaction 中所有 consumption 类型交易的 amount 总和
func getTenantCumulativeConsumptionCents(tenantID int64) (int64, error) {
	var total sql.NullInt64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(bt.amount), 0)
		FROM billing_transaction bt
		JOIN billing_account ba ON bt.account_id = ba.id
		WHERE ba.tenant_id = ? AND bt.transaction_type = 'consumption'`,
		tenantID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}
