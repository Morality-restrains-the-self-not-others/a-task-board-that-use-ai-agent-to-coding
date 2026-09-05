package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func remainingCapacity(total, allocated int64) int64 {
	n := total - allocated
	if n < 0 {
		return 0
	}
	return n
}

func clampRemainingBandwidth(total, remaining int64) int64 {
	if remaining < 0 {
		return 0
	}
	if remaining > total {
		return total
	}
	return remaining
}

func capacityUpdateHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := err.Error()
	if strings.Contains(msg, "至少需要") || strings.Contains(msg, "须") || strings.Contains(msg, "缺少") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func boolToTinyInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func parseOptionalBool(body map[string]interface{}, key string) (value int64, present bool) {
	v, ok := body[key]
	if !ok || v == nil {
		return 0, false
	}
	if b, ok2 := v.(bool); ok2 {
		return boolToTinyInt(b), true
	}
	if n, err := parseNonNegInt64(v, key); err == nil {
		if n > 0 {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func applyGitlabRegionCapacity(ctx context.Context, regionSlug string, body map[string]interface{}) (*GitlabRegion, error) {
	setClauses := []string{}
	args := []interface{}{}

	addInt := func(key, column string) error {
		v, ok := body[key]
		if !ok || v == nil {
			return nil
		}
		n, err := parseNonNegInt64(v, key)
		if err != nil {
			return err
		}
		setClauses = append(setClauses, column+" = ?")
		args = append(args, n)
		return nil
	}

	if err := addInt("total_disk_gb", "total_disk_gb"); err != nil {
		return nil, err
	}
	if err := addInt("total_traffic_gb", "total_traffic_gb"); err != nil {
		return nil, err
	}
	if err := addInt("total_bandwidth_mbps", "total_bandwidth_mbps"); err != nil {
		return nil, err
	}
	if err := addInt("remaining_bandwidth_mbps", "remaining_bandwidth_mbps"); err != nil {
		return nil, err
	}
	if n, ok := parseOptionalBool(body, "bandwidth_shared"); ok {
		setClauses = append(setClauses, "bandwidth_shared = ?")
		args = append(args, n)
	}
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("至少需要 total_disk_gb、total_traffic_gb 或带宽字段")
	}

	args = append(args, regionSlug)
	_, err := db.Exec(fmt.Sprintf("UPDATE billing_gitlab_region SET %s, updated_at = NOW() WHERE slug = ?",
		strings.Join(setClauses, ", ")), args...)
	if err != nil {
		slog.ErrorContext(ctx, "region_capacity_update_failed", "level", "error", "error", err.Error(), "region", regionSlug)
		return nil, err
	}

	region, err := getAnyGitlabRegionBySlug(regionSlug)
	if err != nil {
		return nil, err
	}
	clamped := clampRemainingBandwidth(region.TotalBandwidthMbps, region.RemainingBandwidthMbps)
	if clamped != region.RemainingBandwidthMbps {
		if _, err := db.Exec(`UPDATE billing_gitlab_region SET remaining_bandwidth_mbps = ?, updated_at = NOW() WHERE slug = ?`,
			clamped, regionSlug); err != nil {
			return nil, err
		}
		region.RemainingBandwidthMbps = clamped
	}
	slog.InfoContext(ctx, "region_capacity_updated",
		"level", "info",
		"region", regionSlug,
		"bandwidth_shared", region.BandwidthShared,
		"total_bandwidth_mbps", region.TotalBandwidthMbps,
		"remaining_bandwidth_mbps", region.RemainingBandwidthMbps,
	)
	return region, nil
}

func gitlabRegionCapacityView(region *GitlabRegion) map[string]interface{} {
	return map[string]interface{}{
		"region":                   region.Slug,
		"region_name":              region.Name,
		"total_disk_gb":            region.TotalDiskGB,
		"total_traffic_gb":         region.TotalTrafficGB,
		"allocated_disk_gb":        region.AllocatedDiskGB,
		"allocated_traffic_gb":     region.AllocatedTrafficGB,
		"remaining_disk_gb":        remainingCapacity(region.TotalDiskGB, region.AllocatedDiskGB),
		"remaining_traffic_gb":     remainingCapacity(region.TotalTrafficGB, region.AllocatedTrafficGB),
		"bandwidth_shared":         region.BandwidthShared,
		"total_bandwidth_mbps":     region.TotalBandwidthMbps,
		"remaining_bandwidth_mbps": clampRemainingBandwidth(region.TotalBandwidthMbps, region.RemainingBandwidthMbps),
		"infra_status":             NormalizeGitlabRegionInfraStatus(region.InfraStatus),
		"updated":                  true,
	}
}
