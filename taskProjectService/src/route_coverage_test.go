package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

// OPT-20260808-011 路由覆盖守卫。
//
// 背景：6edee88 重构把旧 /api/tenant/* 分发器替换为约定式 /api/projects/ 路由后，
// 已三次出现「迁移目标路径 → 实际注册路由」错位，落入 handleProjectsRoute 的
// 「项目 ID 分支」返回 404 {"error":"project not found"}（分发丢失类 bug）。
// 本测试以 2026-08-04 API 路径约定审计文档 A4 表为准，对 taskProjectService 的
// 每个 A4 迁移目标 URL 模板做注册级（走 mux）守卫断言：
//   1. 不得返回 404 + "project not found"（分发丢失特征）；
//   2. 关键契约路由额外断言响应体形状（如 progress-systems 必须返回
//      project_progress_systems 字段，而非 deliverable-systems 的数组）。

func doRouteCoverageReq(mux *http.ServeMux, method, url, tenantID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, nil)
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	req.Header.Set("X-Auth-User-Id", "u-route-coverage")
	rec := httptest.NewRecorder()
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)
	return rec
}

func TestRouteCoverage_A4Targets_NoDispatchLoss(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	cases := []struct {
		name             string
		method           string
		url              string
		wantOK           bool // true 时断言 200；false 时仅断言非「404 project not found」
		wantBodyContains string
	}{
		{
			name:   "A4 projects collection",
			method: http.MethodGet,
			url:    "/api/projects/tenant_id/t1/",
			wantOK: true,
		},
		{
			name:   "A4 workspaces collection",
			method: http.MethodGet,
			url:    "/api/projects/workspaces/tenant_id/t1/",
			wantOK: true,
		},
		{
			name:   "A4 deliverable-systems",
			method: http.MethodGet,
			url:    "/api/projects/deliverable-systems/tenant_id/t1/",
			wantOK: true,
		},
		{
			name:   "A4 progress-systems",
			method: http.MethodGet,
			url:    "/api/projects/progress-systems/tenant_id/t1/",
			wantOK: true,
			// FE WorkspaceSettingsStatus.vue/ColumnSystemSettings.vue 依赖该响应形状。
			// 6edee88 误把 progress-systems 并入 handleDeliverableSystemsRoute → 返回
			// deliverable 数组，缺该字段 —— 本断言即第 4 次分发丢失的回归锁。
			wantBodyContains: `"project_progress_systems"`,
		},
		{
			name:   "A4 settings default-progress-system",
			method: http.MethodGet,
			url:    "/api/projects/settings/default-progress-system/tenant_id/t1/",
			wantOK: true,
		},
		{
			name:   "A4 manage-progress-column",
			method: http.MethodGet,
			url:    "/api/projects/manage-progress-column/tenant_id/t1/",
			wantOK: true,
		},
		{
			name:   "A4 manage-deliverable-system",
			method: http.MethodGet,
			url:    "/api/projects/manage-deliverable-system/tenant_id/t1/?workspace_id=ws_rc",
			wantOK: true,
		},
		{
			name:   "A4 workspace-access",
			method: http.MethodGet,
			url:    "/api/projects/workspace-access/workspace-collaborators/tenant_id/t1/?workspace_id=ws_rc",
			wantOK: false,
		},
		{
			name:   "A4 daydaymoney resolve",
			method: http.MethodGet,
			url:    "/api/projects/daydaymoney/tenant_id/t1/resolve?service_id=no-such-service",
			wantOK: true,
		},
		{
			name:   "A4 switch",
			method: http.MethodPost,
			url:    "/api/projects/switch/tenant_id/t1/",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRouteCoverageReq(mux, tc.method, tc.url, "t1")

			body := rec.Body.String()
			// 分发丢失特征：404 {"error":"project not found"}
			if rec.Code == http.StatusNotFound && strings.Contains(body, "project not found") {
				t.Fatalf("dispatch-loss: %s %s → 404 project not found (body=%s)",
					tc.method, tc.url, body)
			}
			if tc.wantOK && rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body=%s)", rec.Code, body)
			}
			if tc.wantBodyContains != "" && !strings.Contains(body, tc.wantBodyContains) {
				t.Fatalf("expected body to contain %q, got %d (body=%s)",
					tc.wantBodyContains, rec.Code, body)
			}
		})
	}
}

// TestRouteCoverage_ProgressSystems_POST — POST 创建走 handleTenantProgressSystems，
// 响应含 project_progress_systems；此前并入 deliverable 分支会走 handleCreateDeliverableSystem，
// 创建进 deliverable 表且返回结构不符。
func TestRouteCoverage_ProgressSystems_POST(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/progress-systems/tenant_id/t1/",
		strings.NewReader(`{"name":"红线防线","columns":[{"name":"待开始"},{"name":"已完成"}]}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %#v", body)
	}
	systems, _ := body["project_progress_systems"].([]interface{})
	if len(systems) != 1 {
		t.Fatalf("expected 1 progress system created, got %#v", body["project_progress_systems"])
	}

	// 校验落库表：必须写 project_progress_systems_tenant，而非 deliverable 表
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM project_progress_systems_tenant WHERE tenant_id='t1'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("expected 1 row in project_progress_systems_tenant, got %d err=%v", n, err)
	}
	var dn int
	if err := db.QueryRow("SELECT COUNT(*) FROM project_deliverable_systems WHERE company_id='t1'").Scan(&dn); err != nil || dn != 0 {
		t.Fatalf("expected 0 rows in project_deliverable_systems, got %d err=%v", dn, err)
	}
}
