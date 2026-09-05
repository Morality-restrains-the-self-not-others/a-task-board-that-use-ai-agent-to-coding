package main

import (
	"context"
	"database/sql"
	"fmt"

	"tracelog"
)

// 会员等级常量
const (
	MembershipTierNormal = "normal"
	MembershipTierVIP1   = "vip1"
)

// VIP1 自动升级门槛：累计消费满 10000 分（100 元）
const VIP1AutoUpgradeThresholdCents int64 = 10000

// Membership 会员记录
type Membership struct {
	ID                         int64
	TenantID                   int64
	Tier                       string
	CumulativeConsumptionCents int64
	UpgradedAt                 sql.NullString
	AdminTierLocked            bool
	CreatedAt                  string
	UpdatedAt                  string
}

// getMembership 查询租户会员等级
func getMembership(tenantID int64) (*Membership, error) {
	var m Membership
	var locked int64
	err := db.QueryRow(`
		SELECT id, tenant_id, tier, cumulative_consumption_cents, upgraded_at, admin_tier_locked, created_at, updated_at
		FROM billing_membership WHERE tenant_id = ?`, tenantID,
	).Scan(&m.ID, &m.TenantID, &m.Tier, &m.CumulativeConsumptionCents, &m.UpgradedAt, &locked, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // 会员记录不存在（不应发生，但安全处理）
	}
	if err != nil {
		return nil, err
	}
	m.AdminTierLocked = locked != 0
	return &m, nil
}

// ensureMembership 确保租户有会员记录，新租户默认 normal
func ensureMembership(tenantID int64) (*Membership, error) {
	m, err := getMembership(tenantID)
	if err != nil {
		return nil, err
	}
	if m != nil {
		return m, nil
	}

	// 不存在则创建默认 normal 会员
	now := utcNow()
	id := generateSnowflakeID()
	_, err = db.Exec(`
		INSERT INTO billing_membership (id, tenant_id, tier, cumulative_consumption_cents, created_at, updated_at)
		VALUES (?, ?, 'normal', 0, ?, ?)`,
		id, tenantID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("创建会员记录失败: %w", err)
	}
	return getMembership(tenantID)
}

func applyMembershipTierTx(tx *sql.Tx, tenantID int64, tier, now string) error {
	if tier != MembershipTierNormal && tier != MembershipTierVIP1 {
		return fmt.Errorf("invalid membership tier: %s", tier)
	}
	var upgradedArg any
	if tier == MembershipTierVIP1 {
		upgradedArg = now
	}
	var existingID int64
	err := tx.QueryRow(`SELECT id FROM billing_membership WHERE tenant_id = ?`, tenantID).Scan(&existingID)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(`
			INSERT INTO billing_membership (
				id, tenant_id, tier, cumulative_consumption_cents, admin_tier_locked, upgraded_at, created_at, updated_at
			) VALUES (?, ?, ?, 0, 1, ?, ?, ?)`,
			generateSnowflakeID(), tenantID, tier, upgradedArg, now, now,
		)
		return err
	}
	if err != nil {
		return err
	}
	if tier == MembershipTierVIP1 {
		_, err = tx.Exec(`
			UPDATE billing_membership SET tier = ?, admin_tier_locked = 1, upgraded_at = ?, updated_at = ?
			WHERE tenant_id = ?`,
			tier, now, now, tenantID,
		)
		return err
	}
	_, err = tx.Exec(`
		UPDATE billing_membership SET tier = ?, admin_tier_locked = 1, updated_at = ?
		WHERE tenant_id = ?`,
		tier, now, tenantID,
	)
	return err
}

// getTenantCumulativeConsumptionCents 查询租户累计核销消费金额（分）
// 复用 orders.go 中的实现，此处提供别名便于在 membership 上下文中使用
func getMembershipCumulativeConsumption(tenantID int64) (int64, error) {
	return getTenantCumulativeConsumptionCents(tenantID)
}

