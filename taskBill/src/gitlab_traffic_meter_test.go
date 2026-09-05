package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedGitlabMeterFixture(t *testing.T, tenantID int64, region, webURL string, prepaidGB int64) {
	t.Helper()
	now := utcNow()
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
		) VALUES (?, '上海一区', ?, 'desc', 'http://127.0.0.1:1', ?,
			1, 1, 50, 500, 0, 0, '', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE gitlab_web_url = VALUES(gitlab_web_url), name = VALUES(name)`,
		generateSnowflakeID(), region, webURL, now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, ?, 1, '', 0, 0, 'active', ?, ?)`,
		tenantID, region, prepaidGB, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}
}

func TestGitlabHostFromURL(t *testing.T) {
	if got := gitlabHostFromURL("https://gitlab-tencent-sh-1.daydaymoney.com/ljy/somanyad.git"); got != "gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("host=%q", got)
	}
	if got := gitlabHostFromURL("github.com/foo/bar"); got != "github.com" {
		t.Fatalf("host=%q", got)
	}
	if got := gitlabHostFromURL("git@gitlab-tencent-sh-1.daydaymoney.com:ljy/somanyad.git"); got != "gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("ssh host=%q", got)
	}
	if got := gitlabHostFromURL("ssh://git@gitlab-tencent-sh-1.daydaymoney.com/ljy/somanyad.git"); got != "gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("ssh:// host=%q", got)
	}
}

func TestMeterGitlabOutboundTraffic_CloneBytesRaiseUsed(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000101
	const region = "tencent-sh-1"
	web := "https://gitlab-tencent-sh-1.daydaymoney.com"
	seedGitlabMeterFixture(t, tenantID, region, web, 1)

	const cloneBytes int64 = 10 << 20
	wantGB := diskUsedGBFromBytes(cloneBytes)
	if wantGB <= 0 {
		t.Fatalf("fixture bytes must convert to >0 GB")
	}
	got, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		TenantID:       tenantID,
		Bytes:          cloneBytes,
		RepoURL:        web + "/example-user/somanyad.git",
		IdempotencyKey: "clone-session-1",
		TaskID:         "task_1",
	})
	if err != nil {
		t.Fatalf("meter: %v", err)
	}
	if skipped, _ := got["skipped"].(bool); skipped {
		t.Fatalf("expected metered, got %+v", got)
	}
	res, err := getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if res.TrafficUsedGB != wantGB {
		t.Fatalf("traffic_used_gb=%v want %v", res.TrafficUsedGB, wantGB)
	}
}

func TestMeterGitlabOutboundTraffic_ReplaySameKeyDoesNotDouble(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000102
	const region = "tencent-sh-1"
	web := "https://gitlab-tencent-sh-1.daydaymoney.com"
	seedGitlabMeterFixture(t, tenantID, region, web, 1)

	in := gitlabTrafficMeterReq{
		TenantID:       tenantID,
		Bytes:          8 << 20,
		RepoURL:        web + "/g/r.git",
		IdempotencyKey: "clone-session-replay",
	}
	if _, err := meterGitlabOutboundTraffic(context.Background(), in); err != nil {
		t.Fatalf("first: %v", err)
	}
	res1, _ := getTenantGitlabResourceByRegion(tenantID, region)
	second, err := meterGitlabOutboundTraffic(context.Background(), in)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if reason, _ := second["reason"].(string); reason != "idempotent" {
		t.Fatalf("second reason=%v want idempotent %+v", reason, second)
	}
	res2, _ := getTenantGitlabResourceByRegion(tenantID, region)
	if res2.TrafficUsedGB != res1.TrafficUsedGB {
		t.Fatalf("replay raised counter %v → %v", res1.TrafficUsedGB, res2.TrafficUsedGB)
	}
}

func TestMeterGitlabOutboundTraffic_DifferentRegionDoesNotCollapse(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000103
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("account: %v", err)
	}
	for _, row := range []struct{ slug, web string }{
		{"tencent-sh-1", "https://gitlab-tencent-sh-1.daydaymoney.com"},
		{"tencent-shanghai-5", "https://gitlab.daydaymoney.com"},
	} {
		if _, err := db.Exec(`
			INSERT INTO billing_gitlab_region (
				id, name, slug, description, gitlab_api_base, gitlab_web_url,
				is_active, sort_order, total_disk_gb, total_traffic_gb,
				allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
			) VALUES (?, ?, ?, 'desc', 'http://127.0.0.1:1', ?,
				1, 1, 50, 500, 0, 0, '', 'tencent', ?, ?)
			ON DUPLICATE KEY UPDATE gitlab_web_url = VALUES(gitlab_web_url)`,
			generateSnowflakeID(), row.slug, row.slug, row.web, now, now,
		); err != nil {
			t.Fatalf("region %s: %v", row.slug, err)
		}
		if _, err := db.Exec(`
			INSERT INTO billing_tenant_gitlab_resource (
				tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
				disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
			) VALUES (?, ?, 1, 1, 1, '', 0, 0, 'active', ?, ?)`,
			tenantID, row.slug, now, now,
		); err != nil {
			t.Fatalf("resource %s: %v", row.slug, err)
		}
	}
	if _, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		TenantID: tenantID, Bytes: 5 << 20, RepoURL: "https://gitlab-tencent-sh-1.daydaymoney.com/a/b.git",
		IdempotencyKey: "r1",
	}); err != nil {
		t.Fatalf("sh-1: %v", err)
	}
	if _, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		TenantID: tenantID, Bytes: 6 << 20, RepoURL: "https://gitlab.daydaymoney.com/a/b.git",
		IdempotencyKey: "r5",
	}); err != nil {
		t.Fatalf("sh-5: %v", err)
	}
	sh1, _ := getTenantGitlabResourceByRegion(tenantID, "tencent-sh-1")
	sh5, _ := getTenantGitlabResourceByRegion(tenantID, "tencent-shanghai-5")
	if sh1.TrafficUsedGB <= 0 || sh5.TrafficUsedGB <= 0 {
		t.Fatalf("both regions should have usage sh1=%v sh5=%v", sh1.TrafficUsedGB, sh5.TrafficUsedGB)
	}
	if sh1.TrafficUsedGB == sh5.TrafficUsedGB {
		t.Fatalf("regions collapsed to same used=%v", sh1.TrafficUsedGB)
	}
}

