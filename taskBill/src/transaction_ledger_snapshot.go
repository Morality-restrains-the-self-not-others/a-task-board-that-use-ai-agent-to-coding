package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

func formatCashSnapshotLine(before, after int64) string {
	if before == after {
		return fmt.Sprintf("余额 %s 元", centsToYuanStr(after))
	}
	return fmt.Sprintf("余额 %s → %s 元", centsToYuanStr(before), centsToYuanStr(after))
}

func attachLedgerSnapshots(ctx context.Context, tenantID, accountID int64, list []map[string]interface{}) {
	if len(list) == 0 {
		return
	}
	remainingByID := replayTaskPostRemaining(ctx, tenantID, accountID, list)
	for _, row := range list {
		bb := jsonNumberAsInt64(row["balance_before_points"])
		ba := jsonNumberAsInt64(row["balance_after_points"])
		cashLine := formatCashSnapshotLine(bb, ba)
		lines := []string{cashLine}
		id := fmt.Sprint(row["id"])
		if rem, ok := remainingByID[id]; ok {
			lines = append(lines, fmt.Sprintf("任务帖剩余 %d", rem))
			annotateTaskPostRemaining(row, rem)
		}
		row["ledger_snapshot"] = map[string]interface{}{
			"balance_before_points": bb,
			"balance_after_points":  ba,
			"balance_before_yuan":   centsToYuanStr(bb),
			"balance_after_yuan":    centsToYuanStr(ba),
			"display":               strings.Join(lines, "；"),
			"display_lines":         lines,
		}
	}
	// OPT-20260819-014: GitLab 磁盘/流量按 region 回放瞬时配额剩余。
	// 独立调用而非并入上式，避免 task_post 回放与 GitLab 回放相互干扰。
	attachGitlabRemainingSnapshot(ctx, tenantID, accountID, list)
}

func annotateTaskPostRemaining(row map[string]interface{}, remaining int64) {
	changes, ok := row["resource_changes"].([]map[string]interface{})
	if !ok {
		return
	}
	for _, c := range changes {
		if fmt.Sprint(c["resource_type"]) == ResourceTypeTaskPost {
			c["remaining_after"] = remaining
		}
	}
}

type replayTxn struct {
	id               string
	pointsSourceType string
	transactionID    string
	usageAmount      float64
	createdAt        string
}

func replayTaskPostRemaining(ctx context.Context, tenantID, accountID int64, list []map[string]interface{}) map[string]int64 {
	out := map[string]int64{}
	oldest := ""
	for _, row := range list {
		sec := createdAtSecond(fmt.Sprint(row["created_at"]))
		if sec == "" {
			continue
		}
		if oldest == "" || sec < oldest {
			oldest = sec
		}
	}
	if oldest == "" {
		return out
	}
	current, err := getTaskPostQuotaRemaining(tenantID)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_quota_remaining_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return out
	}

	rows, err := db.Query(`
		SELECT id, COALESCE(points_source_type, ''), COALESCE(transaction_id, ''), usage_amount, created_at
		FROM billing_transaction
		WHERE account_id = ? AND created_at >= ?
		ORDER BY created_at DESC, id DESC`,
		accountID, oldest)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_quota_replay_query_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return out
	}
	defer rows.Close()

	var history []replayTxn
	orderNumbers := []string{}
	seenOrder := map[string]struct{}{}
	for rows.Next() {
		var id int64
		var src, txnID, created string
		var usage float64
		if err := rows.Scan(&id, &src, &txnID, &usage, &created); err != nil {
			continue
		}
		item := replayTxn{
			id:               formatID(id),
			pointsSourceType: src,
			transactionID:    txnID,
			usageAmount:      usage,
			createdAt:        created,
		}
		history = append(history, item)
		if src == "resource_purchase" {
			if num, ok := orderNumberFromTxnID(txnID); ok {
				if _, dup := seenOrder[num]; !dup {
					seenOrder[num] = struct{}{}
					orderNumbers = append(orderNumbers, num)
				}
			}
		}
	}

	grantQty := loadTaskPostGrantQtyBySecond(ctx, tenantID, oldest)
	purchaseQty := loadTaskPostQtyByOrderNumbers(ctx, tenantID, orderNumbers)

	quota := current
	for _, item := range history {
		out[item.id] = quota
		quota -= taskPostDelta(item, grantQty, purchaseQty)
	}
	return out
}

func taskPostDelta(item replayTxn, grantQty map[string]int64, purchaseQty map[string]int64) int64 {
	switch item.pointsSourceType {
	case "quota_consumption", resourceGrantExpiryPointsSourceType:
		return -int64(item.usageAmount)
	case "admin_grant":
		return grantQty[createdAtSecond(item.createdAt)]
	case "resource_purchase":
		if num, ok := orderNumberFromTxnID(item.transactionID); ok {
			return purchaseQty[num]
		}
	}
	return 0
}

func orderNumberFromTxnID(txnID string) (string, bool) {
	const prefix = "order:"
	if !strings.HasPrefix(txnID, prefix) {
		return "", false
	}
	num := strings.TrimSpace(strings.TrimPrefix(txnID, prefix))
	if num == "" {
		return "", false
	}
	return num, true
}

func loadTaskPostGrantQtyBySecond(ctx context.Context, tenantID int64, oldest string) map[string]int64 {
	out := map[string]int64{}
	rows, err := db.Query(`
		SELECT created_at, quantity
		FROM billing_resource_grant
		WHERE tenant_id = ? AND resource_type = ? AND created_at >= ?`,
		tenantID, ResourceTypeTaskPost, oldest)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_grant_qty_load_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var created string
		var qty int64
		if err := rows.Scan(&created, &qty); err != nil {
			continue
		}
		out[createdAtSecond(created)] += qty
	}
	return out
}

func loadTaskPostQtyByOrderNumbers(ctx context.Context, tenantID int64, orderNumbers []string) map[string]int64 {
	out := map[string]int64{}
	if len(orderNumbers) == 0 {
		return out
	}
	placeholders := strings.Repeat("?,", len(orderNumbers))
	placeholders = strings.TrimSuffix(placeholders, ",")
	args := make([]interface{}, 0, len(orderNumbers)+2)
	args = append(args, tenantID, ResourceTypeTaskPost)
	for _, n := range orderNumbers {
		args = append(args, n)
	}
	q := `
		SELECT o.order_number, COALESCE(SUM(i.quantity), 0)
		FROM billing_resource_order o
		JOIN billing_resource_order_item i ON i.order_id = o.id
		WHERE o.tenant_id = ? AND i.resource_type = ? AND o.order_number IN (` + placeholders + `)
		GROUP BY o.order_number`
	rows, err := db.Query(q, args...)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_purchase_qty_load_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var num string
		var qty int64
		if err := rows.Scan(&num, &qty); err != nil {
			continue
		}
		out[num] = qty
	}
	return out
}
