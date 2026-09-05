package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	gitlabDiskMonthsMin = int64(1)
	gitlabDiskMonthsMax = int64(6)
)

type TenantGitlabResource struct {
	TenantID           int64
	Region             string
	DiskGB             int64
	TrafficPrepaidGB   int64
	DiskMonths         int64
	DiskExpiresAt      string
	DiskUsedBytes      int64
	TrafficUsedGB      float64
	ProvisioningStatus string
	CreatedAt          string
	UpdatedAt          string
}

const bytesPerGiB = 1024.0 * 1024.0 * 1024.0

func getTenantGitlabResourceByRegion(tenantID int64, regionSlug string) (*TenantGitlabResource, error) {
	regionSlug = strings.TrimSpace(regionSlug)
	if regionSlug == "" {
		return nil, fmt.Errorf("region required")
	}
	var r TenantGitlabResource
	err := db.QueryRow(`
		SELECT tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
		       disk_used_bytes, traffic_used_gb, COALESCE(provisioning_status, 'active'), created_at, updated_at
		FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`, tenantID, regionSlug).Scan(
		&r.TenantID, &r.Region, &r.DiskGB, &r.TrafficPrepaidGB, &r.DiskMonths, &r.DiskExpiresAt,
		&r.DiskUsedBytes, &r.TrafficUsedGB, &r.ProvisioningStatus, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return &TenantGitlabResource{TenantID: tenantID, Region: regionSlug, ProvisioningStatus: "not_purchased"}, nil
	}
	if err != nil {
		return nil, err
	}
	// OPT-20260818-030: 零配额行（disk_gb=0 且 traffic=0）视为未购买，
	// 忽略历史遗留 provisioning_status='active'，避免误触发区域 GitLab 组开通。
	if r.DiskGB <= 0 && r.TrafficPrepaidGB <= 0 {
		r.ProvisioningStatus = "not_purchased"
	}
	return &r, nil
}

// getTenantGitlabResource returns resources for the default region (backward compat).
func getTenantGitlabResource(tenantID int64) (*TenantGitlabResource, error) {
	return getTenantGitlabResourceByRegion(tenantID, defaultGitlabRegion)
}

// sumGitlabTrafficUsedGB sums billable outbound traffic (excludes prepaid purchase rows).
func sumGitlabTrafficUsedGB(accountID int64) (float64, error) {
	unitID, err := ensureBillingUnit("gitlab_traffic", "GitLab 流量费", 1)
	if err != nil {
		return 0, err
	}
	var total sql.NullFloat64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM billing_usage
		WHERE account_id = ? AND billing_unit_id = ? AND description LIKE 'GitLab 流量费%'`,
		accountID, unitID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid || total.Float64 < 0 {
		return 0, nil
	}
	return total.Float64, nil
}

func diskUsedGBFromBytes(bytes int64) float64 {
	if bytes <= 0 {
		return 0
	}
	// 保留 6 位小数，避免数十 MB 级占用被四舍五入成 0
	return float64(int64(float64(bytes)/bytesPerGiB*1e6+0.5)) / 1e6
}

// gitlabResourceView returns the latest purchased region for dashboard/compat callers.
func gitlabResourceView(tenantID int64) (map[string]interface{}, error) {
	var slug string
	err := db.QueryRow(`
		SELECT region FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND (disk_gb > 0 OR traffic_prepaid_gb > 0)
		ORDER BY updated_at DESC LIMIT 1`, tenantID).Scan(&slug)
	if err == sql.ErrNoRows || strings.TrimSpace(slug) == "" {
		return map[string]interface{}{
			"tenant_id":           formatID(tenantID),
			"provisioning_status": "not_purchased",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return regionResourceView(tenantID, slug)
}

// listTenantGitlabResourceSummaries returns purchased/granted GitLab regions for a tenant
// (Navbar「代码仓库」下拉). Lightweight: no disk refresh / group ensure side effects.
// Enrich via getGitlabRegionBySlug (no JOIN) to avoid collation mismatch across tables.
func listTenantGitlabResourceSummaries(tenantID int64, isTester bool) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT region,
		       COALESCE(NULLIF(provisioning_status, ''), 'active'),
		       disk_gb,
		       traffic_prepaid_gb
		FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND (disk_gb > 0 OR traffic_prepaid_gb > 0)
		ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		var region, status string
		var diskGB, trafficGB int64
		if err := rows.Scan(&region, &status, &diskGB, &trafficGB); err != nil {
			return nil, err
		}
		regionName := region
		webURL := ""
		if meta, mErr := getGitlabRegionBySlug(region); mErr == nil && meta != nil {
			if !CanUseGitlabRegion(meta.AccessMode, isTester) {
				continue
			}
			if strings.TrimSpace(meta.Name) != "" {
				regionName = meta.Name
			}
			webURL = strings.TrimSpace(meta.GitlabWebURL)
		}
		out = append(out, map[string]interface{}{
			"region":              region,
			"region_name":         regionName,
			"gitlab_web_url":      webURL,
			"provisioning_status": status,
			"disk_gb":             diskGB,
			"traffic_prepaid_gb":  trafficGB,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// reportGitlabDiskUsage upserts measured disk usage for a tenant+region (internal metering).
func reportGitlabDiskUsage(tenantID, diskUsedBytes int64, regionSlug string) error {
	if diskUsedBytes < 0 {
		return fmt.Errorf("disk_used_bytes 须 >= 0")
	}
	if regionSlug == "" {
		regionSlug = defaultGitlabRegion
	}
	now := utcNow()
	// OPT-20260818-030: 用量刷新不得把从未购买租户的计量行写成 provisioning_status=active，
	// 否则 regionResourceView 会误触发 ensureTenantGitlabGroupForRegion。新建行显式置 not_purchased；
	// ON DUPLICATE KEY UPDATE 只刷新用量，不触碰已购买行的配额/状态。
	_, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 0, 0, 0, '', ?, 0, 'not_purchased', ?, ?)
		ON DUPLICATE KEY UPDATE
			disk_used_bytes = VALUES(disk_used_bytes),
			updated_at = VALUES(updated_at)`,
		tenantID, regionSlug, diskUsedBytes, now, now,
	)
	return err
}

