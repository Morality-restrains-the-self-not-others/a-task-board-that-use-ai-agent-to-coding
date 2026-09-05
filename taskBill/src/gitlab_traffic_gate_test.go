package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitlabTrafficDownloadAllowed(t *testing.T) {
	cases := []struct {
		name     string
		prepaid  int64
		used     float64
		fromCI   bool
		intranet bool
		wantOK   bool
		wantCode string
	}{
		{"not purchased", 0, 0, false, false, false, trafficGateNotPurchased},
		{"not purchased with disk watermark", 0, 0.01049, false, false, false, trafficGateNotPurchased},
		{"exceeded", 1, 1.0, false, false, false, trafficGateExceeded},
		{"exceeded over", 1, 1.2, false, false, false, trafficGateExceeded},
		{"remaining", 5, 0.01, false, false, true, trafficGateOK},
		{"task node public unpaid still gated", 0, 9, false, false, false, trafficGateNotPurchased},
		{"task node public exceeded still gated", 1, 9, false, false, false, trafficGateExceeded},
		{"true intranet unpaid skipped", 0, 9, false, true, true, trafficGateIntranet},
		{"ci unpaid skipped", 0, 9, true, false, true, trafficGateCI},
		{"ci exceeded skipped", 1, 9, true, false, true, trafficGateCI},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, code := gitlabTrafficDownloadAllowed(tc.prepaid, tc.used, tc.fromCI, tc.intranet)
			if ok != tc.wantOK || code != tc.wantCode {
				t.Fatalf("allowed=%v code=%s, want %v %s", ok, code, tc.wantOK, tc.wantCode)
			}
		})
	}
}

func TestParseTenantIDFromGitlabProjectPath(t *testing.T) {
	id, err := parseTenantIDFromGitlabProjectPath("tenant-877397588196749312/valueStream")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if id != 877397588196749312 {
		t.Fatalf("id=%d", id)
	}
	if _, err := parseTenantIDFromGitlabProjectPath("ljy/repo"); err == nil {
		t.Fatal("user namespace must not parse")
	}
	if _, err := parseTenantIDFromGitlabProjectPath("tenant-abc/repo"); err == nil {
		t.Fatal("non-numeric tenant path must not parse")
	}
}

func stubUsernameResolver(t *testing.T, fn func(string, string) (int64, error)) {
	t.Helper()
	old := resolveTenantIDFromGitlabUsername
	resolveTenantIDFromGitlabUsername = fn
	t.Cleanup(func() { resolveTenantIDFromGitlabUsername = old })
}

func TestEvaluateGitlabTrafficGate_UnmappedProjectFailOpen(t *testing.T) {
	// cfg.TrafficGateUnmappedReject 默认 false → fail-open 保持（解析失败不阻断）
	stubUsernameResolver(t, func(_, _ string) (int64, error) { return 0, fmt.Errorf("stub unresolved") })
	old := cfg.TrafficGateUnmappedReject
	cfg.TrafficGateUnmappedReject = false
	defer func() { cfg.TrafficGateUnmappedReject = old }()
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath: "user/orphan",
		Region:      "tencent-sh-1",
	})
	if !got.Allowed || got.Code != trafficGateUnmapped {
		t.Fatalf("got allowed=%v code=%s", got.Allowed, got.Code)
	}
}

func TestEvaluateGitlabTrafficGate_UnmappedProjectReject(t *testing.T) {
	// OPT-20260823-013: 开关置 true 后未映射 tenant-{id} 组的仓默认拒绝，且带 region/说明
	stubUsernameResolver(t, func(_, _ string) (int64, error) { return 0, fmt.Errorf("stub unresolved") })
	old := cfg.TrafficGateUnmappedReject
	cfg.TrafficGateUnmappedReject = true
	defer func() { cfg.TrafficGateUnmappedReject = old }()
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath: "user/orphan",
		Region:      "tencent-sh-1",
	})
	if got.Allowed {
		t.Fatal("reject-mode must deny unmapped project")
	}
	if got.Code != trafficGateUnmapped {
		t.Fatalf("code=%s want %s", got.Code, trafficGateUnmapped)
	}
	if got.Region != "tencent-sh-1" {
		t.Fatalf("region=%s want tencent-sh-1", got.Region)
	}
	if !strings.Contains(got.Message, "reject-mode") {
		t.Fatalf("message=%q want reject-mode hint", got.Message)
	}
}

