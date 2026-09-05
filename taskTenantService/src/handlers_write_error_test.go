package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

// OPT-20260808-010e: 错误响应统一注入 trace_id（X-Trace-Id 头优先，
// context 兜底），并保持 FE 读取顺序契约（detail 优先）。

func decodeBodyTenant(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body not json: %v (%s)", err, rec.Body.String())
	}
	return body
}

func TestWriteErrorInjectTraceIDFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/groups", nil)
	req.Header.Set("X-Trace-Id", "trace-tenant-1")
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusForbidden, "分组不存在")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	body := decodeBodyTenant(t, rec)
	if body["trace_id"] != "trace-tenant-1" {
		t.Errorf("trace_id = %v, want trace-tenant-1", body["trace_id"])
	}
	if body["error"] != "分组不存在" || body["message"] != "分组不存在" {
		t.Errorf("error/message = %v/%v", body["error"], body["message"])
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

func TestWriteErrorInjectTraceIDFromContext(t *testing.T) {
	ctx := tracelog.ContextWithTraceID(context.Background(), "trace-tenant-ctx-2")
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/groups", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusInternalServerError, "db error")
	if body := decodeBodyTenant(t, rec); body["trace_id"] != "trace-tenant-ctx-2" {
		t.Errorf("trace_id = %v, want trace-tenant-ctx-2", body["trace_id"])
	}
}

func TestWriteErrorNilRequest(t *testing.T) {
	// 无 r 作用域的内部 helper 传 nil：不注入、不 panic
	rec := httptest.NewRecorder()
	writeError(rec, nil, http.StatusBadGateway, "upstream failed")
	body := decodeBodyTenant(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present with nil request: %v", body["trace_id"])
	}
}

func TestWriteErrorDetailContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/groups", nil)
	req.Header.Set("X-Trace-Id", "trace-tenant-detail-3")
	rec := httptest.NewRecorder()
	writeErrorDetail(rec, req, http.StatusNotFound, "超级管理员权限校验失败")

	body := decodeBodyTenant(t, rec)
	if body["detail"] != "超级管理员权限校验失败" || body["error"] != "超级管理员权限校验失败" || body["message"] != "超级管理员权限校验失败" {
		t.Errorf("detail/error/message = %v/%v/%v", body["detail"], body["error"], body["message"])
	}
	if body["trace_id"] != "trace-tenant-detail-3" {
		t.Errorf("trace_id = %v, want trace-tenant-detail-3", body["trace_id"])
	}
}

func TestWriteErrorMapPreservesExtraKeys(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/invites", nil)
	req.Header.Set("X-Trace-Id", "trace-tenant-map-4")
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"valid": false,
		"error": "邀请链接无效",
	})
	body := decodeBodyTenant(t, rec)
	if body["trace_id"] != "trace-tenant-map-4" {
		t.Errorf("trace_id = %v, want trace-tenant-map-4", body["trace_id"])
	}
	if body["valid"] != false {
		t.Errorf("valid = %v, want false", body["valid"])
	}
	if body["message"] != "邀请链接无效" {
		t.Errorf("message = %v, want 邀请链接无效", body["message"])
	}
}

func TestWriteErrorMapRespectsExistingStatusAndMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/groups", nil)
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"status":  "fail",
		"message": "keep me",
		"detail":  "do not overwrite",
	})
	body := decodeBodyTenant(t, rec)
	if body["status"] != "fail" {
		t.Errorf("status = %v, want existing fail", body["status"])
	}
	if body["message"] != "keep me" {
		t.Errorf("message = %v, want existing keep me", body["message"])
	}
}
