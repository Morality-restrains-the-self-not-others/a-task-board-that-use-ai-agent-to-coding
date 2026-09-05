package main

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// DefaultTaskPostUnitPriceCents 任务帖默认定价：0.55 元/帖/12个月（续存只扣配额、不另扣费）。
const DefaultTaskPostUnitPriceCents int64 = 55

// DefaultGitlabDiskUnitPriceCents GitLab 磁盘默认定价：4.00 元/GB/月。
const DefaultGitlabDiskUnitPriceCents int64 = 400

// GitlabDiskMinPurchaseGB 用户下单 GitLab 磁盘起购 GB（管理端赠送不受此限）。
const GitlabDiskMinPurchaseGB int64 = 10

func validateGitlabDiskPurchaseQuantity(qty int64) error {
	if qty < GitlabDiskMinPurchaseGB {
		return fmt.Errorf("GitLab 磁盘起购 %d GB", GitlabDiskMinPurchaseGB)
	}
	return nil
}

// ResourcePricing 资源定价（来自 billing_unit 表）
type ResourcePricing struct {
	TaskPostUnitPriceCents           int64  `json:"task_post_unit_price_cents"`           // 任务帖单价（分）
	TaskPostMinConsumptionCents      int64  `json:"task_post_min_consumption_cents"`      // 任务帖最低累计消费门槛（分），0=无门槛
	GitlabDiskUnitPriceCents         int64  `json:"gitlab_disk_unit_price_cents"`         // GitLab 磁盘单价（分/GB/月）
	GitlabDiskMinConsumptionCents    int64  `json:"gitlab_disk_min_consumption_cents"`    // GitLab 磁盘最低累计消费门槛（分），0=无门槛
	GitlabTrafficUnitPriceCents      int64  `json:"gitlab_traffic_unit_price_cents"`      // GitLab 流量单价（分/GB）
	GitlabTrafficMinConsumptionCents int64  `json:"gitlab_traffic_min_consumption_cents"` // GitLab 流量最低累计消费门槛（分），0=无门槛
	GitlabTrafficRequiresUnitType    string `json:"gitlab_traffic_requires_unit_type"`    // GitLab 流量前置依赖 unit_type
	UpdatedAt                        string `json:"updated_at"`
}

// getCurrentResourcePricing 从 billing_unit 表读取当前资源定价
func getCurrentResourcePricing() (*ResourcePricing, error) {
	rp := &ResourcePricing{}
	var updatedAt sql.NullString

	// 任务（server_start）
	var taskMinConsumption sql.NullInt64
	err := db.QueryRow(`SELECT price, COALESCE(min_consumption_cents, 0), updated_at FROM billing_unit WHERE unit_type = 'server_start'`).
		Scan(&rp.TaskPostUnitPriceCents, &taskMinConsumption, &updatedAt)
	if err == sql.ErrNoRows {
		slog.Warn("billing_unit missing, using default task post price",
			"unit_type", "server_start",
			"price_cents", DefaultTaskPostUnitPriceCents)
		rp.TaskPostUnitPriceCents = DefaultTaskPostUnitPriceCents
	} else if err != nil {
		return nil, fmt.Errorf("读取任务定价失败: %w", err)
	}
	if taskMinConsumption.Valid {
		rp.TaskPostMinConsumptionCents = taskMinConsumption.Int64
	}
	if updatedAt.Valid && updatedAt.String > rp.UpdatedAt {
		rp.UpdatedAt = updatedAt.String
	}

	// GitLab 磁盘
	var diskMinConsumption sql.NullInt64
	err = db.QueryRow(`SELECT price, COALESCE(min_consumption_cents, 0), updated_at FROM billing_unit WHERE unit_type = 'gitlab_disk'`).
		Scan(&rp.GitlabDiskUnitPriceCents, &diskMinConsumption, &updatedAt)
	if err == sql.ErrNoRows {
		rp.GitlabDiskUnitPriceCents = DefaultGitlabDiskUnitPriceCents
	} else if err != nil {
		return nil, fmt.Errorf("读取 GitLab 磁盘定价失败: %w", err)
	}
	if diskMinConsumption.Valid {
		rp.GitlabDiskMinConsumptionCents = diskMinConsumption.Int64
	}
	if updatedAt.Valid && updatedAt.String > rp.UpdatedAt {
		rp.UpdatedAt = updatedAt.String
	}

	// GitLab 流量费
	var trafficMinConsumption sql.NullInt64
	var requiresUnitType sql.NullString
	err = db.QueryRow(`SELECT price, COALESCE(min_consumption_cents, 0), COALESCE(requires_unit_type, ''), updated_at FROM billing_unit WHERE unit_type = 'gitlab_traffic'`).
		Scan(&rp.GitlabTrafficUnitPriceCents, &trafficMinConsumption, &requiresUnitType, &updatedAt)
	if err == sql.ErrNoRows {
		rp.GitlabTrafficUnitPriceCents = 100 // 默认 1.00 元
	} else if err != nil {
		return nil, fmt.Errorf("读取 GitLab 流量定价失败: %w", err)
	}
	if trafficMinConsumption.Valid {
		rp.GitlabTrafficMinConsumptionCents = trafficMinConsumption.Int64
	}
	if requiresUnitType.Valid {
		rp.GitlabTrafficRequiresUnitType = requiresUnitType.String
	}
	if updatedAt.Valid && updatedAt.String > rp.UpdatedAt {
		rp.UpdatedAt = updatedAt.String
	}

	return rp, nil
}