func TestHandleInternalGitlabTrafficGate_NotPurchased(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
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
		) VALUES (?, '上海一区', 'tencent-sh-1', 'desc', 'http://127.0.0.1:1', 'https://example.com',
			1, 1, 50, 500, 0, 0, '', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		generateSnowflakeID(), now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, 'tencent-sh-1', 1, 0, 1, '', 11263830, 0.01049, 'active', ?, ?)`,
		tenantID, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	body := `{"project_path":"tenant-877397588196749399/demo","region":"tencent-sh-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/gitlab-traffic-gate/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleInternalGitlabTrafficGate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if allowed, _ := got["allowed"].(bool); allowed {
		t.Fatalf("expected deny, body=%v", got)
	}
	if got["code"] != trafficGateNotPurchased {
		t.Fatalf("code=%v", got["code"])
	}
}

func TestEvaluateGitlabTrafficGate_IntranetAllowedWhenUnpaid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
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
		) VALUES (?, '上海一区', 'tencent-sh-1', 'desc', 'http://127.0.0.1:1', 'https://example.com',
			1, 1, 50, 500, 0, 0, '', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		generateSnowflakeID(), now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, 'tencent-sh-1', 1, 0, 1, '', 11263830, 0.01049, 'active', ?, ?)`,
		tenantID, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath: "tenant-877397588196749399/demo",
		Region:      "tencent-sh-1",
		IsIntranet:  true,
		FromCI:      false,
	})
	if !got.Allowed {
		t.Fatalf("same-region intranet must skip quota when unpaid, got %+v", got)
	}
	if got.Code != trafficGateIntranet {
		t.Fatalf("code=%s want %s", got.Code, trafficGateIntranet)
	}

	ci := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath: "tenant-877397588196749399/demo",
		Region:      "tencent-sh-1",
		IsIntranet:  true,
		FromCI:      true,
	})
	if !ci.Allowed || ci.Code != trafficGateCI {
		t.Fatalf("CI must skip quota, got allowed=%v code=%s", ci.Allowed, ci.Code)
	}
}

func TestRefreshTenantDiskUsageDoesNotRaiseTrafficFloor(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9000000192
	const region = "tencent-sh-1"
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, 0, 1, '', 1024, 0.01049, 'active', ?, ?)`,
		tenantID, region, now, now,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := reportGitlabDiskUsage(tenantID, 50<<20, region); err != nil {
		t.Fatalf("disk report: %v", err)
	}
	res, err := getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if res.TrafficUsedGB != 0.01049 {
		t.Fatalf("traffic_used_gb raised by disk refresh: %v", res.TrafficUsedGB)
	}
	if res.DiskUsedBytes != 50<<20 {
		t.Fatalf("disk_used_bytes=%d", res.DiskUsedBytes)
	}
}

// ── OPT-20260824-079：gitlab 用户名 → 租户归集（个人命名空间仓不再绕过流量闸门）──