// addGitlabTrafficUsedGB increments the denormalized traffic usage counter
// for the default region (charge path has no region yet).
func addGitlabTrafficUsedGB(tenantID int64, gb float64) error {
	return addGitlabTrafficUsedGBForRegion(tenantID, defaultGitlabRegion, gb)
}

func addGitlabTrafficUsedGBForRegion(tenantID int64, regionSlug string, gb float64) error {
	if gb <= 0 {
		return nil
	}
	regionSlug = strings.TrimSpace(regionSlug)
	if regionSlug == "" {
		regionSlug = defaultGitlabRegion
	}
	now := utcNow()
	_, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 0, 0, 0, '', 0, ?, 'not_purchased', ?, ?)
		ON DUPLICATE KEY UPDATE
			traffic_used_gb = traffic_used_gb + VALUES(traffic_used_gb),
			updated_at = VALUES(updated_at)`,
		tenantID, regionSlug, gb, now, now,
	)
	return err
}

func parseNonNegInt64(v interface{}, field string) (int64, error) {
	switch t := v.(type) {
	case nil:
		return 0, fmt.Errorf("缺少 %s", field)
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s 须为整数", field)
		}
		if n < 0 {
			return 0, fmt.Errorf("%s 须 >= 0", field)
		}
		return n, nil
	case float64:
		n := int64(t)
		if float64(n) != t {
			return 0, fmt.Errorf("%s 须为整数", field)
		}
		if n < 0 {
			return 0, fmt.Errorf("%s 须 >= 0", field)
		}
		return n, nil
	case int64:
		if t < 0 {
			return 0, fmt.Errorf("%s 须 >= 0", field)
		}
		return t, nil
	case int:
		if t < 0 {
			return 0, fmt.Errorf("%s 须 >= 0", field)
		}
		return int64(t), nil
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s 须为整数", field)
		}
		if n < 0 {
			return 0, fmt.Errorf("%s 须 >= 0", field)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%s 须为整数", field)
	}
}

// parseDiskMonths: disk_gb>0 时须 1..6（缺省 1）；disk_gb==0 时忽略，返回 0。
func parseDiskMonths(body map[string]interface{}, diskGB int64) (int64, error) {
	if diskGB <= 0 {
		return 0, nil
	}
	raw, ok := body["disk_months"]
	if !ok || raw == nil {
		return 1, nil
	}
	months, err := parseNonNegInt64(raw, "disk_months")
	if err != nil {
		return 0, err
	}
	if months < gitlabDiskMonthsMin || months > gitlabDiskMonthsMax {
		return 0, fmt.Errorf("disk_months 须在 %d～%d 之间", gitlabDiskMonthsMin, gitlabDiskMonthsMax)
	}
	return months, nil
}

// gitlabProvisionExpiresAt 管理员开通成功后的到期日：
// 已有到期时间的行沿用（历史旧逻辑/续费已在支付时计时）；
// 为空则按购买月数自开通时刻起算 now+months（OPT-20260903-012）。
func gitlabProvisionExpiresAt(months int64, existing string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	return diskExpiresAtFromMonths(months, "")
}

// diskExpiresAtFromMonths 计算磁盘到期时间。
// 新购（disk_expires_at 为空）：now + months
// 续费（disk_expires_at 非空）：max(now, current_expires_at) + months
// 续费场景下不会缩短已购买的时长。
func diskExpiresAtFromMonths(months int64, currentExpiresAt ...string) string {
	if months <= 0 {
		return ""
	}
	base := time.Now().UTC()
	// 续费：以当前到期时间和 now 中的较大者为基准
	if len(currentExpiresAt) > 0 && currentExpiresAt[0] != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05.000000", currentExpiresAt[0]); err == nil {
			parsedUTC := parsed.UTC()
			if parsedUTC.After(base) {
				base = parsedUTC
			}
		}
	}
	return base.AddDate(0, int(months), 0).Format("2006-01-02 15:04:05.000000")
}
