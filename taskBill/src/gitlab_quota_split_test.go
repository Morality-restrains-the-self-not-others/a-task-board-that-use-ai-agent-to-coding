package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260820-041: quotas 接口按区返回 GitLab 磁盘/流量「赠送/购买」拆分。
// 赠送来自 billing_resource_grant(source_kind=gift, region=slug)；购买 = 区域合计 - 赠送。
func TestHandleResourceQuotasGitlabGiftedPurchasedSplit(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9000000050)
	const regionSlug = "tencent-sh-1"
	now := utcNow()

	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, ?, ?, 'desc', 'https://api.daydaymoney.com', 'https://example.com', 1, 99, 50, 500, 0, 0, 'test-token', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = VALUES(updated_at)`,
		generateSnowflakeID(), "腾讯上海一区", regionSlug, now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	// 区域合计：磁盘 100GB / 流量 500GB
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 100, 500, 1, '2026-12-31', 0, 0, 'active', ?, ?)`,
		tenantID, regionSlug, now, now,
	); err != nil {
		t.Fatalf("seed tenant resource: %v", err)
	}

	// 后台赠送批次（admin_grant 写 billing_resource_grant 带 region）：磁盘 30GB + 流量 100GB
	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, payment_method, created_at, paid_at)
		VALUES (?, ?, 'OPT-041-gift-1', 'paid', 0, 'admin_grant', ?, ?)`,
		orderID, tenantID, now, now,
	); err != nil {
		t.Fatalf("seed gift order: %v", err)
	}
	for _, item := range []struct {
		resourceType string
		qty          int64
	}{
		{ResourceTypeGitlabDisk, 30},
		{ResourceTypeGitlabTraffic, 100},
	} {
		if _, err := db.Exec(`
			INSERT INTO billing_resource_order_item (id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, created_at)
			VALUES (?, ?, ?, ?, 0, 0, ?, ?)`,
			generateSnowflakeID(), orderID, item.resourceType, item.qty, regionSlug, now,
		); err != nil {
			t.Fatalf("seed gift order item: %v", err)
		}
	}
	for _, item := range []struct {
		resourceType string
		qty          int64
	}{
		{ResourceTypeGitlabDisk, 30},
		{ResourceTypeGitlabTraffic, 100},
	} {
		if _, err := db.Exec(`
			INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at, source_kind, region, order_id)
			VALUES (?, ?, ?, ?, ?, 'OPT-041 test gift', '2099-12-31 23:59:59', ?, 'gift', ?, ?)`,
			generateSnowflakeID(), tenantID, item.resourceType, item.qty, item.qty, now, regionSlug, orderID,
		); err != nil {
			t.Fatalf("seed grant: %v", err)
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
	if !ok || len(resources) != 1 {
		t.Fatalf("gitlab_resources should have 1 region, got %T %v", aggregate["gitlab_resources"], aggregate["gitlab_resources"])
	}
	m, ok := resources[0].(map[string]interface{})
	if !ok {
		t.Fatalf("gitlab_resources[0] not object: %T", resources[0])
	}
	wantSplit := map[string]int64{
		"disk_gifted_gb":       30,
		"disk_purchased_gb":    70,
		"traffic_gifted_gb":    100,
		"traffic_purchased_gb": 400,
	}
	for field, want := range wantSplit {
		got := int64(m[field].(float64))
		if got != want {
			t.Errorf("%s = %d, want %d", field, got, want)
		}
	}
}
