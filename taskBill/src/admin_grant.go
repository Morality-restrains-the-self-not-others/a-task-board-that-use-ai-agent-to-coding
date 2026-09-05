package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"strings"
)

// ResourceGrantInput 单条资源赠送输入
type ResourceGrantInput struct {
	ResourceType string `json:"resource_type"`
	Quantity     int64  `json:"quantity"`
	ExpiresAt    string `json:"expires_at"`
	Reason       string `json:"reason"`
	Region       string `json:"region,omitempty"` // gitlab_disk / gitlab_traffic 必填，禁止静默默认区
}

// backfillGrantOrders 为已有 grants 但没有对应 admin_grant 订单的租户补建订单。
// 按租户聚合 grants，每个租户生成一个汇总订单。幂等：已有订单的租户跳过。
// 返回创建的订单数量。
func backfillGrantOrders() (int, error) {
	// 查找有 grants 但没有 admin_grant 订单的租户
	rows, err := db.Query(`
		SELECT DISTINCT g.tenant_id
		FROM billing_resource_grant g
		WHERE NOT EXISTS (
			SELECT 1 FROM billing_resource_order o
			WHERE o.tenant_id = g.tenant_id AND o.payment_method = 'admin_grant'
		)
	`)
	if err != nil {
		return 0, fmt.Errorf("查询待补建租户失败: %w", err)
	}
	defer rows.Close()

	var tenantIDs []int64
	for rows.Next() {
		var tid int64
		if err := rows.Scan(&tid); err != nil {
			return 0, err
		}
		tenantIDs = append(tenantIDs, tid)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	created := 0
	now := utcNow()
	for _, tid := range tenantIDs {
		// 查询该租户的所有 grants
		grantRows, err := db.Query(`
			SELECT resource_type, SUM(quantity) as total_qty, MAX(reason) as reason
			FROM billing_resource_grant
			WHERE tenant_id = ?
			GROUP BY resource_type`, tid)
		if err != nil {
			log.Printf("[backfillGrantOrders] tenant=%d 查询 grants 失败: %v", tid, err)
			continue
		}

		type grantSummary struct {
			resourceType string
			quantity     int64
			reason       string
		}
		var summaries []grantSummary
		for grantRows.Next() {
			var rt string
			var qty int64
			var reason sql.NullString
			if err := grantRows.Scan(&rt, &qty, &reason); err != nil {
				grantRows.Close()
				return created, err
			}
			r := ""
			if reason.Valid {
				r = reason.String
			}
			summaries = append(summaries, grantSummary{rt, qty, r})
		}
		grantRows.Close()

		if len(summaries) == 0 {
			continue
		}

		// 使用最早 grant 的 reason 作为订单描述
		reason := summaries[0].reason
		if reason == "" {
			reason = "历史赠送补建"
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("[backfillGrantOrders] tenant=%d 开启事务失败: %v", tid, err)
			continue
		}

		// 撞号重试与 createOrder 共用 insertOrderWithRetry，避免与用户下单并发窗口内 1062 直接跳过该租户
		orderID, orderNumber, err := insertOrderWithRetry(context.Background(), tx.Exec, tid, maxOrderInsertRetries, func(oid int64, onum string) (string, []any) {
			return `
				INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, payment_method, payment_ref, created_at, paid_at)
				VALUES (?, ?, ?, 'paid', 0, 'admin_grant', ?, ?, ?)`,
				[]any{oid, tid, onum, reason, now, now}
		})
		if err != nil {
			tx.Rollback()
			log.Printf("[backfillGrantOrders] tenant=%d 创建订单失败: %v", tid, err)
			continue
		}

		itemFailed := false
		for _, gs := range summaries {
			itemID := generateSnowflakeID()
			_, err = tx.Exec(`
				INSERT INTO billing_resource_order_item (id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, created_at)
				VALUES (?, ?, ?, ?, 0, 0, '', ?)`,
				itemID, orderID, gs.resourceType, gs.quantity, now)
			if err != nil {
				tx.Rollback()
				log.Printf("[backfillGrantOrders] tenant=%d 创建订单行项失败: %v", tid, err)
				itemFailed = true
				break
			}
		}
		if itemFailed {
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Printf("[backfillGrantOrders] tenant=%d 提交事务失败: %v", tid, err)
			continue
		}
		created++
		log.Printf("[backfillGrantOrders] tenant=%d order=%s created", tid, orderNumber)
	}

	return created, nil
}

// adminGrantResources 管理员/系统赠送资源（含资源类型、过期时间、原因，可同时送多种）。
// 在同一事务中完成：配额更新 + grant 记录 + billing_transaction + 订单记录。
// idempotencyKey 用于防止重复赠送（Kafka 重放或管理员重复提交），为空则跳过幂等检查。
func adminGrantResources(ctx context.Context, tenantID int64, grants []ResourceGrantInput, userID string, idempotencyKey string) (map[string]interface{}, error) {
	return adminGrantResourcesWithMembership(ctx, tenantID, grants, userID, idempotencyKey, "", "")
}

// adminGrantResourcesWithMembership 赠送资源并可在同一事务中修改会员等级。
func adminGrantResourcesWithMembership(ctx context.Context, tenantID int64, grants []ResourceGrantInput, userID string, idempotencyKey string, membershipTier string, membershipReason string) (map[string]interface{}, error) {
	membershipTier = strings.TrimSpace(membershipTier)
	if membershipTier != "" && membershipTier != MembershipTierNormal && membershipTier != MembershipTierVIP1 {
		return nil, fmt.Errorf("invalid membership tier: %s", membershipTier)
	}
	if len(grants) == 0 && membershipTier == "" {
		return nil, fmt.Errorf("至少需要一项资源赠送或指定会员等级")
	}

	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		return nil, err
	}

	now := utcNow()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txnID := fmt.Sprintf("admin_grant:%d:%d", tenantID, generateSnowflakeID())
	// 汇总流水主键提前生成，供 billing_resource_grant.billing_transaction_id 外键引用
	tid := generateSnowflakeID()

	// 幂等键检查：如果已存在相同的 idempotency_key，直接返回已有结果
	if idempotencyKey != "" {
		var existing string
		if err := tx.QueryRow(`SELECT transaction_id FROM billing_idempotency_key WHERE `+"`"+`key`+"`"+` = ?`, idempotencyKey).Scan(&existing); err == nil {
			return map[string]interface{}{
				"idempotent":     true,
				"transaction_id": existing,
				"tenant_id":      formatID(tenantID),
			}, nil
		}
	}
	grantResults := make([]map[string]interface{}, 0, len(grants))
	gitlabRegionsGranted := map[string]struct{}{}

	fromTier := MembershipTierNormal
	if prev, err := getMembership(tenantID); err == nil && prev != nil && prev.Tier != "" {
		fromTier = prev.Tier
	}
	if membershipTier != "" {
		if err := applyMembershipTierTx(tx, tenantID, membershipTier, now); err != nil {
			return nil, fmt.Errorf("更新会员等级失败: %w", err)
		}
	}

	for _, g := range grants {
		if g.Quantity <= 0 {
			return nil, fmt.Errorf("资源数量须 >= 1，收到: resource_type=%s quantity=%d", g.ResourceType, g.Quantity)
		}

		appliedRegion := strings.TrimSpace(g.Region)

		switch g.ResourceType {
		case ResourceTypeTaskPost:
			if _, err := tx.Exec(
				`UPDATE billing_account SET task_post_quota = task_post_quota + ?, updated_at = ? WHERE tenant_id = ?`,
				g.Quantity, now, tenantID,
			); err != nil {
				return nil, fmt.Errorf("更新任务帖配额失败: %w", err)
			}

		case ResourceTypeGitlabDisk:
			region, err := getGitlabRegionBySlug(g.Region)
			if err != nil {
				return nil, err
			}
			appliedRegion = region.Slug
			gitlabRegionsGranted[region.Slug] = struct{}{}
			diskMonths := int64(1)
			currentExpiresAt := g.ExpiresAt
			expiresAt := diskExpiresAtFromMonths(diskMonths, currentExpiresAt)
			if _, err := tx.Exec(`
				INSERT INTO billing_tenant_gitlab_resource (
					tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at, created_at, updated_at
				) VALUES (?, ?, ?, 0, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE
					disk_gb = disk_gb + VALUES(disk_gb),
					disk_months = VALUES(disk_months),
					disk_expires_at = VALUES(disk_expires_at),
					updated_at = VALUES(updated_at)`,
				tenantID, region.Slug, g.Quantity, diskMonths, expiresAt, now, now,
			); err != nil {
				return nil, fmt.Errorf("更新 GitLab 磁盘配额失败: %w", err)
			}
			slog.InfoContext(ctx, "admin_grant_gitlab_quota_applied",
				"level", "info",
				"tenant_id", formatID(tenantID),
				"resource_type", g.ResourceType,
				"region", region.Slug,
				"quantity", g.Quantity,
			)

		case ResourceTypeGitlabTraffic:
			region, err := getGitlabRegionBySlug(g.Region)
			if err != nil {
				return nil, err
			}
			appliedRegion = region.Slug
			gitlabRegionsGranted[region.Slug] = struct{}{}
			if _, err := tx.Exec(`
				INSERT INTO billing_tenant_gitlab_resource (
					tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at, created_at, updated_at
				) VALUES (?, ?, 0, ?, 0, '', ?, ?)
				ON DUPLICATE KEY UPDATE
					traffic_prepaid_gb = traffic_prepaid_gb + VALUES(traffic_prepaid_gb),
					updated_at = VALUES(updated_at)`,
				tenantID, region.Slug, g.Quantity, now, now,
			); err != nil {
				return nil, fmt.Errorf("更新 GitLab 流量配额失败: %w", err)
			}
			slog.InfoContext(ctx, "admin_grant_gitlab_quota_applied",
				"level", "info",
				"tenant_id", formatID(tenantID),
				"resource_type", g.ResourceType,
				"region", region.Slug,
				"quantity", g.Quantity,
			)

		default:
			return nil, fmt.Errorf("暂不支持的资源类型: %s", g.ResourceType)
		}

		grantID := generateSnowflakeID()
		expiresAt := g.ExpiresAt
		if expiresAt == "" {
			expiresAt = "2099-12-31 23:59:59"
		}
		if _, err := tx.Exec(
			`INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at, billing_transaction_id, source_kind, region)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			grantID, tenantID, g.ResourceType, g.Quantity, g.Quantity, g.Reason, expiresAt, now, tid, grantSourceGift, appliedRegion,
		); err != nil {
			return nil, fmt.Errorf("记录资源赠送失败: %w", err)
		}

		row := map[string]interface{}{
			"grant_id":      formatID(grantID),
			"resource_type": g.ResourceType,
			"quantity":      g.Quantity,
			"expires_at":    expiresAt,
			"reason":        g.Reason,
		}
		if appliedRegion != "" && (g.ResourceType == ResourceTypeGitlabDisk || g.ResourceType == ResourceTypeGitlabTraffic) {
			row["region"] = appliedRegion
		}
		grantResults = append(grantResults, row)
	}

	// 记录汇总交易流水（points_source_type=admin_grant）— 具体资源文案写入库，避免仅靠 enrich 回填
	desc := adminGrantTransactionDescription(grants, membershipTier, membershipReason)

	var quotaAfter int64
	if err := tx.QueryRow(
		`SELECT COALESCE(task_post_quota, 0) FROM billing_account WHERE tenant_id = ?`, tenantID,
	).Scan(&quotaAfter); err != nil {
		quotaAfter = 0
	}

	if _, err := tx.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, user_id, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'recharge', 0, 0, 0, 'admin_grant', ?, ?, ?, ?, 0)`,
		tid, acc.ID, nullStr(userID), desc, txnID, now,
	); err != nil {
		return nil, fmt.Errorf("记录交易流水失败: %w", err)
	}

	var orderID int64
	var orderNumber string
	if len(grants) > 0 {
		// 创建资源赠送对应的订单记录，使赠送资源在订单列表中可见
		// 撞号重试与 createOrder 共用 insertOrderWithRetry，避免与用户下单并发窗口内 1062 直接失败
		orderID, orderNumber, err = insertOrderWithRetry(ctx, tx.Exec, tenantID, maxOrderInsertRetries, func(oid int64, onum string) (string, []any) {
			return `
			INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, payment_method, created_at, paid_at)
			VALUES (?, ?, ?, 'paid', 0, 'admin_grant', ?, ?)`,
				[]any{oid, tenantID, onum, now, now}
		})
		if err != nil {
			return nil, fmt.Errorf("创建赠送订单记录失败: %w", err)
		}
		for _, g := range grants {
			itemID := generateSnowflakeID()
			itemRegion := ""
			if g.ResourceType == ResourceTypeGitlabDisk || g.ResourceType == ResourceTypeGitlabTraffic {
				itemRegion = strings.TrimSpace(g.Region)
			}
			if _, err := tx.Exec(`
			INSERT INTO billing_resource_order_item (id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, created_at)
			VALUES (?, ?, ?, ?, 0, 0, ?, ?)`,
				itemID, orderID, g.ResourceType, g.Quantity, itemRegion, now,
			); err != nil {
				return nil, fmt.Errorf("创建赠送订单行项失败: %w", err)
			}
		}
		if _, err := tx.Exec(`UPDATE billing_transaction SET related_order_id = ? WHERE id = ?`, orderID, tid); err != nil {
			return nil, fmt.Errorf("关联赠送订单失败: %w", err)
		}
		if _, err := tx.Exec(`UPDATE billing_resource_grant SET order_id = ? WHERE billing_transaction_id = ? AND tenant_id = ?`, orderID, tid, tenantID); err != nil {
			return nil, fmt.Errorf("关联赠送批次订单失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	for slug := range gitlabRegionsGranted {
		if recErr := recalcRegionAllocated(slug); recErr != nil {
			slog.WarnContext(ctx, "admin_grant_region_allocated_recalc_failed",
				"level", "warn",
				"region", slug,
				"error", recErr.Error(),
			)
		}
		// OPT-20260818-020: 赠送成功后按区域确保租户 GitLab 组存在（与购买路径一致）。
		// 失败仅 warn 不回滚配额；无 PAT 时保持可观测 warn。
		region, rErr := getGitlabRegionBySlug(slug)
		if rErr != nil {
			slog.WarnContext(ctx, "admin_grant_region_lookup_failed",
				"level", "warn",
				"region", slug,
				"error", rErr.Error(),
			)
			continue
		}
		res, lErr := getTenantGitlabResourceByRegion(tenantID, slug)
		if lErr != nil {
			slog.WarnContext(ctx, "admin_grant_region_resource_load_failed",
				"level", "warn",
				"tenant_id", formatID(tenantID),
				"region", slug,
				"error", lErr.Error(),
			)
			continue
		}
		limitBytes := diskLimitBytesFromGB(res.DiskGB)
		if eErr := ensureTenantGitlabGroupForRegion(ctx, tenantID, limitBytes, region); eErr != nil {
			slog.WarnContext(ctx, "admin_grant_region_group_ensure_failed",
				"level", "warn",
				"tenant_id", formatID(tenantID),
				"region", slug,
				"error", eErr.Error(),
			)
		}
	}
	slog.InfoContext(ctx, "admin_grant_resources_ok",
		"level", "info",
		"tenant_id", formatID(tenantID),
		"transaction_id", txnID,
		"grant_count", len(grants),
		"membership_tier", membershipTier,
	)
	if membershipTier != "" {
		slog.InfoContext(ctx, "membership_admin_tier_set",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"from_tier", fromTier,
			"to_tier", membershipTier,
			"admin_tier_locked", true,
		)
	}

	// 记录幂等键（事务已提交，insert 失败不影响结果）
	// OPT-20260823-058：附带操作者 X-User-Id 与目标租户，管理端可追溯「哪次操作产生哪张赠送订单」。
	if idempotencyKey != "" {
		ikID := generateSnowflakeID()
		_, _ = db.Exec(
			`INSERT IGNORE INTO billing_idempotency_key (id, `+"`"+`key`+"`"+`, created_at, transaction_id, operator_user_id, tenant_id, op_type) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			ikID, idempotencyKey, now, txnID, nullStr(userID), tenantID, "admin_grant",
		)
	}

	out := map[string]interface{}{
		"status":                "ok",
		"grants":                grantResults,
		"task_post_quota_after": quotaAfter,
		"tenant_id":             formatID(tenantID),
		"transaction_id":        txnID,
	}
	if orderID != 0 {
		out["order_id"] = formatID(orderID)
		out["order_number"] = orderNumber
	}
	if membershipTier != "" {
		if mem, err := getMembership(tenantID); err == nil {
			out["membership"] = membershipJSON(mem)
		}
	}
	return out, nil
}
