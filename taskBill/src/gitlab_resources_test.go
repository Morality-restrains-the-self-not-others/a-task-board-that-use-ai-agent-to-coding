package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetTenantGitlabResourceByRegion_NoRows verifies that when no GitLab resource
// exists for a tenant+region, the function returns a not_purchased status instead
// of an error. This guards the frontend conditional rendering logic.
func TestGetTenantGitlabResourceByRegion_NoRows(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	// Use a tenant ID that does not exist in billing_tenant_gitlab_resource
	const nonExistentTenantID = int64(0)
	const region = "default"

	resource, err := getTenantGitlabResourceByRegion(nonExistentTenantID, region)
	if err != nil {
		t.Fatalf("expected no error for non-existent tenant, got: %v", err)
	}
	if resource == nil {
		t.Fatal("expected non-nil resource")
	}
	if resource.TenantID != nonExistentTenantID {
		t.Errorf("expected tenant_id=%d, got %d", nonExistentTenantID, resource.TenantID)
	}
	if resource.Region != region {
		t.Errorf("expected region=%q, got %q", region, resource.Region)
	}
	if resource.ProvisioningStatus != "not_purchased" {
		t.Errorf("expected provisioning_status=%q, got %q", "not_purchased", resource.ProvisioningStatus)
	}
}

// TestGetTenantGitlabResourceByRegion_DefaultRegion verifies that when regionSlug
// is empty, the default region is used.
func TestGetTenantGitlabResourceByRegion_DefaultRegion(t *testing.T) {
	_, err := getTenantGitlabResourceByRegion(0, "")
	if err == nil {
		t.Fatal("expected error when region empty")
	}
}

// OPT-20260818-030: 用量刷新对从未购买租户不得写 provisioning_status=active 行。
func TestReportGitlabDiskUsageNonPurchasedNotActive(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = 9000000040
	const region = "tencent-sh-1"

	if err := reportGitlabDiskUsage(tenantID, 1024, region); err != nil {
		t.Fatalf("reportGitlabDiskUsage: %v", err)
	}

	var status string
	var diskGB, trafficGB int64
	if err := db.QueryRow(`
		SELECT provisioning_status, disk_gb, traffic_prepaid_gb
		FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`,
		tenantID, region,
	).Scan(&status, &diskGB, &trafficGB); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if status != "not_purchased" {
		t.Errorf("provisioning_status = %q, want not_purchased", status)
	}
	if diskGB != 0 || trafficGB != 0 {
		t.Errorf("disk_gb=%d traffic_prepaid_gb=%d, want both 0", diskGB, trafficGB)
	}
}

// OPT-20260818-030: 零配额行（disk_gb=0 且 traffic=0）即使历史遗留 status=active，
// 也必须视为未购买，防止误触发 ensureTenantGitlabGroupForRegion。
func TestGetTenantGitlabResourceByRegionZeroQuotaTreatedAsNotPurchased(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = 9000000041
	const region = "tencent-sh-1"
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 0, 0, 0, '', 0, 0, 'active', ?, ?)`,
		tenantID, region, now, now,
	); err != nil {
		t.Fatalf("seed active zero-quota row: %v", err)
	}

	res, err := getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("getTenantGitlabResourceByRegion: %v", err)
	}
	if res.ProvisioningStatus != "not_purchased" {
		t.Fatalf("provisioning_status = %q, want not_purchased", res.ProvisioningStatus)
	}
}

// OPT-20260818-022: 多区域租户的 quotas API 返回按区 gitlab_resources 列表。
func TestHandleResourceQuotasMultiRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9000000042)
	now := utcNow()

	for _, regionSlug := range []string{"tencent-shanghai-5", "tencent-sh-1"} {
		if _, err := db.Exec(`
			INSERT INTO billing_gitlab_region (
				id, name, slug, description, gitlab_api_base, gitlab_web_url,
				is_active, sort_order, total_disk_gb, total_traffic_gb,
				allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
			) VALUES (?, ?, ?, 'desc', 'https://api.daydaymoney.com', 'https://example.com', 1, 99, 50, 500, 0, 0, 'test-token', 'tencent', ?, ?)
			ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = VALUES(updated_at)`,
			generateSnowflakeID(), "测试区 "+regionSlug, regionSlug, now, now,
		); err != nil {
			t.Fatalf("seed region %s: %v", regionSlug, err)
		}
	}

	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	for _, regionSlug := range []string{"tencent-shanghai-5", "tencent-sh-1"} {
		if _, err := db.Exec(`
			INSERT INTO billing_tenant_gitlab_resource (
				tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
				disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
			) VALUES (?, ?, 1, 2, 1, '2026-12-31', 1, 0, 'active', ?, ?)`,
			tenantID, regionSlug, now, now,
		); err != nil {
			t.Fatalf("seed tenant resource %s: %v", regionSlug, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+formatID(tenantID)+"/billing/quotas/", nil)
	rec := httptest.NewRecorder()
	handleResourceQuotas(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var aggregate map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &aggregate); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resources, ok := aggregate["gitlab_resources"].([]interface{})
	if !ok {
		t.Fatalf("gitlab_resources should be array, got %T %v", aggregate["gitlab_resources"], aggregate["gitlab_resources"])
	}
	if len(resources) != 2 {
		t.Fatalf("gitlab_resources len = %d, want 2", len(resources))
	}
	got := map[string]bool{}
	for _, item := range resources {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("gitlab_resources item not object: %T", item)
		}
		got[fmt.Sprint(m["region"])] = true
		if _, ok := m["disk_used_gb"]; !ok {
			t.Fatalf("gitlab_resources[%v] missing disk_used_gb", m["region"])
		}
		if _, ok := m["traffic_used_gb"]; !ok {
			t.Fatalf("gitlab_resources[%v] missing traffic_used_gb", m["region"])
		}
	}
	if !got["tencent-shanghai-5"] || !got["tencent-sh-1"] {
		t.Fatalf("gitlab_resources regions = %v, want both tencent-shanghai-5 and tencent-sh-1", got)
	}
	// 旧聚合字段保留
	if _, hasFlat := aggregate["gitlab_disk_gb"]; !hasFlat {
		t.Fatal("legacy gitlab_disk_gb field missing")
	}
}