// updateResourcePricing 更新 billing_unit 表中的资源定价
func updateResourcePricing(rp *ResourcePricing) error {
	now := utcNow()

	// 基本定价字段
	updates := []struct {
		unitType string
		name     string
		unit     string
		price    int64
	}{
		// v15: server_start 语义 = 创建帖费用（0.55元/帖/12个月）；续存只扣配额、不另扣费
		{"server_start", "创建任务帖", "帖/12个月", rp.TaskPostUnitPriceCents},
		{"gitlab_disk", "GitLab 磁盘", "GB/月", rp.GitlabDiskUnitPriceCents},
		{"gitlab_traffic", "GitLab 流量费", "GB", rp.GitlabTrafficUnitPriceCents},
	}

	for _, u := range updates {
		var existingID int64
		err := db.QueryRow(`SELECT id FROM billing_unit WHERE unit_type = ?`, u.unitType).Scan(&existingID)
		if err == sql.ErrNoRows {
			id := generateSnowflakeID()
			_, err = db.Exec(`
				INSERT INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
				id, u.unitType, u.name, u.price, u.unit, now, now)
		} else if err == nil {
			_, err = db.Exec(`
				UPDATE billing_unit SET name = ?, price = ?, unit = ?, updated_at = ?
				WHERE unit_type = ?`,
				u.name, u.price, u.unit, now, u.unitType)
		}
		if err != nil {
			return fmt.Errorf("更新 %s 定价失败: %w", u.unitType, err)
		}
	}

	return nil
}

// updateResourcePricingConstraint 仅更新约束字段（由 handler 在需要时显式调用）
func updateResourcePricingConstraint(unitType, requiresUnitType string) error {
	now := utcNow()
	if unitType == "gitlab_traffic" {
		_, err := db.Exec(`
			UPDATE billing_unit SET requires_unit_type = ?, updated_at = ?
			WHERE unit_type = 'gitlab_traffic'`,
			requiresUnitType, now)
		if err != nil {
			return fmt.Errorf("更新 GitLab 流量前置依赖失败: %w", err)
		}
	}
	return nil
}

// getUnitPriceCents 获取指定资源类型的当前单价（分）
func getUnitPriceCents(resourceType string) (int64, error) {
	var unitType string
	switch resourceType {
	case ResourceTypeTaskPost:
		unitType = "server_start"
	case ResourceTypeGitlabDisk:
		unitType = "gitlab_disk"
	case ResourceTypeGitlabTraffic:
		unitType = "gitlab_traffic"
	default:
		return 0, fmt.Errorf("不支持的资源类型: %s", resourceType)
	}

	var price int64
	err := db.QueryRow(`SELECT price FROM billing_unit WHERE unit_type = ?`, unitType).Scan(&price)
	if err == sql.ErrNoRows {
		// 回退默认值
		switch resourceType {
		case ResourceTypeTaskPost:
			return DefaultTaskPostUnitPriceCents, nil
		case ResourceTypeGitlabDisk:
			return DefaultGitlabDiskUnitPriceCents, nil
		case ResourceTypeGitlabTraffic:
			return 100, nil
		}
	}
	if err != nil {
		return 0, fmt.Errorf("读取 %s 单价失败: %w", resourceType, err)
	}
	return price, nil
}

// resourcePricingToJSON 将 ResourcePricing 转为 API 响应
func resourcePricingToJSON(rp *ResourcePricing) map[string]interface{} {
	// 构建 requires_unit_type 的中文标签（用于前端展示前置依赖）
	requiresLabel := ""
	if rp.GitlabTrafficRequiresUnitType != "" {
		switch rp.GitlabTrafficRequiresUnitType {
		case "gitlab_disk":
			requiresLabel = "GitLab 磁盘"
		case "server_start":
			requiresLabel = "任务"
		default:
			requiresLabel = rp.GitlabTrafficRequiresUnitType
		}
	}

	result := map[string]interface{}{
		"task_post": map[string]interface{}{
			"name":                  "创建任务帖",
			"unit":                  "帖/12个月",
			"price_yuan":            centsToYuanStr(rp.TaskPostUnitPriceCents),
			"price_cents":           rp.TaskPostUnitPriceCents,
			"required_tier":         MembershipTierNormal,
			"required_tier_label":   "普通会员",
			"min_consumption_yuan":  centsToYuanStr(rp.TaskPostMinConsumptionCents),
			"min_consumption_cents": rp.TaskPostMinConsumptionCents,
			// v15: 创建帖 = 12 个月存续期；存续期内执行不再按次扣费
			"description":            "创建任务帖（含 12 个月存续期；续存只扣配额、不另扣费）",
			"expires_months":         12,
			"renewal_consumes_quota": true,
			"renewal_extra_charge":   false,
		},
		"gitlab_disk": map[string]interface{}{
			"name":                  "GitLab 磁盘",
			"unit":                  "GB/月",
			"price_yuan":            centsToYuanStr(rp.GitlabDiskUnitPriceCents),
			"price_cents":           rp.GitlabDiskUnitPriceCents,
			"required_tier":         MembershipTierVIP1,
			"required_tier_label":   "VIP1",
			"min_consumption_yuan":  centsToYuanStr(rp.GitlabDiskMinConsumptionCents),
			"min_consumption_cents": rp.GitlabDiskMinConsumptionCents,
			"description":           fmt.Sprintf("GitLab 仓库磁盘空间，起购 %d GB", GitlabDiskMinPurchaseGB),
			"min_quantity":          GitlabDiskMinPurchaseGB,
		},
		"gitlab_traffic": map[string]interface{}{
			"name":                  "GitLab 流量费",
			"unit":                  "GB",
			"price_yuan":            centsToYuanStr(rp.GitlabTrafficUnitPriceCents),
			"price_cents":           rp.GitlabTrafficUnitPriceCents,
			"required_tier":         MembershipTierVIP1,
			"required_tier_label":   "VIP1",
			"min_consumption_yuan":  centsToYuanStr(rp.GitlabTrafficMinConsumptionCents),
			"min_consumption_cents": rp.GitlabTrafficMinConsumptionCents,
			"requires_unit_type":    rp.GitlabTrafficRequiresUnitType,
			"requires_label":        requiresLabel,
			"description":           "GitLab 外网流量（同区域内网不计费）",
		},
		"tiers": map[string]interface{}{
			MembershipTierNormal: map[string]interface{}{
				"name":                "普通会员",
				"description":         "注册用户默认等级，可购买任务帖",
				"can_purchase":        []string{"task_post"},
				"can_purchase_labels": []string{"任务"},
			},
			MembershipTierVIP1: map[string]interface{}{
				"name":                    "VIP1",
				"description":             "可购买全部资源：任务帖、GitLab 磁盘、GitLab 流量费",
				"upgrade_threshold_yuan":  "100.00",
				"upgrade_threshold_cents": VIP1AutoUpgradeThresholdCents,
				"can_purchase":            []string{"task_post", "gitlab_disk", "gitlab_traffic"},
				"can_purchase_labels":     []string{"任务", "GitLab 磁盘", "GitLab 流量费"},
			},
		},
		"updated_at": rp.UpdatedAt,
	}
	return result
}

// yuanToCents 元转分（四舍五入）
func yuanToCents(yuan float64) int64 {
	return int64(yuan*100 + 0.5)
}
