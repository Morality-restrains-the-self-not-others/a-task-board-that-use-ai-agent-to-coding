package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

func resetGateMetricsForTest() {
	gateMetricsMu.Lock()
	gateCodeCount = map[string]int64{}
	gateMetricsMu.Unlock()
}

func seedGateTenant(t *testing.T, tenantID int64) {
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
}

// OPT-20260823-062: 闸门判定按 code 计数，供 Grafana 观测 fail-open 静默失效。
func TestGateCodeMetricsIncrementedOnEvaluation(t *testing.T) {
	resetGateMetricsForTest()
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGateTenant(t, tenantID)

	// 未映射 project → fail-open UNMAPPED（不触库）
	evaluateGitlabTrafficGate(gitlabTrafficGateRequest{ProjectPath: "personal/repo"})
	// 未预购 → TRAFFIC_NOT_PURCHASED
	evaluateGitlabTrafficGate(gitlabTrafficGateRequest{TenantID: tenantID, Region: "tencent-sh-1"})
	// CI → CI_SKIP
	evaluateGitlabTrafficGate(gitlabTrafficGateRequest{TenantID: tenantID, Region: "tencent-sh-1", FromCI: true})
	// 内网 → INTRANET_SKIP
	evaluateGitlabTrafficGate(gitlabTrafficGateRequest{TenantID: tenantID, Region: "tencent-sh-1", IsIntranet: true})

	gateMetricsMu.Lock()
	defer gateMetricsMu.Unlock()
	if gateCodeCount[trafficGateUnmapped] != 1 {
		t.Fatalf("UNMAPPED count=%d want 1", gateCodeCount[trafficGateUnmapped])
	}
	if gateCodeCount[trafficGateNotPurchased] != 1 {
		t.Fatalf("NOT_PURCHASED count=%d want 1", gateCodeCount[trafficGateNotPurchased])
	}
	if gateCodeCount[trafficGateCI] != 1 {
		t.Fatalf("CI_SKIP count=%d want 1", gateCodeCount[trafficGateCI])
	}
	if gateCodeCount[trafficGateIntranet] != 1 {
		t.Fatalf("INTRANET_SKIP count=%d want 1", gateCodeCount[trafficGateIntranet])
	}
}

// OPT-20260823-033: 生产链路 tracelog.MetricsMiddleware 包裹 mux 后，请求路径应
// 进入 http_request_duration_seconds 直方图（供 Grafana 观测 wechat 关联账号查单 p95）。
func TestMetricsMiddlewareRecordsPathDurationHistogram(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	handler := tracelog.MetricsMiddleware(mux)

	// 走一次业务路由（health），确认 MetricsMiddleware 能捕获路径与状态码
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("health status=%d", rr.Code)
	}

	// 读 /metrics，断言该路径出现于直方图桶与计数（_count 至少 1）
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("metrics status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	body := rr2.Body.String()
	if !strings.Contains(body, `http_request_duration_seconds_bucket{method="GET",path="/api/health"`) {
		t.Fatalf("metrics missing duration histogram for /api/health, body=%s", body)
	}
	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",path="/api/health"} `) {
		t.Fatalf("metrics missing duration count for /api/health, body=%s", body)
	}
	if !strings.Contains(body, `http_requests_total{method="GET",path="/api/health",status="200"} `) {
		t.Fatalf("metrics missing request counter for /api/health, body=%s", body)
	}
}

func TestMetricsEndpointIncludesGateCounters(t *testing.T) {
	resetGateMetricsForTest()
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tenantID int64 = 877397588196749399
	seedGateTenant(t, tenantID)
	evaluateGitlabTrafficGate(gitlabTrafficGateRequest{ProjectPath: "x/y"})
	evaluateGitlabTrafficGate(gitlabTrafficGateRequest{TenantID: tenantID, Region: "tencent-sh-1"})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `taskBill_gitlab_traffic_gate_total{code="UNMAPPED_PROJECT"} 1`) {
		t.Fatalf("metrics missing UNMAPPED counter, body=%s", body)
	}
	if !strings.Contains(body, `taskBill_gitlab_traffic_gate_total{code="TRAFFIC_NOT_PURCHASED"} 1`) {
		t.Fatalf("metrics missing NOT_PURCHASED counter, body=%s", body)
	}
	// 同时确认 tracelog HTTP RED 指标仍存在（聚合未破坏）
	if !strings.Contains(body, "_http_requests_total") && !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics missing tracelog http counters, body=%s", body)
	}
	// 非 GET 拒绝
	req2 := httptest.NewRequest(http.MethodPost, "/metrics", strings.NewReader(""))
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /metrics status=%d want 405", rr2.Code)
	}
}
