package main

// OPT-20260819-014: 交易流水账按 region 回放 GitLab 磁盘/流量「瞬时配额剩余」。
//
// 与 task_post 剩余回放（replayTaskPostRemaining）同一思路：
// 以当前每区配额（disk_gb / traffic_prepaid_gb）为起点，沿时间倒退逐笔扣减
// 交易增量（资源购买 / 后台赠送），从而还原「该笔交易发生时该区还剩多少配额」。
// 用量（disk_used_gb / traffic_used_gb）非事务化、由计量持续写入，无法按交易回放，
// 因此展示口径为「重放得到的配额总量 − 当前用量」（瞬时口径，与设置页一致）。

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// gitlabRegionRemainingLine 是流水单行要追加的「区域配额剩余」展示数据。
// 一条记录对应一笔交易触及的某个资源类型（磁盘或流量）。
type gitlabRegionRemainingLine struct {
	Region       string
	ResourceType string
	RemainingGB  float64
}

// gitlabRegionQuota 是某租户某区域当前配额与用量快照（回放起点）。
type gitlabRegionQuota struct {
	Region           string
	DiskGB           int64
	TrafficPrepaidGB int64
	DiskUsedGB       float64
	TrafficUsedGB    float64
}

// gitlabRegionDelta 是一笔交易对某区某资源类型配额的增量。
type gitlabRegionDelta struct {
	Region       string
	ResourceType string
	Quantity     int64
}

// loadGitlabRegionQuotaSnapshot 读取租户当前各区域 GitLab 配额与用量。
// 无任何 GitLab 购买/赠送记录时返回空 map（调用方直接跳过回放）。
func loadGitlabRegionQuotaSnapshot(tenantID int64) (map[string]*gitlabRegionQuota, error) {
	rows, err := db.Query(`
		SELECT region, disk_gb, traffic_prepaid_gb, disk_used_bytes, traffic_used_gb
		FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND (disk_gb > 0 OR traffic_prepaid_gb > 0)
		ORDER BY region`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]*gitlabRegionQuota{}
	for rows.Next() {
		var region string
		var diskGB, trafficPrepaidGB, diskUsedBytes int64
		var trafficUsedGB float64
		if err := rows.Scan(&region, &diskGB, &trafficPrepaidGB, &diskUsedBytes, &trafficUsedGB); err != nil {
			return nil, err
		}
		out[region] = &gitlabRegionQuota{
			Region:           region,
			DiskGB:           diskGB,
			TrafficPrepaidGB: trafficPrepaidGB,
			DiskUsedGB:       diskUsedGBFromBytes(diskUsedBytes),
			TrafficUsedGB:    trafficUsedGB,
		}
	}
	return out, rows.Err()
}

// loadGitlabOrderDeltasByOrderID 加载订单行项中 gitlab_disk / gitlab_traffic 的区域增量。
// 键为订单主键（billing_transaction.related_order_id）。
func loadGitlabOrderDeltasByOrderID(ctx context.Context, orderIDs []int64) map[int64][]gitlabRegionDelta {
	out := map[int64][]gitlabRegionDelta{}
	if len(orderIDs) == 0 {
		return out
	}
	rows, err := db.Query(`
		SELECT order_id, resource_type, quantity, region
		FROM billing_resource_order_item
		WHERE order_id IN (`+placeholders(len(orderIDs))+`)
		  AND resource_type IN (?, ?)
		  AND region <> ''`,
		append(anySlice(orderIDs), ResourceTypeGitlabDisk, ResourceTypeGitlabTraffic)...)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_gitlab_order_delta_failed",
			"level", "warn", "error", err.Error())
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int64
		var resourceType, region string
		var quantity int64
		if err := rows.Scan(&orderID, &resourceType, &quantity, &region); err != nil {
			continue
		}
		out[orderID] = append(out[orderID], gitlabRegionDelta{
			Region: region, ResourceType: resourceType, Quantity: quantity,
		})
	}
	return out
}

