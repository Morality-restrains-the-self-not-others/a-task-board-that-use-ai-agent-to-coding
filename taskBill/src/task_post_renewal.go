package main

import (
	"context"
	"encoding/json"
	"fmt"

	"tracelog"
)

// taskPostExpiresAt 计算任务帖存续到期时间。
// 创建（currentExpiresAt 为空）: now + 12 个月
// 续存（currentExpiresAt 非空）: max(now, currentExpiresAt) + 12 个月（不缩短已购时长）
// 与 gitlab_resources.go diskExpiresAtFromMonths 同款算法。
const taskPostValidityMonths int64 = 12

func taskPostExpiresAt(now string, currentExpiresAt string) string {
	return diskExpiresAtFromMonths(taskPostValidityMonths, currentExpiresAt)
}

// consumeTaskPostRenewal 续存任务帖：只消耗 1 个创建帖次数（FEFO），不扣钱包、不按续存单价计费。
// 幂等键 task_post_renewal:{task_id}；返回新到期时间供 taskTaskService 写入。
// 审计流水挂在价为 0 的 task_post_quota 单位上（与创建帖消耗配额同模式）。
func consumeTaskPostRenewal(ctx context.Context, tenantID int64, taskID, workspaceID, userID, projectID, currentExpiresAt, idempotencyKey string) (map[string]interface{}, error) {
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		return nil, err
	}

	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("task_post_renewal:%s", taskID)
	}
	txnID := fmt.Sprintf("task_post_renewal:%s", taskID)
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
		_ = tx.Commit()
		return map[string]interface{}{
			"success":        true,
			"idempotent":     true,
			"transaction_id": txnID,
			"account_id":     formatID(acc.ID),
			"cost_cents":     0,
			"quota_consumed": 0,
			"expires_at":     taskPostExpiresAt(now, currentExpiresAt),
		}, nil
	}

	// Step 1: 原子消耗 1 个创建帖次数
	hit, err := consumeTaskPostQuotaLotTx(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}

	// Step 2: 记录消费流水（cost=0，次数已预购支付）
	desc := fmt.Sprintf("续存任务帖 (task_id=%s)", taskID)
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
		return nil, fmt.Errorf("记录续存流水失败: %w", err)
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
		return nil, fmt.Errorf("记录续存用量失败: %w", err)
	}

	// Step 4: outbox 消息（Kafka 事件发布）
	payload := map[string]interface{}{
		"transaction_id":     txnID,
		"account_id":         fmt.Sprintf("%d", acc.ID),
		"amount_points":      "0",
		"amount_yuan":        "0.00",
		"points_source_type": "quota_consumption",
		"transaction_type":   "consumption",
		"resource_type":      "task_post_renewal",
		"task_id":            taskID,
		"source_kind":        hit.SourceKind,
		"order_id":           formatID(hit.OrderID),
		"source_grant_id":    formatID(hit.GrantID),
		"expires_at":         taskPostExpiresAt(now, currentExpiresAt),
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

	// Step 5: 幂等键
	ikID := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO billing_idempotency_key (id, `+"`"+`key`+"`"+`, created_at, transaction_id)
		VALUES (?, ?, ?, ?)`, ikID, idempotencyKey, now, tid)
	if err != nil {
		return nil, fmt.Errorf("记录幂等键失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交续存事务失败: %w", err)
	}

	tracelog.LogForwardStage(ctx, "task_post_renewed", map[string]any{
		"tenant_id":   formatID(tenantID),
		"task_id":     taskID,
		"account_id":  formatID(acc.ID),
		"cost_cents":  0,
		"quota_after": getTaskPostQuotaAfter(tenantID),
	})

	go djangoEmitBillingEvent(ctx, outboxID, payload)

	return map[string]interface{}{
		"success":           true,
		"transaction_id":    txnID,
		"account_id":        formatID(acc.ID),
		"cost_cents":        0,
		"quota_consumed":    1,
		"transaction_db_id": formatID(tid),
		"expires_at":        taskPostExpiresAt(now, currentExpiresAt),
	}, nil
}
