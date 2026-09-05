package main

import (
	"fmt"
	"log/slog"
	"strings"
)

func gbToMicro(gb float64) int64 {
	if gb <= 0 {
		return 0
	}
	return int64(gb*1e6 + 0.5)
}

func gbIsWholeAtLeastOne(gb float64) bool {
	if gb < 1 {
		return false
	}
	return gbToMicro(gb) == gbToMicro(float64(int64(gb+0.5)))
}

// trafficUsedIsStaleDiskFloor reports watermarks copied from repository size
// by persistTrafficFloorFromDisk / reportGitlabTrafficUsageFloor(disk bytes).
// Integer GB ≥ 1 may be real chargeGitlabTraffic increments that happen to
// equal disk occupancy; those are kept.
func trafficUsedIsStaleDiskFloor(trafficGB float64, diskBytes int64) bool {
	diskGB := diskUsedGBFromBytes(diskBytes)
	if diskGB <= 0 || trafficGB <= 0 {
		return false
	}
	if gbToMicro(trafficGB) != gbToMicro(diskGB) {
		return false
	}
	if gbIsWholeAtLeastOne(trafficGB) {
		return false
	}
	return true
}

// gitlabTrafficUsedGB is the settings-page "已用流量".
// Prefer denormalized measured usage, then billed GB. Disk occupancy is NOT
// a traffic watermark: push would otherwise keep raising "used" after quota
// exhaustion, and GitLab CE does not meter git-upload-pack by itself.
func gitlabTrafficUsedGB(res *TenantGitlabResource, billedGB float64) float64 {
	if res == nil {
		if billedGB > 0 {
			return billedGB
		}
		return 0
	}
	used := res.TrafficUsedGB
	if trafficUsedIsStaleDiskFloor(used, res.DiskUsedBytes) {
		if res.TenantID > 0 {
			slog.Info("gitlab_traffic_discard_stale_disk_floor",
				"level", "info",
				"tenant_id", formatID(res.TenantID),
				"region", res.Region,
				"disk_used_bytes", res.DiskUsedBytes,
				"discarded_traffic_used_gb", used,
			)
		}
		used = 0
	}
	if billedGB > used {
		used = billedGB
	}
	if used < 0 {
		return 0
	}
	return used
}

// reportGitlabTrafficUsageFloor records measured outbound traffic as an
// absolute watermark (bytes → GB). Never decreases an existing counter.
// New rows are not_purchased so GET refresh cannot provision a GitLab group.
func reportGitlabTrafficUsageFloor(tenantID, trafficUsedBytes int64, regionSlug string) error {
	if trafficUsedBytes < 0 {
		return fmt.Errorf("traffic_used_bytes 须 >= 0")
	}
	regionSlug = strings.TrimSpace(regionSlug)
	if regionSlug == "" {
		regionSlug = defaultGitlabRegion
	}
	gb := diskUsedGBFromBytes(trafficUsedBytes)
	if gb <= 0 {
		return nil
	}
	now := utcNow()
	_, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 0, 0, 0, '', 0, ?, 'not_purchased', ?, ?)
		ON DUPLICATE KEY UPDATE
			traffic_used_gb = GREATEST(traffic_used_gb, VALUES(traffic_used_gb)),
			updated_at = VALUES(updated_at)`,
		tenantID, regionSlug, gb, now, now,
	)
	return err
}
