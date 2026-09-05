package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

const defaultGitlabRegion = "tencent-shanghai-5"

// GitlabRegion represents an available GitLab deployment region.
type GitlabRegion struct {
	ID                     int64  `json:"id"`
	Name                   string `json:"name"`
	Slug                   string `json:"slug"`
	Description            string `json:"description"`
	GitlabAPIBase          string `json:"gitlab_api_base"`
	GitlabWebURL           string `json:"gitlab_web_url"`
	AdminPrivateToken      string `json:"admin_private_token,omitempty"`
	CloudProvider          string `json:"cloud_provider"`
	IsActive               bool   `json:"is_active"`
	SortOrder              int    `json:"sort_order"`
	TotalDiskGB            int64  `json:"total_disk_gb"`
	TotalTrafficGB         int64  `json:"total_traffic_gb"`
	AllocatedDiskGB        int64  `json:"allocated_disk_gb"`
	AllocatedTrafficGB     int64  `json:"allocated_traffic_gb"`
	BandwidthShared        bool   `json:"bandwidth_shared"`
	TotalBandwidthMbps     int64  `json:"total_bandwidth_mbps"`
	RemainingBandwidthMbps int64  `json:"remaining_bandwidth_mbps"`
	AccessMode             string `json:"access_mode"`
	InfraStatus            string `json:"infra_status"`
}

const gitlabRegionSelectCols = `id, name, slug, description, gitlab_api_base, gitlab_web_url,
		       COALESCE(admin_private_token, '') AS admin_private_token,
		       COALESCE(cloud_provider, '') AS cloud_provider,
		       is_active, sort_order,
		       total_disk_gb, total_traffic_gb, allocated_disk_gb, allocated_traffic_gb,
		       bandwidth_shared, total_bandwidth_mbps, remaining_bandwidth_mbps,
		       COALESCE(NULLIF(access_mode, ''), 'release') AS access_mode,
		       COALESCE(NULLIF(infra_status, ''), 'ready') AS infra_status`

func scanGitlabRegion(sc interface{ Scan(dest ...any) error }) (GitlabRegion, error) {
	var r GitlabRegion
	var active, bwShared int
	err := sc.Scan(&r.ID, &r.Name, &r.Slug, &r.Description,
		&r.GitlabAPIBase, &r.GitlabWebURL, &r.AdminPrivateToken, &r.CloudProvider,
		&active, &r.SortOrder,
		&r.TotalDiskGB, &r.TotalTrafficGB, &r.AllocatedDiskGB, &r.AllocatedTrafficGB,
		&bwShared, &r.TotalBandwidthMbps, &r.RemainingBandwidthMbps, &r.AccessMode, &r.InfraStatus)
	if err != nil {
		return r, err
	}
	r.IsActive = active != 0
	r.BandwidthShared = bwShared != 0
	r.AccessMode = NormalizeGitlabRegionAccessMode(r.AccessMode)
	r.InfraStatus = NormalizeGitlabRegionInfraStatus(r.InfraStatus)
	return r, nil
}

