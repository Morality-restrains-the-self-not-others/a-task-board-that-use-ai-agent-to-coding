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

// OPT-20260808-004 回归测试：租户默认交付物体系与默认进度体系必须互不覆盖。
//
// 线上根因：project_progress_systems_default_tenant 表 UNIQUE(tenant_id) 每租户仅一行，
// 公司创建链上 deliverable 默认（intent 1）与 progress 默认（intent 2）共用该行，
// intent 2 的 upsert 覆盖 intent 1 写入 → 租户默认交付物体系恒丢失，
// 交付物体系列表页恒显示「暂无默认交付物体系」。
//
// 修复：交付物默认独立存储于 project_deliverable_systems_default_tenant（迁移 011）。

func doDeliverableRequest(method, url string, tenant string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenant)
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)
	return rec
}

func doDeliverableList(tenant string) ([]map[string]interface{}, int) {
	rec := doDeliverableRequest(http.MethodGet, "/api/projects/deliverable-systems/tenant_id/"+tenant, tenant)
	if rec.Code != http.StatusOK {
		return nil, rec.Code
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		return nil, -1
	}
	return list, rec.Code
}

func defaultOf(list []map[string]interface{}) string {
	for _, s := range list {
		if s["is_default"] == true {
			return fmt.Sprint(s["id"])
		}
	}
	return ""
}

// TestDeliverableDefault_SurvivesProgressDefaultSet — 核心回归：
// 设置交付物默认后，再设置进度体系默认，交付物默认不得被覆盖。
func TestDeliverableDefault_SurvivesProgressDefaultSet(t *testing.T) {
	setupTestDB(t)
	const tenant = "t1"

	// 1. 设置交付物默认（ds_default_global 为 007 seed 的系统级体系）
	rec := doDeliverableRequest(http.MethodPost,
		"/api/projects/deliverable-systems/ds_default_global/set-default/tenant_id/"+tenant, tenant)
	if rec.Code != http.StatusOK {
		t.Fatalf("set deliverable default: %d %s", rec.Code, rec.Body.String())
	}

	list, code := doDeliverableList(tenant)
	if code != http.StatusOK {
		t.Fatalf("list after set-default: %d", code)
	}
	if got := defaultOf(list); got != "ds_default_global" {
		t.Fatalf("expected deliverable default=ds_default_global after set-default, got %q", got)
	}

	// 2. 设置进度体系默认（company-created intent 2 的等价路径，曾覆盖交付物默认）
	body := `{"system_id":"ps_default_system"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/settings/default-progress-system/tenant_id/"+tenant+"/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenant)
	req.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("set progress default: %d %s", rec2.Code, rec2.Body.String())
	}

	// 3. 交付物默认必须仍然存在（旧实现此处被覆盖为 ps_default_system → 回归失败）
	list2, code2 := doDeliverableList(tenant)
	if code2 != http.StatusOK {
		t.Fatalf("list after progress set-default: %d", code2)
	}
	if got := defaultOf(list2); got != "ds_default_global" {
		t.Fatalf("deliverable default was clobbered by progress default: got %q, want ds_default_global", got)
	}

	// 4. 进度默认行保持 system/ps_default_system（互不干扰）
	var targetType, targetID string
	if err := db.QueryRow(
		"SELECT target_type, target_id FROM project_progress_systems_default_tenant WHERE tenant_id=?", tenant,
	).Scan(&targetType, &targetID); err != nil {
		t.Fatalf("progress default row missing: %v", err)
	}
	if targetType != "system" || targetID != "ps_default_system" {
		t.Fatalf("progress default changed: type=%s id=%s", targetType, targetID)
	}
}

// TestGetDefaultDeliverableSystem_NoDefault — 无默认时返回 success + nil id（不 404）。
func TestGetDefaultDeliverableSystem_NoDefault(t *testing.T) {
	setupTestDB(t)

	rec := doDeliverableRequest(http.MethodGet, "/api/projects/default-deliverable-system/tenant_id/t1", "t1")
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
	if body["default_deliverable_system_id"] != nil {
		t.Fatalf("expected nil default id, got %v", body["default_deliverable_system_id"])
	}
}

// TestGetDefaultDeliverableSystem_AfterSetDefault — 设置后端点返回默认 id（前端契约）。
func TestGetDefaultDeliverableSystem_AfterSetDefault(t *testing.T) {
	setupTestDB(t)

	rec := doDeliverableRequest(http.MethodPost,
		"/api/projects/deliverable-systems/ds_default_global/set-default/tenant_id/t1", "t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("set-default: %d %s", rec.Code, rec.Body.String())
	}

	rec2 := doDeliverableRequest(http.MethodGet, "/api/projects/default-deliverable-system/tenant_id/t1", "t1")
	if rec2.Code != http.StatusOK {
		t.Fatalf("get default: %d %s", rec2.Code, rec2.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["default_deliverable_system_id"] != "ds_default_global" {
		t.Fatalf("expected ds_default_global, got %v", body["default_deliverable_system_id"])
	}
}

// TestDeleteDefaultDeliverableSystem_Rejected — 默认交付物体系不可删除（独立表检查）。
func TestDeleteDefaultDeliverableSystem_Rejected(t *testing.T) {
	setupTestDB(t)
	const tenant = "t1"

	// 创建公司级交付物体系（非系统级，才能命中"不能删除默认"检查）
	createBody := `{"name":"公司默认体系","level_names":["价值流","活动"]}`
	reqCreate := httptest.NewRequest(http.MethodPost,
		"/api/projects/deliverable-systems/tenant_id/"+tenant, strings.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("X-Auth-Tenant-Id", tenant)
	reqCreate.Header.Set("X-Auth-User-Id", "u1")
	recCreate := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create system: %d %s", recCreate.Code, recCreate.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(recCreate.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	sysID := fmt.Sprint(created["id"])

	// 设为公司默认
	rec := doDeliverableRequest(http.MethodPost,
		"/api/projects/deliverable-systems/"+sysID+"/set-default/tenant_id/"+tenant, tenant)
	if rec.Code != http.StatusOK {
		t.Fatalf("set-default: %d %s", rec.Code, rec.Body.String())
	}

	// 删除默认体系 → 400
	recDel := doDeliverableRequest(http.MethodDelete,
		"/api/projects/deliverable-systems/"+sysID+"/tenant_id/"+tenant+"/", tenant)
	if recDel.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 deleting default, got %d: %s", recDel.Code, recDel.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(recDel.Body).Decode(&body); err != nil {
		t.Fatalf("decode delete: %v", err)
	}
	if body["message"] != "不能删除默认交付物体系" {
		t.Fatalf("expected 不能删除默认交付物体系, got %v", body["message"])
	}
}
