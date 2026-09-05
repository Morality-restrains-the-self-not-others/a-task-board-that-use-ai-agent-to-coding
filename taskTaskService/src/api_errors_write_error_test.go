package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

// OPT-20260808-010c: 错误响应统一注入 trace_id（X-Trace-Id 头优先，
// context 兜底），并保持 FE 读取顺序契约（detail 优先）。

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body not json: %v (%s)", err, rec.Body.String())
	}
	return body
}

func TestWriteErrorInjectTraceIDFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/", nil)
	req.Header.Set("X-Trace-Id", "trace-task-1")
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusForbidden, "您没有权限修改此任务")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-task-1" {
		t.Errorf("trace_id = %v, want trace-task-1", body["trace_id"])
	}
	if body["error"] != "您没有权限修改此任务" || body["message"] != "您没有权限修改此任务" {
		t.Errorf("error/message = %v/%v", body["error"], body["message"])
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

func TestWriteErrorInjectTraceIDFromContext(t *testing.T) {
	ctx := tracelog.ContextWithTraceID(context.Background(), "trace-task-ctx-2")
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusInternalServerError, "db error")
	if body := decodeBody(t, rec); body["trace_id"] != "trace-task-ctx-2" {
		t.Errorf("trace_id = %v, want trace-task-ctx-2", body["trace_id"])
	}
}

func TestWriteErrorNilRequest(t *testing.T) {
	// 无 r 作用域的内部 helper 传 nil：不注入、不 panic
	rec := httptest.NewRecorder()
	writeError(rec, nil, http.StatusBadGateway, "upstream failed")
	body := decodeBody(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present with nil request: %v", body["trace_id"])
	}
}

func TestWriteErrorDetailContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/", nil)
	req.Header.Set("X-Trace-Id", "trace-task-detail-3")
	rec := httptest.NewRecorder()
	writeErrorDetail(rec, req, http.StatusNotFound, "任务不存在")

	body := decodeBody(t, rec)
	if body["detail"] != "任务不存在" || body["error"] != "任务不存在" || body["message"] != "任务不存在" {
		t.Errorf("detail/error/message = %v/%v/%v", body["detail"], body["error"], body["message"])
	}
	if body["trace_id"] != "trace-task-detail-3" {
		t.Errorf("trace_id = %v, want trace-task-detail-3", body["trace_id"])
	}
}

func TestWriteErrorMapPreservesExtraKeys(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/", nil)
	req.Header.Set("X-Trace-Id", "trace-task-map-4")
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"error": "校验失败",
		"field": "title",
		"code":  "VALIDATION_ERROR",
		"count": 3,
	})
	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-task-map-4" {
		t.Errorf("trace_id = %v, want trace-task-map-4", body["trace_id"])
	}
	if body["field"] != "title" || body["code"] != "VALIDATION_ERROR" || body["count"] != float64(3) {
		t.Errorf("extra keys lost: %v", body)
	}
	if body["message"] != "校验失败" {
		t.Errorf("message = %v, want 校验失败", body["message"])
	}
}