// syncMembershipConsumption 同步会员累计消费（在订单支付后调用）
// 返回更新后的会员等级信息
func syncMembershipConsumption(ctx context.Context, tenantID int64) (*Membership, error) {
	// 确保会员记录存在
	m, err := ensureMembership(tenantID)
	if err != nil {
		return nil, err
	}

	// 查询最新累计消费
	cumulative, err := getTenantCumulativeConsumptionCents(tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询累计消费失败: %w", err)
	}

	now := utcNow()
	// 更新累计消费金额
	_, err = db.Exec(`
		UPDATE billing_membership SET cumulative_consumption_cents = ?, updated_at = ?
		WHERE tenant_id = ?`,
		cumulative, now, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("更新会员累计消费失败: %w", err)
	}

	m.CumulativeConsumptionCents = cumulative

	// 自动升级检查：normal → vip1（管理员锁定后不得覆盖）
	if m.Tier == MembershipTierNormal && !m.AdminTierLocked && cumulative >= VIP1AutoUpgradeThresholdCents {
		_, err = db.Exec(`
			UPDATE billing_membership SET tier = 'vip1', upgraded_at = ?, updated_at = ?
			WHERE tenant_id = ? AND tier = 'normal'`,
			now, now, tenantID,
		)
		if err != nil {
			return nil, fmt.Errorf("自动升级 VIP1 失败: %w", err)
		}
		m.Tier = MembershipTierVIP1
		m.UpgradedAt = sql.NullString{String: now, Valid: true}

		tracelog.LogForwardStage(ctx, "membership_upgraded_to_vip1", map[string]any{
			"tenant_id":                    formatID(tenantID),
			"from_tier":                    MembershipTierNormal,
			"to_tier":                      MembershipTierVIP1,
			"cumulative_consumption_cents": cumulative,
		})
	}

	m.UpdatedAt = now
	return m, nil
}

// canPurchaseResource 检查会员是否可以购买指定资源类型
func canPurchaseResource(tier string, resourceType string) bool {
	switch tier {
	case MembershipTierNormal:
		// 普通会员只能购买任务帖
		return resourceType == ResourceTypeTaskPost
	case MembershipTierVIP1:
		// VIP1 可以购买所有资源类型
		return resourceType == ResourceTypeTaskPost ||
			resourceType == ResourceTypeGitlabDisk ||
			resourceType == ResourceTypeGitlabTraffic
	default:
		return false
	}
}

// checkMembershipPurchasePermission 检查会员购买权限，返回错误信息
func checkMembershipPurchasePermission(tenantID int64, resourceType string) error {
	m, err := getMembership(tenantID)
	if err != nil {
		return fmt.Errorf("查询会员等级失败: %w", err)
	}
	if m == nil {
		// 没有会员记录，默认按 normal 处理
		if !canPurchaseResource(MembershipTierNormal, resourceType) {
			return fmt.Errorf("当前会员等级（普通会员）不支持购买 %s，累计消费满 100 元可升级 VIP1", resourceType)
		}
		return nil
	}

	if !canPurchaseResource(m.Tier, resourceType) {
		switch m.Tier {
		case MembershipTierNormal:
			switch resourceType {
			case ResourceTypeGitlabDisk:
				return fmt.Errorf("普通会员不支持购买 GitLab 磁盘，累计消费满 100 元可自动升级 VIP1。当前累计消费: %.2f 元",
					float64(m.CumulativeConsumptionCents)/100.0)
			case ResourceTypeGitlabTraffic:
				return fmt.Errorf("普通会员不支持购买 GitLab 流量，累计消费满 100 元可自动升级 VIP1。当前累计消费: %.2f 元",
					float64(m.CumulativeConsumptionCents)/100.0)
			}
		}
		return fmt.Errorf("当前会员等级 %s 不支持购买 %s", m.Tier, resourceType)
	}

	return nil
}

// membershipJSON 将 Membership 转为 API 响应
func membershipJSON(m *Membership) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{
			"tier":                         MembershipTierNormal,
			"cumulative_consumption_cents": 0,
			"cumulative_consumption_yuan":  "0.00",
			"upgraded_at":                  nil,
			"admin_tier_locked":            false,
			"next_tier":                    MembershipTierVIP1,
			"next_tier_threshold_yuan":     "100.00",
			"next_tier_threshold_cents":    VIP1AutoUpgradeThresholdCents,
		}
	}

	result := map[string]interface{}{
		"tier":                         m.Tier,
		"cumulative_consumption_cents": m.CumulativeConsumptionCents,
		"cumulative_consumption_yuan":  centsToYuanStr(m.CumulativeConsumptionCents),
		"admin_tier_locked":            m.AdminTierLocked,
	}

	if m.UpgradedAt.Valid {
		result["upgraded_at"] = m.UpgradedAt.String
	} else {
		result["upgraded_at"] = nil
	}

	// 升级信息
	if m.Tier == MembershipTierNormal {
		result["next_tier"] = MembershipTierVIP1
		result["next_tier_threshold_yuan"] = "100.00"
		result["next_tier_threshold_cents"] = VIP1AutoUpgradeThresholdCents
		remaining := VIP1AutoUpgradeThresholdCents - m.CumulativeConsumptionCents
		if remaining < 0 {
			remaining = 0
		}
		result["remaining_to_upgrade_yuan"] = centsToYuanStr(remaining)
		result["remaining_to_upgrade_cents"] = remaining
	} else {
		result["next_tier"] = nil
		result["next_tier_threshold_yuan"] = nil
		result["next_tier_threshold_cents"] = nil
		result["remaining_to_upgrade_yuan"] = "0.00"
		result["remaining_to_upgrade_cents"] = 0
	}

	return result
}
