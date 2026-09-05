package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

// OPT-20260808 回归测试：work-panel 创建任务弹窗「任务类型」加载失败。
//
// 生产故障（traceId 17860cc3-a968-4e83-ad56-f37e46519833）：taskFE
// fetchTaskTypes 请求 GET /api/projects/manage-deliverable-system/tenant_id/{tid}
// 命中 taskProjectService 404 "project not found"。根因：handleProjectsRoute 的
// 动作分发缺 manage-deliverable-system → 落入「项目 ID 子资源」分支 → handleGetProject
// 用 X-Resource-Id="manage-deliverable-system"（key）按 ID 查 project_entries →
// sql.ErrNoRows → 404。handleLegacyManageDeliverableSystem（OPT-049）定义了但从未挂载。

func seedManageDeliverableSystem(t *testing.T, tenantID, wsID, systemID string, withSystem bool) {
	t.Helper()
	if withSystem {
		if _, err := db.Exec(
			"INSERT INTO project_deliverable_systems(id,name,description,company_id,is_system) VALUES(?,?,?,?,0)",
			systemID, "测试交付体系", "seed", tenantID); err != nil {
			t.Fatalf("seed system: %v", err)
		}
		for i, ln := range []string{"待开发", "开发中", "已完成"} {
			if _, err := db.Exec(
				"INSERT INTO project_deliverable_columns(id,system_id,name,order_num) VALUES(?,?,?,?)",
				"dc_"+systemID+fmt.Sprintf("_%d", i), systemID, ln, i); err != nil {
				t.Fatalf("seed column: %v", err)
			}
		}
	}
	if _, err := db.Exec(
		`INSERT INTO project_workspace_entries(id,name,description,company_id,deliverable_system_id) VALUES(?,?,?,?,?)`,
		wsID, "ws", "seed", tenantID, systemID); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
}

func doManageDeliverableSystemGET(tenantID, wsID, traceID string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/projects/manage-deliverable-system/tenant_id/%s?workspace_id=%s", tenantID, wsID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "u1")
	if traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
		req.Header.Set("X-Parent-Span-Id", "0e833d09bd0f7d72")
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)
	return rec
}

// Red 前行为：404 {"error":"project not found"} —— 与生产故障一致。
func TestManageDeliverableSystemRoute_GET_TaskTypes(t *testing.T) {
	setupTestDB(t)
	seedManageDeliverableSystem(t, "t1", "ws_mds", "ds_mds", true)

	rec := doManageDeliverableSystemGET("t1", "ws_mds", "test-trace-mds-20260808")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Errorf("expected status=success, got %v", body["status"])
	}
	if body["current_deliverable_system_id"] != "ds_mds" {
		t.Errorf("expected current_deliverable_system_id=ds_mds, got %v", body["current_deliverable_system_id"])
	}
	objs, ok := body["current_deliverable_objs"].([]interface{})
	if !ok || len(objs) != 3 {
		t.Fatalf("expected 3 deliverable objs, got %v", body["current_deliverable_objs"])
	}
	first := objs[0].(map[string]interface{})
	if first["name"] != "待开发" {
		t.Errorf("expected first obj name=待开发, got %v", first["name"])
	}
	if first["order"] == nil {
		t.Errorf("expected order field present, got %v", first)
	}
}

// 工作空间未绑定交付物体系时返回空数组（前端回退默认任务状态），而非报错。
func TestManageDeliverableSystemRoute_GET_NoWorkspaceSystem(t *testing.T) {
	setupTestDB(t)
	seedManageDeliverableSystem(t, "t1", "ws_mds_empty", "", false)

	rec := doManageDeliverableSystemGET("t1", "ws_mds_empty", "test-trace-mds-empty")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if objs, ok := body["current_deliverable_objs"].([]interface{}); !ok || len(objs) != 0 {
		t.Errorf("expected empty deliverable objs, got %v", body["current_deliverable_objs"])
	}
}

// 租户隔离：t1 请求 t2 的工作空间时不得返回 t2 的交付物列（不泄漏跨租户数据）。
func TestManageDeliverableSystemRoute_GET_CrossTenantIsolation(t *testing.T) {
	setupTestDB(t)
	seedManageDeliverableSystem(t, "t2", "ws_mds_t2", "ds_mds_t2", true)

	rec := doManageDeliverableSystemGET("t1", "ws_mds_t2", "test-trace-mds-x-tenant")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if objs, ok := body["current_deliverable_objs"].([]interface{}); !ok || len(objs) != 0 {
		t.Errorf("cross-tenant leak: expected empty objs, got %v", body["current_deliverable_objs"])
	}
}

// ============================================================================
// 同类回归（同源 6edee88）：manage-progress-column 动作分发被移除。
// WorkspaceSettingsStatus.vue 调用 GET/POST /api/projects/manage-progress-column/
// tenant_id/{tid}，6edee88 前挂载 handleLegacyManageProgressColumn，重构后与
// manage-deliverable-system 一并丢失 → handleGetProject 404 "project not found"。
// ============================================================================