func listGitlabRegions() ([]GitlabRegion, error) {
	rows, err := db.Query(`SELECT ` + gitlabRegionSelectCols + `
		FROM billing_gitlab_region WHERE is_active = 1 ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GitlabRegion
	for rows.Next() {
		r, err := scanGitlabRegion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func listAllGitlabRegions() ([]GitlabRegion, error) {
	rows, err := db.Query(`SELECT ` + gitlabRegionSelectCols + `
		FROM billing_gitlab_region ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GitlabRegion
	for rows.Next() {
		r, err := scanGitlabRegion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func getGitlabRegionBySlug(slug string) (*GitlabRegion, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("region required")
	}
	r, err := scanGitlabRegion(db.QueryRow(`SELECT `+gitlabRegionSelectCols+`
		FROM billing_gitlab_region WHERE slug = ? AND is_active = 1`, slug))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("region not found: %s", slug)
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// getAnyGitlabRegionBySlug returns a region regardless of is_active (for admin use).
func getAnyGitlabRegionBySlug(slug string) (*GitlabRegion, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("slug required")
	}
	r, err := scanGitlabRegion(db.QueryRow(`SELECT `+gitlabRegionSelectCols+`
		FROM billing_gitlab_region WHERE slug = ?`, slug))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("region not found: %s", slug)
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// requireRegionSlug returns the region from query or body; empty is an error (no platform default).
func requireRegionSlug(r *http.Request, bodyRegion string) (string, error) {
	slug := strings.TrimSpace(r.URL.Query().Get("region"))
	if slug == "" {
		slug = strings.TrimSpace(bodyRegion)
	}
	if slug == "" {
		return "", fmt.Errorf("region required")
	}
	return slug, nil
}

// resolveRegionSlug is kept for callers that still pass query; empty means missing (no silent default).
func resolveRegionSlug(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("region"))
}

// regionResourceView returns the gitlab resource view for a specific region.
func regionResourceView(tenantID int64, regionSlug string) (map[string]interface{}, error) {
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		return nil, err
	}
	region, err := getGitlabRegionBySlug(regionSlug)
	if err != nil {
		return nil, err
	}

	if gitlabRegionIsPendingNode(region) {
		slog.Info("gitlab_region_view_skip_remote_pending_node",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"region", region.Slug,
		)
	} else if err := refreshTenantDiskUsage(context.Background(), tenantID, region.Slug, false); err != nil {
		slog.Warn("gitlab_disk_usage_refresh_skipped",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"region", region.Slug,
			"error", err.Error(),
		)
	}
	res, err := getTenantGitlabResourceByRegion(tenantID, region.Slug)
	if err != nil {
		return nil, err
	}
	// Ensure GitLab tenant group exists for this region (skip if never purchased)
	if res.DiskUsedBytes <= 0 && res.ProvisioningStatus != "not_purchased" && !gitlabRegionIsPendingNode(region) {
		go func(tid int64, limitGB int64, rcopy GitlabRegion) {
			bg := context.Background()
			limitBytes := diskLimitBytesFromGB(limitGB)
			if err := ensureTenantGitlabGroupForRegion(bg, tid, limitBytes, &rcopy); err != nil {
				slog.WarnContext(bg, "gitlab_tenant_group_ensure_failed",
					"level", "warn", "tenant_id", formatID(tid),
					"region", rcopy.Slug, "error", err.Error(),
				)
			} else {
				slog.InfoContext(bg, "gitlab_tenant_group_ensured",
					"level", "info", "tenant_id", formatID(tid),
					"region", rcopy.Slug, "limit_bytes", limitBytes,
				)
				if err := refreshTenantDiskUsage(bg, tid, rcopy.Slug, true); err != nil {
					slog.WarnContext(bg, "gitlab_disk_usage_post_ensure_refresh_failed",
						"level", "warn", "tenant_id", formatID(tid),
						"region", rcopy.Slug, "error", err.Error(),
					)
				}
			}
		}(tenantID, res.DiskGB, *region)
	}
	billedTraffic, err := sumGitlabTrafficUsedGB(acc.ID)
	if err != nil {
		return nil, err
	}
	trafficUsed := gitlabTrafficUsedGB(res, billedTraffic)
	downloadOK, downloadCode := gitlabTrafficDownloadAllowed(res.TrafficPrepaidGB, trafficUsed, false, false)
	diskPrice, _ := getUnitPriceCents(ResourceTypeGitlabDisk)
	trafficPrice, _ := getUnitPriceCents(ResourceTypeGitlabTraffic)
	// OPT-20260820-041: 按区「赠送/购买」拆分（赠送=admin_grant 批次，购买=合计-赠送）。
	diskGifted, trafficGifted := int64(0), int64(0)
	if splitMap, splitErr := getGitlabQuotaSplit(tenantID); splitErr == nil {
		if s, ok := splitMap[region.Slug]; ok {
			diskGifted = s.DiskGifted
			trafficGifted = s.TrafficGifted
		}
	}
	diskPurchased := res.DiskGB - diskGifted
	if diskPurchased < 0 {
		diskPurchased = 0
	}
	trafficPurchased := res.TrafficPrepaidGB - trafficGifted
	if trafficPurchased < 0 {
		trafficPurchased = 0
	}
	return map[string]interface{}{
		"tenant_id":                       formatID(tenantID),
		"region":                          region.Slug,
		"region_name":                     region.Name,
		"gitlab_web_url":                  region.GitlabWebURL,
		"disk_gb":                         res.DiskGB,
		"traffic_prepaid_gb":              res.TrafficPrepaidGB,
		"disk_months":                     res.DiskMonths,
		"disk_expires_at":                 res.DiskExpiresAt,
		"disk_used_bytes":                 res.DiskUsedBytes,
		"disk_used_gb":                    diskUsedGBFromBytes(res.DiskUsedBytes),
		"traffic_used_gb":                 trafficUsed,
		"traffic_download_allowed":        downloadOK,
		"traffic_quota_code":              downloadCode,
		"disk_gifted_gb":                  diskGifted,
		"disk_purchased_gb":               diskPurchased,
		"traffic_gifted_gb":               trafficGifted,
		"traffic_purchased_gb":            trafficPurchased,
		"gitlab_disk_unit_price_cents":    diskPrice,
		"gitlab_traffic_unit_price_cents": trafficPrice,
		"updated_at":                      res.UpdatedAt,
		"provisioning_status":             res.ProvisioningStatus,
	}, nil
}

// listGitlabResourceViews returns per-region GitLab resource views for a tenant.
// OPT-20260818-022/018: 租户购买多区域后账单页需按区展示磁盘/流量，
// gitlabResourceView 只看最近一区会导致第二区域在首页不出现。
func listGitlabResourceViews(tenantID int64, isTester bool) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT region FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND (disk_gb > 0 OR traffic_prepaid_gb > 0)
		ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		if meta, mErr := getGitlabRegionBySlug(slug); mErr == nil && meta != nil {
			if !CanUseGitlabRegion(meta.AccessMode, isTester) {
				continue
			}
		}
		v, err := regionResourceView(tenantID, slug)
		if err != nil {
			slog.Warn("gitlab_region_view_skipped",
				"level", "warn",
				"tenant_id", formatID(tenantID),
				"region", slug,
				"error", err.Error(),
			)
			continue
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// handleGitlabRegionsList returns available GitLab regions.
func handleGitlabRegionsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	regions, err := listGitlabRegions()
	if err != nil {
		slog.ErrorContext(r.Context(), "gitlab_regions_list_failed", "level", "error", "error", err.Error())
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	regions = filterUsableGitlabRegions(regions, requestIsTester(r))
	if regions == nil {
		regions = []GitlabRegion{}
	}
	for i := range regions {
		regions[i].AdminPrivateToken = ""
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"regions": regions, "total": len(regions)})
}

// handleAdminProvisionGitlabResource is called by system admins to provision
// a pending GitLab resource purchase (create GitLab group, set quotas, etc.).
func handleAdminProvisionGitlabResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireTenantAdmin(w, r, "") {
		// Allow system admins; check via internal secret
		if !requireInternalSecret(r) {
			writeErrorJSON(w, http.StatusForbidden, "admin or internal secret required", tracelog.TraceIDFromContext(r.Context()))
			return
		}
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	regionSlug := strings.TrimSpace(stringField(body, "region"))
	if regionSlug == "" {
		writeErrorJSON(w, http.StatusBadRequest, "region required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	region, err := getGitlabRegionBySlug(regionSlug)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ctx := r.Context()
	if gitlabRegionIsPendingNode(region) {
		slog.WarnContext(ctx, "gitlab_provision_blocked_pending_node",
			"level", "warn",
			"tenant_id", formatID(tid),
			"region", region.Slug,
			"cloud_provider", region.CloudProvider,
		)
		writeErrorJSON(w, http.StatusConflict, errGitlabRegionInfraPending.Error(), tracelog.TraceIDFromContext(ctx))
		return
	}

	// 1. Ensure GitLab group exists
	res, err := getTenantGitlabResourceByRegion(tid, region.Slug)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	limitBytes := diskLimitBytesFromGB(res.DiskGB)
	if err := ensureTenantGitlabGroupForRegion(ctx, tid, limitBytes, region); err != nil {
		slog.ErrorContext(ctx, "admin_provision_gitlab_group_failed",
			"level", "error", "tenant_id", formatID(tid),
			"region", region.Slug, "error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "failed to create GitLab group: "+err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 2. Mark as provisioned
	// OPT-20260903-012: 到期日自开通时刻起算。支付时首次购买到期日留空，
	// 此处按 disk_months 从 now 计算（已有到期时间的行保留原到期日）。
	now := utcNow()
	expiresAt := gitlabProvisionExpiresAt(res.DiskMonths, res.DiskExpiresAt)
	_, err = db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			provisioning_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)
		ON DUPLICATE KEY UPDATE
			provisioning_status = 'active',
			disk_expires_at = VALUES(disk_expires_at),
			updated_at = VALUES(updated_at)`,
		tid, region.Slug, res.DiskGB, res.TrafficPrepaidGB, res.DiskMonths, expiresAt, now, now,
	)
	if err != nil {
		slog.ErrorContext(ctx, "admin_provision_update_status_failed",
			"level", "error", "tenant_id", formatID(tid), "error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 3. Force refresh disk usage
	go func(tid int64, rslug string) {
		bg := context.Background()
		if err := refreshTenantDiskUsage(bg, tid, rslug, true); err != nil {
			slog.WarnContext(bg, "admin_provision_refresh_failed",
				"level", "warn", "tenant_id", formatID(tid),
				"region", rslug, "error", err.Error(),
			)
		}
	}(tid, region.Slug)

	slog.InfoContext(ctx, "admin_provision_gitlab_complete",
		"level", "info", "tenant_id", formatID(tid),
		"region", region.Slug, "region_name", region.Name,
	)

	view, _ := regionResourceView(tid, region.Slug)
	if view == nil {
		view = map[string]interface{}{
			"tenant_id":           formatID(tid),
			"region":              region.Slug,
			"provisioning_status": "active",
		}
	}
	view["provisioned"] = true
	writeJSON(w, http.StatusOK, view)
}

// checkRegionCapacity verifies that the region has enough remaining disk/traffic capacity.
// Returns nil if sufficient, or an error describing the shortfall.
func checkRegionCapacity(regionSlug string, diskGB, trafficGB int64) error {
	region, err := getGitlabRegionBySlug(regionSlug)
	if err != nil {
		return err
	}
	if region.TotalDiskGB > 0 {
		remaining := region.TotalDiskGB - region.AllocatedDiskGB
		if diskGB > remaining {
			return fmt.Errorf("区域 %s 磁盘容量不足：需要 %d GB，剩余 %d GB（总量 %d GB）",
				region.Name, diskGB, remaining, region.TotalDiskGB)
		}
	}
	if region.TotalTrafficGB > 0 {
		remaining := region.TotalTrafficGB - region.AllocatedTrafficGB
		if trafficGB > remaining {
			return fmt.Errorf("区域 %s 流量容量不足：需要 %d GB，剩余 %d GB（总量 %d GB）",
				region.Name, trafficGB, remaining, region.TotalTrafficGB)
		}
	}
	return nil
}

// recalcRegionAllocated recomputes allocated_disk_gb and allocated_traffic_gb for a region.
func recalcRegionAllocated(regionSlug string) error {
	_, err := db.Exec(`
		UPDATE billing_gitlab_region SET
			allocated_disk_gb    = COALESCE((SELECT SUM(disk_gb) FROM billing_tenant_gitlab_resource WHERE region = ? AND provisioning_status = 'active'), 0),
			allocated_traffic_gb = COALESCE((SELECT SUM(traffic_prepaid_gb) FROM billing_tenant_gitlab_resource WHERE region = ? AND provisioning_status = 'active'), 0)
		WHERE slug = ?`, regionSlug, regionSlug, regionSlug)
	return err
}
