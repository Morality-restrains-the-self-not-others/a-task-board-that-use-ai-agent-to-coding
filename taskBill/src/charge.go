package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"tracelog"
)

type InsufficientBalanceError struct {
	BalancePoints  int64
	RequiredPoints int64
}

func (e *InsufficientBalanceError) Error() string {
	return fmt.Sprintf(
		"资源配额不足，当前剩余 %d，本次需 %d，请先购买资源",
		e.BalancePoints, e.RequiredPoints,
	)
}

func ensureBillingUnit(unitType, name string, defaultPrice int64) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM billing_unit WHERE unit_type = ?`, unitType).Scan(&id)
	if err == sql.ErrNoRows {
		id = generateSnowflakeID()
		now := utcNow()
		_, err = db.Exec(`
			INSERT INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, '次', 1, ?, ?)`, id, unitType, name, defaultPrice, now, now)
	} else if err == nil {
		_, err = db.Exec(`UPDATE billing_unit SET name = ? WHERE id = ?`, name, id)
	}
	return id, err
}

func recordConsumption(
	ctx context.Context,
	acc *BillingAccount,
	unitID int64,
	cost int64,
	txnID string,
	idempotencyKey string,
	desc string,
	taskID, userID, workspaceID, projectID string,
) (map[string]interface{}, error) {
	return recordConsumptionUsage(ctx, acc, unitID, cost, 1, txnID, idempotencyKey, desc, taskID, userID, workspaceID, projectID)
}

// recordConsumptionUsageTx 在给定事务中记录消费流水（不自行管理事务边界）。
// 调用方负责 tx.Commit()；失败时调用方应 tx.Rollback()。
func recordConsumptionUsageTx(
	ctx context.Context,
	tx *sql.Tx,
	acc *BillingAccount,
	unitID int64,
	cost int64,
	usageAmount float64,
	txnID string,
	idempotencyKey string,
	desc string,
	taskID, userID, workspaceID, projectID string,
) (map[string]interface{}, error) {
	if usageAmount <= 0 {
		usageAmount = 1
	}

	var existing int64
	if idempotencyKey != "" {
		err := tx.QueryRow(`SELECT transaction_id FROM billing_idempotency_key WHERE `+"`"+`key`+"`"+` = ?`, idempotencyKey).Scan(&existing)
		if err == nil {
			return map[string]interface{}{"idempotent": true, "transaction_id": txnID}, nil
		}
	}

	var balance int64
	err := tx.QueryRow(`SELECT balance FROM billing_account WHERE id = ?`, acc.ID).Scan(&balance)
	if err != nil {
		return nil, err
	}
	if balance < cost {
		return nil, &InsufficientBalanceError{balance, cost}
	}
	after := balance - cost
	now := utcNow()
	_, err = tx.Exec(`UPDATE billing_account SET balance = ?, updated_at = ? WHERE id = ?`, after, now, acc.ID)
	if err != nil {
		return nil, err
	}
	if err := consumePaymentLedgerFEFO(ctx, tx, acc.TenantID, cost); err != nil {
		return nil, err
	}

	tid := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, project_id, user_id, workspace_id, task_id,
			billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', ?, ?, ?, 'consumption', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tid, acc.ID, cost, balance, after,
		nullStr(projectID), nullStr(userID), nullStr(workspaceID), nullStr(taskID),
		unitID, usageAmount, desc, txnID, now,
	)
	if err != nil {
		return nil, err
	}

	usageID := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO billing_usage (
			id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
			description, usage_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usageID, acc.ID, unitID, usageAmount,
		nullStr(projectID), nullStr(userID), nullStr(workspaceID), nullStr(taskID),
		desc, now,
	)
	if err != nil {
		return nil, err
	}

	if idempotencyKey != "" {
		ikID := generateSnowflakeID()
		_, err = tx.Exec(`
			INSERT INTO billing_idempotency_key (id, `+"`"+`key`+"`"+`, created_at, transaction_id)
			VALUES (?, ?, ?, ?)`, ikID, idempotencyKey, now, tid)
		if err != nil {
			return nil, err
		}
	}

	payload := map[string]interface{}{
		"transaction_id":     txnID,
		"account_id":         fmt.Sprintf("%d", acc.ID),
		"amount_points":      fmt.Sprintf("%d", cost),
		"amount_yuan":        pointsToYuanEquivalentStr(cost),
		"points_source_type": "consumption",
		"transaction_type":   "consumption",
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
		return nil, err
	}

	// 后置异步任务在事务提交后执行，由调用方负责
	return map[string]interface{}{
		"transaction_db_id": formatID(tid),
		"balance_after":     after,
		"cost":              cost,
		"_tid":              tid,
		"_outbox_id":        outboxID,
		"_payload":          payload,
		"_user_id":          userID,
		"_txn_id":           txnID,
	}, nil
}

func recordConsumptionUsage(
	ctx context.Context,
	acc *BillingAccount,
	unitID int64,
	cost int64,
	usageAmount float64,
	txnID string,
	idempotencyKey string,
	desc string,
	taskID, userID, workspaceID, projectID string,
) (map[string]interface{}, error) {
	if usageAmount <= 0 {
		usageAmount = 1
	}
	if err := expireCreditLots(ctx, acc.TenantID); err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := recordConsumptionUsageTx(
		ctx, tx, acc, unitID, cost, usageAmount, txnID, idempotencyKey, desc,
		taskID, userID, workspaceID, projectID,
	)
	if err != nil {
		return nil, err
	}
	if idempotent, _ := result["idempotent"].(bool); idempotent {
		_ = tx.Commit()
		return result, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 事务已提交，触发异步后置任务
	payload, _ := result["_payload"].(map[string]interface{})
	outboxID, _ := result["_outbox_id"].(int64)
	resUserID, _ := result["_user_id"].(string)
	resTxnID, _ := result["_txn_id"].(string)
	resTid, _ := result["_tid"].(int64)

	go djangoEmitBillingEvent(ctx, outboxID, payload)
	if resUserID != "" {
		go tryAccrueReferralFromConsumption(
			context.Background(),
			resUserID,
			resTxnID,
			resTid,
			referralAccrualPointsFromChargeResult(result),
			utcNow(),
		)
	}

	delete(result, "_tid")
	delete(result, "_outbox_id")
	delete(result, "_payload")
	delete(result, "_user_id")
	delete(result, "_txn_id")
	return result, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// referralAccrualPointsFromChargeResult reads consumption points from a
// recordConsumptionUsage result. Passing 0 skips accrual.
func referralAccrualPointsFromChargeResult(result map[string]interface{}) int64 {
	if result == nil {
		return 0
	}
	switch v := result["cost"].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// chargeServerStart 服务器启动计费 — v15 剥离为 no-op。
// 定价模型重构后：帖子按「创建帖次数 + 12 个月存续期」计费，
// 存续期内的服务器启动不再按次扣费（帖子有效期校验在 taskTaskService 侧）。
// 保留接口以兼容历史调用方（taskCloudService bill_client 已 dormant），
// 随迁移期后删除。返回幂等成功。
func chargeServerStart(ctx context.Context, tenantID int64, cloudEventID, taskID, workspaceID, userID, projectID string) (map[string]interface{}, error) {
	_ = expireCreditLots(ctx, tenantID)
	_, _, _ = getOrCreateBillingAccount(tenantID, true)
	tracelog.LogForwardStage(ctx, "charge_server_start_noop", map[string]any{
		"tenant_id": formatID(tenantID),
		"task_id":   taskID,
	})
	return map[string]interface{}{
		"success":        true,
		"cost_cents":     0,
		"noop":           true,
		"reason":         "post_validity_model: 服务器启动不再按次扣费（v15 存续期模式）",
		"transaction_id": fmt.Sprintf("server_start_noop:%s", cloudEventID),
	}, nil
}

func checkBalanceForServerStart(tenantID int64) error {
	if err := expireCreditLots(context.Background(), tenantID); err != nil {
		return err
	}
	_, _, err := getOrCreateBillingAccount(tenantID, false)
	return err
}

// chargeGitlabTraffic records outbound GitLab traffic against prepaid quota.
// It does not debit billing_account.balance; tenants have no spendable cash.
// Usage is actual GB (not Ceil to 1) so git-upload-pack written_bytes can raise 已用流量
// without immediately exhausting a 1 GB prepaid pack.
func chargeGitlabTraffic(
	ctx context.Context,
	tenantID int64,
	gb float64,
	isIntranet bool,
	idempotencyKey, projectID, userID, workspaceID, taskID, region string,
) (map[string]interface{}, error) {
	return chargeGitlabTrafficFromMeter(
		ctx, tenantID, gb, isIntranet, idempotencyKey, projectID, userID, workspaceID, taskID, region, "",
	)
}

var errInvalidTenant = errors.New("invalid tenant_id")
