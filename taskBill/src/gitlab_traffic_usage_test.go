package main

import "testing"

func TestGitlabTrafficUsedGB_DiskDoesNotInflateUsed(t *testing.T) {
	res := &TenantGitlabResource{DiskUsedBytes: 11263830, TrafficUsedGB: 0}
	got := gitlabTrafficUsedGB(res, 0)
	if got != 0 {
		t.Fatalf("gitlabTrafficUsedGB = %v, want 0 (disk is not traffic)", got)
	}
}

func TestGitlabTrafficUsedGB_StaleDiskFloorWatermarkDiscarded(t *testing.T) {
	const diskBytes int64 = 11263830
	stale := diskUsedGBFromBytes(diskBytes)
	if stale <= 0 {
		t.Fatalf("fixture disk GB must be > 0, got %v", stale)
	}
	res := &TenantGitlabResource{DiskUsedBytes: diskBytes, TrafficUsedGB: stale}
	got := gitlabTrafficUsedGB(res, 0)
	if got != 0 {
		t.Fatalf("gitlabTrafficUsedGB = %v, want 0 (stale disk-floor watermark)", got)
	}
	if billed := gitlabTrafficUsedGB(res, 2); billed != 2 {
		t.Fatalf("billed must still win after discard: got %v", billed)
	}
}

func TestGitlabTrafficUsedGB_IntegerChargeNotDiscardedWhenEqualsDiskGiB(t *testing.T) {
	res := &TenantGitlabResource{DiskUsedBytes: 1 << 30, TrafficUsedGB: 1}
	got := gitlabTrafficUsedGB(res, 0)
	if got != 1 {
		t.Fatalf("integer charged GB must be kept: got %v", got)
	}
}

func TestGitlabTrafficUsedGB_BilledWinsWhenLarger(t *testing.T) {
	res := &TenantGitlabResource{DiskUsedBytes: 1024, TrafficUsedGB: 0.01}
	got := gitlabTrafficUsedGB(res, 2)
	if got != 2 {
		t.Fatalf("gitlabTrafficUsedGB = %v, want billed 2", got)
	}
}

func TestGitlabTrafficUsedGB_MeasuredWinsOverDisk(t *testing.T) {
	res := &TenantGitlabResource{DiskUsedBytes: 1 << 30, TrafficUsedGB: 1.5}
	got := gitlabTrafficUsedGB(res, 0)
	if got != 1.5 {
		t.Fatalf("gitlabTrafficUsedGB = %v, want measured 1.5", got)
	}
}

func TestGitlabTrafficUsedGB_NilResourceUsesBilled(t *testing.T) {
	if got := gitlabTrafficUsedGB(nil, 3); got != 3 {
		t.Fatalf("nil res billed = %v, want 3", got)
	}
	if got := gitlabTrafficUsedGB(nil, 0); got != 0 {
		t.Fatalf("nil res zero = %v, want 0", got)
	}
}

func TestReportGitlabTrafficUsageFloorGreatest(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID int64 = 9000000091
	const region = "tencent-sh-1"

	if err := reportGitlabTrafficUsageFloor(tenantID, 1024, region); err != nil {
		t.Fatalf("first floor: %v", err)
	}
	res, err := getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	first := res.TrafficUsedGB
	if first <= 0 {
		t.Fatalf("traffic_used_gb after 1024 bytes = %v, want > 0", first)
	}

	if err := reportGitlabTrafficUsageFloor(tenantID, 512, region); err != nil {
		t.Fatalf("smaller floor: %v", err)
	}
	res, err = getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("get after smaller: %v", err)
	}
	if res.TrafficUsedGB != first {
		t.Fatalf("counter decreased: got %v want %v", res.TrafficUsedGB, first)
	}

	if err := reportGitlabTrafficUsageFloor(tenantID, 10<<20, region); err != nil {
		t.Fatalf("larger floor: %v", err)
	}
	res, err = getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("get after larger: %v", err)
	}
	if res.TrafficUsedGB <= first {
		t.Fatalf("counter did not rise: got %v prev %v", res.TrafficUsedGB, first)
	}
	if res.ProvisioningStatus != "not_purchased" {
		t.Fatalf("floor insert must stay not_purchased, got %q", res.ProvisioningStatus)
	}
}

func TestRegionResourceViewTrafficFloorFromStoredDisk(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID int64 = 9000000092
	const region = "tencent-sh-1"
	now := utcNow()
	oldBase := cfg.TaskProjectServiceBase
	cfg.TaskProjectServiceBase = "http://127.0.0.1:1"
	defer func() { cfg.TaskProjectServiceBase = oldBase }()

	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, '上海一区', ?, 'desc', 'http://127.0.0.1:1', 'https://example.com',
			1, 1, 50, 500, 0, 0, '', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		generateSnowflakeID(), region, now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, 0, 1, '', 11263830, 0, 'active', ?, ?)`,
		tenantID, region, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	view, err := regionResourceView(tenantID, region)
	if err != nil {
		t.Fatalf("regionResourceView: %v", err)
	}
	got, ok := view["traffic_used_gb"].(float64)
	if !ok {
		t.Fatalf("traffic_used_gb type %T value %#v", view["traffic_used_gb"], view["traffic_used_gb"])
	}
	if got != 0 {
		t.Fatalf("traffic_used_gb = %v, want stored 0 (disk must not inflate)", got)
	}
	if allowed, _ := view["traffic_download_allowed"].(bool); allowed {
		t.Fatalf("prepaid 0 must deny download: %v", view)
	}
	if view["traffic_quota_code"] != trafficGateNotPurchased {
		t.Fatalf("code=%v", view["traffic_quota_code"])
	}
}

func TestRegionResourceViewDiscardsStaleDiskFloorTraffic(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID int64 = 9000000093
	const region = "tencent-sh-1"
	const diskBytes int64 = 11263830
	now := utcNow()
	oldBase := cfg.TaskProjectServiceBase
	cfg.TaskProjectServiceBase = "http://127.0.0.1:1"
	defer func() { cfg.TaskProjectServiceBase = oldBase }()

	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, '上海一区', ?, 'desc', 'http://127.0.0.1:1', 'https://example.com',
			1, 1, 50, 500, 0, 0, '', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		generateSnowflakeID(), region, now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}
	stale := diskUsedGBFromBytes(diskBytes)
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, 0, 1, '', ?, ?, 'active', ?, ?)`,
		tenantID, region, diskBytes, stale, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	view, err := regionResourceView(tenantID, region)
	if err != nil {
		t.Fatalf("regionResourceView: %v", err)
	}
	diskGB, _ := view["disk_used_gb"].(float64)
	if diskGB != stale {
		t.Fatalf("disk_used_gb=%v want stale conversion %v", diskGB, stale)
	}
	got, ok := view["traffic_used_gb"].(float64)
	if !ok {
		t.Fatalf("traffic_used_gb type %T value %#v", view["traffic_used_gb"], view["traffic_used_gb"])
	}
	if got != 0 {
		t.Fatalf("traffic_used_gb = %v, want 0 (must not equal disk_used_gb)", got)
	}
}