// loadGitlabGrantDeltasByTxnID 加载后台赠送（admin_grant）中 GitLab 区域增量，键为流水主键。
func loadGitlabGrantDeltasByTxnID(ctx context.Context, fkIDs []int64) map[int64][]gitlabRegionDelta {
	out := map[int64][]gitlabRegionDelta{}
	if len(fkIDs) == 0 {
		return out
	}
	rows, err := db.Query(`
		SELECT billing_transaction_id, resource_type, quantity, region
		FROM billing_resource_grant
		WHERE billing_transaction_id IN (`+placeholders(len(fkIDs))+`)
		  AND resource_type IN (?, ?)
		  AND region <> ''`,
		append(anySlice(fkIDs), ResourceTypeGitlabDisk, ResourceTypeGitlabTraffic)...)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_gitlab_grant_delta_failed",
			"level", "warn", "error", err.Error())
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var txnID int64
		var resourceType, region string
		var quantity int64
		if err := rows.Scan(&txnID, &resourceType, &quantity, &region); err != nil {
			continue
		}
		out[txnID] = append(out[txnID], gitlabRegionDelta{
			Region: region, ResourceType: resourceType, Quantity: quantity,
		})
	}
	return out
}

// loadGitlabGrantDeltasBySecond 秒级兜底：旧赠送行未写 billing_transaction_id 时，
// 按 created_at 秒关联（与任务帖 grant 回放口径一致）。
func loadGitlabGrantDeltasBySecond(ctx context.Context, tenantID int64, oldest string) map[string][]gitlabRegionDelta {
	out := map[string][]gitlabRegionDelta{}
	if oldest == "" {
		return out
	}
	rows, err := db.Query(`
		SELECT created_at, resource_type, quantity, region
		FROM billing_resource_grant
		WHERE tenant_id = ?
		  AND billing_transaction_id IS NULL
		  AND resource_type IN (?, ?)
		  AND region <> ''
		  AND created_at >= ?`,
		tenantID, ResourceTypeGitlabDisk, ResourceTypeGitlabTraffic, oldest)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_gitlab_grant_second_failed",
			"level", "warn", "error", err.Error())
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var created, resourceType, region string
		var quantity int64
		if err := rows.Scan(&created, &resourceType, &quantity, &region); err != nil {
			continue
		}
		sec := createdAtSecond(created)
		out[sec] = append(out[sec], gitlabRegionDelta{
			Region: region, ResourceType: resourceType, Quantity: quantity,
		})
	}
	return out
}

type gitlabReplayTxn struct {
	id        string
	src       string
	orderID   int64
	createdAt string
}

// replayGitlabRegionRemaining 对传入的流水行（list）按 region 回放 GitLab 配额剩余。
// 返回 map[transactionID][]gitlabRegionRemainingLine —— 仅包含触及 GitLab 配额的行。
func replayGitlabRegionRemaining(ctx context.Context, tenantID, accountID int64, list []map[string]interface{}) map[string][]gitlabRegionRemainingLine {
	out := map[string][]gitlabRegionRemainingLine{}
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

	quota, err := loadGitlabRegionQuotaSnapshot(tenantID)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_gitlab_quota_load_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return out
	}
	if len(quota) == 0 {
		return out
	}

	rows, err := db.Query(`
		SELECT id, COALESCE(points_source_type, ''), COALESCE(related_order_id, 0), created_at
		FROM billing_transaction
		WHERE account_id = ? AND created_at >= ?
		ORDER BY created_at DESC, id DESC`,
		accountID, oldest)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_gitlab_replay_query_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return out
	}
	defer rows.Close()

	var history []gitlabReplayTxn
	orderIDs := []int64{}
	seenOrder := map[int64]struct{}{}
	fkIDs := []int64{}
	for rows.Next() {
		var id int64
		var src string
		var orderID int64
		var created string
		if err := rows.Scan(&id, &src, &orderID, &created); err != nil {
			continue
		}
		item := gitlabReplayTxn{id: formatID(id), src: src, orderID: orderID, createdAt: created}
		history = append(history, item)
		if src == "resource_purchase" && orderID > 0 {
			if _, dup := seenOrder[orderID]; !dup {
				seenOrder[orderID] = struct{}{}
				orderIDs = append(orderIDs, orderID)
			}
		}
		if src == "admin_grant" {
			if v, err := parseIDField(item.id); err == nil && v > 0 {
				fkIDs = append(fkIDs, v)
			}
		}
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "billing_txn_gitlab_replay_scan_failed",
			"level", "warn", "error", err.Error())
		return out
	}

	orderDeltas := loadGitlabOrderDeltasByOrderID(ctx, orderIDs)
	grantDeltas := loadGitlabGrantDeltasByTxnID(ctx, fkIDs)
	grantBySecond := loadGitlabGrantDeltasBySecond(ctx, tenantID, oldest)

	for _, item := range history {
		var deltas []gitlabRegionDelta
		switch item.src {
		case "resource_purchase":
			if item.orderID > 0 {
				deltas = orderDeltas[item.orderID]
			}
		case "admin_grant":
			if v, err := parseIDField(item.id); err == nil && v > 0 {
				deltas = grantDeltas[v]
			}
			if len(deltas) == 0 {
				deltas = grantBySecond[createdAtSecond(item.createdAt)]
			}
		}
		if len(deltas) == 0 {
			continue
		}
		lines := snapshotGitlabRemaining(quota, deltas)
		if len(lines) > 0 {
			out[item.id] = lines
		}
		for _, d := range deltas {
			q := quota[d.Region]
			if q == nil {
				continue
			}
			switch d.ResourceType {
			case ResourceTypeGitlabDisk:
				q.DiskGB -= d.Quantity
			case ResourceTypeGitlabTraffic:
				q.TrafficPrepaidGB -= d.Quantity
			}
		}
	}
	return out
}

