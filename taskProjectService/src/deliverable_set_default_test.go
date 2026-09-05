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

// OPT-20260807-053 回归测试：
//  1. set-default 对不存在的交付物体系 id（如前端曾硬编码的 "1"）返回 404 + 响应体 trace_id + 响应头 X-Trace-Id；
//  2. 对存在的系统级交付物体系（ds_default_global，迁移 seed）返回 200 success，且 systems 带 is_default 标注。
func doSetDefaultDeliverable(method, systemID, tenant, traceID string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/projects/deliverable-systems/%s/set-default/tenant_id/%s/", systemID, tenant)
	req := httptest.NewRequest(method, url, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenant)
	req.Header.Set("X-Auth-User-Id", "u1")
	if traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
		// 生产 FE 由 attachTracePropagationHeaders 携带 span 头；缺省会触发 RejectTraceIdOnlyHTTP 400
		req.Header.Set("X-Parent-Span-Id", "0e833d09bd0f7d72")
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	// 与 main.go ListenAndServe 一致的完整链路：tracelog.Middleware 负责回传 X-Trace-Id 响应头
	tracelog.Middleware(gatewayUserMiddleware(mux)).ServeHTTP(rec, req)
	return rec
}

func TestSetDefaultDeliverableSystem_NotFound_WithTraceID(t *testing.T) {
	setupTestDB(t)

	const traceID = "test-trace-20260807-notfound"
	rec := doSetDefaultDeliverable(http.MethodPost, "1", "t1", traceID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	// 响应头必须回传 X-Trace-Id（tracelog.Middleware 兜底）
	if got := rec.Header().Get("X-Trace-Id"); got != traceID {
		t.Errorf("expected response header X-Trace-Id=%q, got %q", traceID, got)
	}
	// 响应体必须带 trace_id（writeError 兜底，防响应头被剥离）
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("expected status=error, got %v", body["status"])
	}
	if body["message"] != "交付物体系不存在" {
		t.Errorf("expected message=交付物体系不存在, got %v", body["message"])
	}
	if body["trace_id"] != traceID {
		t.Errorf("expected body trace_id=%q, got %v", traceID, body["trace_id"])
	}
}

func TestSetDefaultDeliverableSystem_Success_SystemID(t *testing.T) {
	setupTestDB(t)

	// ds_default_global 为迁移 seed 的系统级交付物体系（id 为 ds_xxx 而非数字 1）
	const systemID = "ds_default_global"
	rec := doSetDefaultDeliverable(http.MethodPost, systemID, "t1", "test-trace-20260807-ok")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	systems, ok := body["systems"].([]interface{})
	if !ok || len(systems) == 0 {
		t.Fatalf("expected non-empty systems array, got %v", body["systems"])
	}
	// 目标体系必须被标注为默认
	foundDefault := false
	for _, s := range systems {
		m := s.(map[string]interface{})
		if m["id"] == systemID {
			foundDefault = true
			if m["is_default"] != true {
				t.Errorf("expected is_default=true for %s, got %v", systemID, m["is_default"])
			}
		}
	}
	if !foundDefault {
		t.Errorf("expected %s in systems list", systemID)
	}
	// 租户默认记录必须已 upsert（OPT-20260808-004：交付物默认独立表，
	// 不再写 progress 默认表，避免被进度体系默认覆盖）
	var targetID string
	if err := db.QueryRow(
		"SELECT deliverable_id FROM project_deliverable_systems_default_tenant WHERE tenant_id='t1'",
	).Scan(&targetID); err != nil {
		t.Fatalf("default_tenant row missing: %v", err)
	}
	if targetID != systemID {
		t.Errorf("expected tenant default=%s, got %s", systemID, targetID)
	}
}