func seedProgressSystemWithColumns(t *testing.T, systemID string) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO project_progress_systems(id,name) VALUES(?,?)", systemID, "进度体系"); err != nil {
		t.Fatalf("seed progress system: %v", err)
	}
	for i, cn := range []string{"待处理", "进行中", "已完成"} {
		if _, err := db.Exec(
			"INSERT INTO project_progress_columns(id,system_id,name,order_num) VALUES(?,?,?,?)",
			"pc_"+systemID+fmt.Sprintf("_%d", i), systemID, cn, i); err != nil {
			t.Fatalf("seed progress column: %v", err)
		}
	}
}

func doManageProgressColumn(method, tenantID, form string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/projects/manage-progress-column/tenant_id/%s", tenantID)
	if method == http.MethodGet {
		url += "?workspace_id=ws_mpc"
	}
	var body *strings.Reader
	if form != "" {
		body = strings.NewReader(form)
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "u1")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)
	return rec
}

// Red 前行为：GET 404 {"error":"project not found"}（与 manage-deliverable-system 同源）。
func TestManageProgressColumnRoute_GET(t *testing.T) {
	setupTestDB(t)
	if _, err := db.Exec(
		"INSERT INTO project_progress_systems_workspace(id,tenant_id,workspace_id,target_type,target_id) VALUES(?,?,?,?,?)",
		"psw_mpc", "t1", "ws_mpc", "system", "ps_mpc"); err != nil {
		t.Fatalf("seed workspace binding: %v", err)
	}

	rec := doManageProgressColumn(http.MethodGet, "t1", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Errorf("expected status=success, got %v", body["status"])
	}
	if body["current_progress_system_id"] != "ps_mpc" {
		t.Errorf("expected current_progress_system_id=ps_mpc, got %v", body["current_progress_system_id"])
	}
}

// POST update_progress_system（legacy form-urlencoded 契约）须能绑定体系并返回列。
func TestManageProgressColumnRoute_POST_UpdateProgressSystem(t *testing.T) {
	setupTestDB(t)
	seedProgressSystemWithColumns(t, "ps_mpc2")

	rec := doManageProgressColumn(http.MethodPost, "t1",
		"action=update_progress_system&workspace_id=ws_mpc2&progress_system_id=ps_mpc2")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	if body["progress_system_id"] != "ps_mpc2" {
		t.Errorf("expected progress_system_id=ps_mpc2, got %v", body["progress_system_id"])
	}
	cols, ok := body["columns"].([]interface{})
	if !ok || len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %v", body["columns"])
	}
	// 绑定必须持久化到租户+工作空间作用域行。
	var targetID string
	if err := db.QueryRow(
		"SELECT target_id FROM project_progress_systems_workspace WHERE tenant_id='t1' AND workspace_id='ws_mpc2'",
	).Scan(&targetID); err != nil {
		t.Fatalf("binding row missing: %v", err)
	}
	if targetID != "ps_mpc2" {
		t.Errorf("expected persisted target_id=ps_mpc2, got %s", targetID)
	}
}

// POST 更新交付物体系须按租户作用域：跨租户工作空间不得被静默更新。
func TestManageDeliverableSystemRoute_POST_TenantScoped(t *testing.T) {
	setupTestDB(t)
	seedManageDeliverableSystem(t, "t2", "ws_mds_t2", "ds_mds_t2", true)
	seedManageDeliverableSystem(t, "t1", "ws_mds_t1", "", false)

	doPOST := func(wsID string) *httptest.ResponseRecorder {
		url := fmt.Sprintf("/api/projects/manage-deliverable-system/tenant_id/t1")
		form := fmt.Sprintf("action=update_deliverable_system&workspace_id=%s&deliverable_system_id=ds_mds_t2", wsID)
		req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		rec := httptest.NewRecorder()
		mux := http.NewServeMux()
		mountRoutes(mux)
		tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)
		return rec
	}

	// 跨租户工作空间：必须拒绝（404），且数据不被修改。
	rec := doPOST("ws_mds_t2")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant POST: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	var dsID string
	if err := db.QueryRow("SELECT deliverable_system_id FROM project_workspace_entries WHERE id='ws_mds_t2'").Scan(&dsID); err != nil {
		t.Fatalf("query ws_mds_t2: %v", err)
	}
	if dsID != "ds_mds_t2" {
		t.Errorf("cross-tenant update leaked: expected ds_mds_t2 unchanged, got %s", dsID)
	}

	// 本租户工作空间：正常更新。
	rec = doPOST("ws_mds_t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("own-tenant POST: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := db.QueryRow("SELECT deliverable_system_id FROM project_workspace_entries WHERE id='ws_mds_t1'").Scan(&dsID); err != nil {
		t.Fatalf("query ws_mds_t1: %v", err)
	}
	if dsID != "ds_mds_t2" {
		t.Errorf("expected ws_mds_t1 updated to ds_mds_t2, got %s", dsID)
	}
}