// snapshotGitlabRemaining 取当前（重放中）配额快照减去当前用量，生成该笔交易
// 实际触及的资源类型展示行（一笔交易只展示它改动的磁盘/流量）。
func snapshotGitlabRemaining(quota map[string]*gitlabRegionQuota, deltas []gitlabRegionDelta) []gitlabRegionRemainingLine {
	var lines []gitlabRegionRemainingLine
	for _, d := range deltas {
		q := quota[d.Region]
		if q == nil {
			continue
		}
		var remaining float64
		switch d.ResourceType {
		case ResourceTypeGitlabDisk:
			remaining = float64(q.DiskGB) - q.DiskUsedGB
		case ResourceTypeGitlabTraffic:
			remaining = float64(q.TrafficPrepaidGB) - q.TrafficUsedGB
		default:
			continue
		}
		lines = append(lines, gitlabRegionRemainingLine{
			Region:       d.Region,
			ResourceType: d.ResourceType,
			RemainingGB:  remaining,
		})
	}
	return lines
}

// formatGitlabRemainingLine 生成「GitLab 磁盘剩余 X GB（region）」展示文案。
func formatGitlabRemainingLine(resourceType string, remaining float64, region string) string {
	label := resourceTypeLabelZH(resourceType)
	val := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", remaining), "0"), ".")
	if val == "" || val == "-" {
		val = "0"
	}
	line := fmt.Sprintf("%s剩余 %s GB", label, val)
	if region != "" {
		line += "（" + region + "）"
	}
	return line
}

// attachGitlabRemainingSnapshot 把按 region 重放的 GitLab 配额剩余写入
// ledger_snapshot.display_lines 与独立字段 gitlab_region_remaining。
func attachGitlabRemainingSnapshot(ctx context.Context, tenantID, accountID int64, list []map[string]interface{}) {
	if len(list) == 0 {
		return
	}
	remainingByID := replayGitlabRegionRemaining(ctx, tenantID, accountID, list)
	if len(remainingByID) == 0 {
		return
	}
	for _, row := range list {
		lines, ok := remainingByID[fmt.Sprint(row["id"])]
		if !ok || len(lines) == 0 {
			continue
		}
		payload := make([]map[string]interface{}, 0, len(lines))
		for _, l := range lines {
			payload = append(payload, map[string]interface{}{
				"region":        l.Region,
				"resource_type": l.ResourceType,
				"remaining_gb":  l.RemainingGB,
				"display_line":  formatGitlabRemainingLine(l.ResourceType, l.RemainingGB, l.Region),
			})
		}
		row["gitlab_region_remaining"] = payload
		snap, _ := row["ledger_snapshot"].(map[string]interface{})
		if snap == nil {
			snap = map[string]interface{}{}
			row["ledger_snapshot"] = snap
		}
		var displayLines []string
		if raw, ok := snap["display_lines"].([]string); ok {
			displayLines = raw
		}
		for _, l := range lines {
			displayLines = append(displayLines, formatGitlabRemainingLine(l.ResourceType, l.RemainingGB, l.Region))
		}
		snap["display_lines"] = displayLines
		snap["display"] = strings.Join(displayLines, "；")
	}
}