func TestMeterGitlabOutboundTraffic_SkipsGithub(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000104
	seedGitlabMeterFixture(t, tenantID, "tencent-sh-1", "https://gitlab-tencent-sh-1.daydaymoney.com", 1)
	got, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		TenantID: tenantID, Bytes: 10 << 20, RepoURL: "https://github.com/foo/bar.git",
	})
	if err != nil {
		t.Fatalf("meter: %v", err)
	}
	if reason, _ := got["reason"].(string); reason != "not_platform_gitlab" {
		t.Fatalf("reason=%v %+v", reason, got)
	}
	res, _ := getTenantGitlabResourceByRegion(tenantID, "tencent-sh-1")
	if res.TrafficUsedGB != 0 {
		t.Fatalf("github clone must not raise used=%v", res.TrafficUsedGB)
	}
}

func TestHandleInternalChargeGitlabTraffic_BytesNotCeiledToOneGB(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000105
	const region = "tencent-sh-1"
	seedGitlabMeterFixture(t, tenantID, region, "https://gitlab-tencent-sh-1.daydaymoney.com", 1)

	body := `{"tenant_id":"9400000105","bytes":10485760,"repo_url":"https://gitlab-tencent-sh-1.daydaymoney.com/a/b.git","idempotency_key":"http-clone-1","region":"tencent-sh-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/charge-gitlab-traffic/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleInternalChargeGitlabTraffic(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	gb, _ := got["gb"].(float64)
	if gb >= 1 {
		t.Fatalf("must not ceil 10MiB to 1GB, gb=%v body=%s", gb, rr.Body.String())
	}
	if gb <= 0 {
		t.Fatalf("gb must be > 0, got %v", gb)
	}
}

func TestHandleInternalChargeGitlabTraffic_ProjectPathResolvesTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000106
	const region = "tencent-sh-1"
	seedGitlabMeterFixture(t, tenantID, region, "https://gitlab-tencent-sh-1.daydaymoney.com", 1)

	body := `{"project_path":"tenant-9400000106/app","bytes":10485760,"region":"tencent-sh-1","idempotency_key":"wh:corr-project-path"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/charge-gitlab-traffic/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleInternalChargeGitlabTraffic(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	res, err := getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if res.TrafficUsedGB <= 0 || res.TrafficUsedGB >= 1 {
		t.Fatalf("used=%v want (0,1)", res.TrafficUsedGB)
	}
}

func TestHandleInternalChargeGitlabTraffic_ReplayCorrelationDoesNotDouble(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000107
	const region = "tencent-sh-1"
	seedGitlabMeterFixture(t, tenantID, region, "https://gitlab-tencent-sh-1.daydaymoney.com", 1)
	body := `{"project_path":"tenant-9400000107/app","bytes":8388608,"region":"tencent-sh-1","idempotency_key":"wh:corr-replay"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/charge-gitlab-traffic/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handleInternalChargeGitlabTraffic(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("pass %d status=%d body=%s", i, rr.Code, rr.Body.String())
		}
	}
	res, _ := getTenantGitlabResourceByRegion(tenantID, region)
	want := diskUsedGBFromBytes(8388608)
	if res.TrafficUsedGB != want {
		t.Fatalf("used=%v want %v (replay must not double)", res.TrafficUsedGB, want)
	}
}

func TestMeterGitlabOutboundTraffic_MissingRegionSkips(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000108
	seedGitlabMeterFixture(t, tenantID, "tencent-sh-1", "https://gitlab-tencent-sh-1.daydaymoney.com", 1)
	got, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		TenantID: tenantID, Bytes: 10 << 20,
	})
	if err != nil {
		t.Fatalf("meter: %v", err)
	}
	if reason, _ := got["reason"].(string); reason != "missing_region" {
		t.Fatalf("reason=%v %+v", reason, got)
	}
}

func TestMeterGitlabOutboundTraffic_FromCISkips(t *testing.T) {
	got, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		TenantID: 1, Bytes: 10, FromCI: true, Region: "tencent-sh-1",
	})
	if err != nil {
		t.Fatalf("meter: %v", err)
	}
	if reason, _ := got["reason"].(string); reason != "ci_skip" {
		t.Fatalf("reason=%v %+v", reason, got)
	}
}

func TestMeterGitlabOutboundTraffic_UnmappedProjectSkips(t *testing.T) {
	got, err := meterGitlabOutboundTraffic(context.Background(), gitlabTrafficMeterReq{
		Bytes: 10 << 20, Region: "tencent-sh-1", ProjectPath: "unknown-ns/repo",
	})
	if err != nil {
		t.Fatalf("meter: %v", err)
	}
	if reason, _ := got["reason"].(string); reason != "unmapped_project" {
		t.Fatalf("reason=%v %+v", reason, got)
	}
}
