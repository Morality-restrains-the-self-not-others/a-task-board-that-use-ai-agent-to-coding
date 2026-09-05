package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestAdminGrantGitlabDiskRequiresRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9100000001)
	_, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Reason: "no-region"},
	}, "admin-1", "")
	if err == nil {
		t.Fatal("expected error when gitlab_disk grant missing region")
	}
	if !strings.Contains(err.Error(), "region required") {
		t.Fatalf("error should mention region required, got %v", err)
	}
}

func TestAdminGrantGitlabTrafficRequiresRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9100000002)
	_, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabTraffic, Quantity: 3, Reason: "no-region"},
	}, "admin-1", "")
	if err == nil {
		t.Fatal("expected error when gitlab_traffic grant missing region")
	}
	if !strings.Contains(err.Error(), "region required") {
		t.Fatalf("got %v", err)
	}
}

func TestAdminGrantGitlabDiskUnknownRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9100000003)
	_, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Region: "no-such-region", Reason: "bad"},
	}, "admin-1", "")
	if err == nil {
		t.Fatal("expected error for unknown region")
	}
	if !strings.Contains(err.Error(), "region not found") {
		t.Fatalf("got %v", err)
	}
}

func TestAdminGrantGitlabDiskWritesSelectedRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9100000004)
	result, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 2, Region: "tencent-sh-1", Reason: "指定区域"},
	}, "admin-1", "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	var disk int64
	var region string
	if err := db.QueryRow(
		`SELECT disk_gb, region FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`,
		tenantID, "tencent-sh-1",
	).Scan(&disk, &region); err != nil {
		t.Fatalf("query resource: %v", err)
	}
	if disk != 2 || region != "tencent-sh-1" {
		t.Fatalf("disk=%d region=%q, want 2 / tencent-sh-1", disk, region)
	}
	var other int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`,
		tenantID, "tencent-shanghai-5",
	).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if other != 0 {
		t.Fatalf("must not write default region, count=%d", other)
	}
	orderID, _ := result["order_id"].(string)
	if orderID == "" {
		t.Fatal("missing order_id")
	}
	var itemRegion string
	if err := db.QueryRow(
		`SELECT region FROM billing_resource_order_item WHERE order_id = ? AND resource_type = ?`,
		mustParseID(t, orderID), ResourceTypeGitlabDisk,
	).Scan(&itemRegion); err != nil {
		t.Fatalf("query order item: %v", err)
	}
	if itemRegion != "tencent-sh-1" {
		t.Fatalf("order item region=%q", itemRegion)
	}
}

func TestAdminGrantGitlabDiskRegionsIndependent(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9100000005)
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Region: "tencent-sh-1"},
	}, "admin-1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Region: "tencent-shanghai-5"},
	}, "admin-1", ""); err != nil {
		t.Fatal(err)
	}
	var sh1, sh5 int64
	if err := db.QueryRow(`SELECT disk_gb FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`, tenantID, "tencent-sh-1").Scan(&sh1); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT disk_gb FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`, tenantID, "tencent-shanghai-5").Scan(&sh5); err != nil {
		t.Fatal(err)
	}
	if sh1 != 1 || sh5 != 1 {
		t.Fatalf("sh1=%d sh5=%d, want 1 and 1", sh1, sh5)
	}
}

func TestAdminGrantTaskPostWithoutRegionStillWorks(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9100000006)
	result, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 7, Reason: "帖"},
	}, "admin-1", "")
	if err != nil {
		t.Fatalf("task_post grant: %v", err)
	}
	after, _ := result["task_post_quota_after"].(int64)
	if after < 7 {
		t.Fatalf("quota_after=%v", result["task_post_quota_after"])
	}
}

// OPT-20260818-020: 赠送 GitLab 磁盘后按区域确保租户 GitLab 组存在（与购买路径一致）。
func TestAdminGrantGitlabDiskEnsuresGroupForRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9100000008)
	const regionSlug = "test-ensure-region"

	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		// 组不存在 → 404，确保逻辑应尝试创建
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, ?, ?, 'desc', ?, 'https://example.com', 1, 99, 50, 500, 0, 0, 'test-token', 'tencent', ?, ?)`,
		generateSnowflakeID(), "测试区域", regionSlug, srv.URL, utcNow(), utcNow(),
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}

	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 2, Region: regionSlug},
	}, "admin-1", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	var disk int64
	if err := db.QueryRow(`SELECT disk_gb FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`, tenantID, regionSlug).Scan(&disk); err != nil {
		t.Fatalf("query resource: %v", err)
	}
	if disk != 2 {
		t.Fatalf("disk=%d, want 2", disk)
	}

	mu.Lock()
	defer mu.Unlock()
	var sawGet bool
	for _, c := range calls {
		if strings.HasPrefix(c, "GET /api/v4/groups/tenant-") {
			sawGet = true
		}
	}
	if !sawGet {
		t.Fatalf("expected ensure group GET, calls=%v", calls)
	}
}

func TestHandleAdminGrantResourcesParsesRegion(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid = int64(9100000007)
	body := `{"resources":[{"resource_type":"gitlab_disk","quantity":1,"region":"tencent-sh-1","reason":"http"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/9100000007/billing/accounts/admin_grant_points/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var disk int64
	if err := db.QueryRow(`SELECT disk_gb FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`, tid, "tencent-sh-1").Scan(&disk); err != nil {
		t.Fatal(err)
	}
	if disk != 1 {
		t.Fatalf("disk=%d", disk)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	grants, _ := parsed["grants"].([]interface{})
	if len(grants) != 1 {
		t.Fatalf("grants=%v", parsed["grants"])
	}
}

func mustParseID(t *testing.T, s string) int64 {
	t.Helper()
	id, err := parseIDField(s)
	if err != nil {
		t.Fatalf("parse id %q: %v", s, err)
	}
	return id
}
