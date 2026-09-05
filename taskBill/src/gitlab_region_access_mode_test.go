package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBillingEventTopicGitlabRegionAccessModeChanged(t *testing.T) {
	if billingEventTopics["GitlabRegionAccessModeChanged"] != "gitlab-region-access-mode-changed" {
		t.Fatalf("topic=%q", billingEventTopics["GitlabRegionAccessModeChanged"])
	}
}

func TestHandleGitlabRegionsListHidesDevelopmentForNonTester(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	now := utcNow()
	slug := "dev-mode-only-region"
	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider,
			access_mode, created_at, updated_at
		) VALUES (?, ?, ?, 'desc', 'https://api.daydaymoney.com', 'https://example.com', 1, 1, 50, 500, 0, 0, 'tok', 'tencent', 'development', ?, ?)
		ON DUPLICATE KEY UPDATE access_mode = 'development', is_active = 1`,
		generateSnowflakeID(), "开发区", slug, now, now,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-regions/", nil)
	rec := httptest.NewRecorder()
	handleGitlabRegionsList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if regionListHasSlug(t, rec.Body.Bytes(), slug) {
		t.Fatal("non-tester must not see development region")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-regions/", nil)
	req2.Header.Set("X-User-Is-Tester", "1")
	rec2 := httptest.NewRecorder()
	handleGitlabRegionsList(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("tester code=%d body=%s", rec2.Code, rec2.Body.String())
	}
	if !regionListHasSlug(t, rec2.Body.Bytes(), slug) {
		t.Fatal("tester must see development region")
	}
}

func TestCreateOrderDevRegionForbiddenWithoutTester(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000099
	seedVIP1Membership(t, tenantID)
	if _, err := db.Exec(`UPDATE billing_gitlab_region SET access_mode = 'development' WHERE slug = 'tencent-sh-1'`); err != nil {
		t.Fatalf("mark development: %v", err)
	}
	_, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	}, "请开通", "user-dev")
	if !errors.Is(err, errGitlabRegionDevModeForbidden) {
		t.Fatalf("want dev-mode forbidden, got %v", err)
	}
	_, _, err = createOrderWithNote(contextWithTester(context.Background(), true), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	}, "请开通", "user-dev")
	if err != nil {
		t.Fatalf("tester should create order: %v", err)
	}
}

func TestUpdateRegionAccessMode(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]string{"access_mode": "development"})
	req := httptest.NewRequest(http.MethodPut, "/api/system-admin/gitlab-regions/tencent-sh-1/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleSystemAdminGitlabRegions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	region, err := getGitlabRegionBySlug("tencent-sh-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if region.AccessMode != gitlabRegionAccessDevelopment {
		t.Fatalf("access_mode=%q", region.AccessMode)
	}
}

func TestListTenantSummariesHidesDevelopment(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000100
	now := utcNow()
	if _, err := db.Exec(`UPDATE billing_gitlab_region SET access_mode = 'development' WHERE slug = 'tencent-sh-1'`); err != nil {
		t.Fatalf("mark development: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, 'tencent-sh-1', 1, 0, 1, '2026-12-31', 0, 0, 'active', ?, ?)`,
		tenantID, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}
	hidden, err := listTenantGitlabResourceSummaries(tenantID, false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hidden) != 0 {
		t.Fatalf("non-tester summaries=%v", hidden)
	}
	shown, err := listTenantGitlabResourceSummaries(tenantID, true)
	if err != nil {
		t.Fatalf("list tester: %v", err)
	}
	if len(shown) != 1 {
		t.Fatalf("tester summaries len=%d", len(shown))
	}
}

func regionListHasSlug(t *testing.T, body []byte, slug string) bool {
	t.Helper()
	var out struct {
		Regions []struct {
			Slug string `json:"slug"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	for _, r := range out.Regions {
		if r.Slug == slug {
			return true
		}
	}
	return false
}