func seedGitlabResource(t *testing.T, tenantID int64, region string, prepaidGB int64, usedGB float64) {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, ?, 1, '', 0, ?, 'active', ?, ?)`,
		tenantID, region, prepaidGB, usedGB, now, now,
	); err != nil {
		t.Fatalf("seed resource: %v", err)
	}
}

func TestEvaluateGitlabTrafficGate_PersonalNamespaceUnpaidBlocked(t *testing.T) {
	// BDD：容器克隆 example-user/somanyad（个人命名空间），租户流量未预购 → 必须阻断
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGitlabResource(t, tenantID, "tencent-sh-1", 0, 0.01049)

	var gotUsername string
	stubUsernameResolver(t, func(username, region string) (int64, error) {
		gotUsername = username
		if region != "tencent-sh-1" {
			return 0, fmt.Errorf("unexpected region %s", region)
		}
		return tenantID, nil
	})

	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "example-user/somanyad",
		Region:         "tencent-sh-1",
		GitlabUsername: "example-user",
	})
	if got.Allowed {
		t.Fatalf("personal-namespace clone with unpaid tenant must be blocked, got %+v", got)
	}
	if got.Code != trafficGateNotPurchased {
		t.Fatalf("code=%s want %s", got.Code, trafficGateNotPurchased)
	}
	if got.TenantID != fmt.Sprintf("%d", tenantID) {
		t.Fatalf("tenant_id=%q want %d", got.TenantID, tenantID)
	}
	if gotUsername == "" {
		t.Fatal("resolver not called")
	}
}

func TestEvaluateGitlabTrafficGate_PersonalNamespacePaidAllowed(t *testing.T) {
	// BDD：租户有预购流量且未超 → 个人命名空间仓正常放行（解析不误伤）
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGitlabResource(t, tenantID, "tencent-sh-1", 5, 0.01)

	stubUsernameResolver(t, func(_, region string) (int64, error) {
		return tenantID, nil
	})
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "example-user/somanyad",
		Region:         "tencent-sh-1",
		GitlabUsername: "example-user",
	})
	if !got.Allowed || got.Code != trafficGateOK {
		t.Fatalf("paid tenant must pass, got allowed=%v code=%s", got.Allowed, got.Code)
	}
}

func TestEvaluateGitlabTrafficGate_NamespaceDerivationWithoutUsername(t *testing.T) {
	// BDD：rails 未传认证用户名时，按 project_path 顶层命名空间归集（仓所有者）
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGitlabResource(t, tenantID, "tencent-sh-1", 0, 0)

	var gotUsername string
	stubUsernameResolver(t, func(username, _ string) (int64, error) {
		gotUsername = username
		return tenantID, nil
	})
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath: "example-user/somanyad",
		Region:      "tencent-sh-1",
	})
	if got.Allowed || got.Code != trafficGateNotPurchased {
		t.Fatalf("namespace-derived resolution must gate, got allowed=%v code=%s", got.Allowed, got.Code)
	}
	if gotUsername != "example-user" {
		t.Fatalf("resolver called with %q, want example-user", gotUsername)
	}
}

func TestEvaluateGitlabTrafficGate_GroupProjectFallsBackToAuthenticatedUser(t *testing.T) {
	// BDD：组仓 devteam/repo（无 tenant-{id} 前缀）→ 命名空间非用户名解析失败 → 认证用户归集
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGitlabResource(t, tenantID, "tencent-sh-1", 0, 0)

	var calls []string
	stubUsernameResolver(t, func(username, _ string) (int64, error) {
		calls = append(calls, username)
		if username == "devteam" {
			return 0, fmt.Errorf("no credential binding for devteam")
		}
		return tenantID, nil
	})
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "devteam/repo",
		Region:         "tencent-sh-1",
		GitlabUsername: "example-user",
	})
	if got.Allowed || got.Code != trafficGateNotPurchased {
		t.Fatalf("group project must gate via authenticated user, got allowed=%v code=%s", got.Allowed, got.Code)
	}
	if len(calls) != 2 || calls[0] != "devteam" || calls[1] != "example-user" {
		t.Fatalf("resolver calls=%v want [devteam example-user]", calls)
	}
}

func TestEvaluateGitlabTrafficGate_TenantPathSkipsUsernameResolution(t *testing.T) {
	// BDD：tenant-{id} 前缀仓直接按租户计费，不触发用户名解析
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGitlabResource(t, tenantID, "tencent-sh-1", 0, 0)

	resolverCalled := false
	stubUsernameResolver(t, func(_, _ string) (int64, error) {
		resolverCalled = true
		return 0, fmt.Errorf("must not be called")
	})
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "tenant-877397588196749399/demo",
		Region:         "tencent-sh-1",
		GitlabUsername: "example-user",
	})
	if got.Allowed || got.Code != trafficGateNotPurchased {
		t.Fatalf("tenant-path must gate, got allowed=%v code=%s", got.Allowed, got.Code)
	}
	if resolverCalled {
		t.Fatal("resolver must not run for tenant-{id} project path")
	}
}

func TestEvaluateGitlabTrafficGate_UsernameUnresolvedKeepsFailOpen(t *testing.T) {
	// BDD：解析失败（无绑定/多租户歧义）→ UNMAPPED fail-open 保持，不误伤
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	stubUsernameResolver(t, func(_, _ string) (int64, error) {
		return 0, fmt.Errorf("ambiguous tenant set")
	})
	got := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "someone/else",
		Region:         "tencent-sh-1",
		GitlabUsername: "someone",
	})
	if !got.Allowed || got.Code != trafficGateUnmapped {
		t.Fatalf("unresolved username must fail-open, got allowed=%v code=%s", got.Allowed, got.Code)
	}
}

func TestEvaluateGitlabTrafficGate_CIAndIntranetSkipResolution(t *testing.T) {
	// BDD：CI 与同区域内网不做用户名解析（无 DB 命中）；未映射路径本就 fail-open 放行。
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	resolverCalled := false
	stubUsernameResolver(t, func(_, _ string) (int64, error) {
		resolverCalled = true
		return 0, fmt.Errorf("must not be called")
	})
	ci := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "example-user/somanyad",
		Region:         "tencent-sh-1",
		GitlabUsername: "example-user",
		FromCI:         true,
	})
	if !ci.Allowed {
		t.Fatalf("CI must stay allowed, got allowed=%v code=%s", ci.Allowed, ci.Code)
	}
	net := evaluateGitlabTrafficGate(gitlabTrafficGateRequest{
		ProjectPath:    "example-user/somanyad",
		Region:         "tencent-sh-1",
		GitlabUsername: "example-user",
		IsIntranet:     true,
	})
	if !net.Allowed {
		t.Fatalf("intranet must stay allowed, got allowed=%v code=%s", net.Allowed, net.Code)
	}
	if resolverCalled {
		t.Fatal("resolver must not run for CI/intranet")
	}
}
