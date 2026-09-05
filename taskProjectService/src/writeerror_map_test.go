package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260807-059 回归测试：writeErrorMap 在保留调用方附加字段（git_repos /
// oauth_bound 等前端表单校验/流程状态键）的同时注入 trace_id，使 body 兜底契约
// 对复合错误体同样成立（resolveRequestTraceId 可读 body.trace_id）。
func TestWriteErrorMap_AddsTraceIDAndKeepsExtraKeys(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("X-Trace-Id", "tid-wem-1")
	rec := httptest.NewRecorder()

	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"error":       "别名重复",
		"git_repos":   "别名重复",
		"oauth_bound": false,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["trace_id"] != "tid-wem-1" {
		t.Errorf("expected body trace_id=tid-wem-1, got %v", body["trace_id"])
	}
	if body["git_repos"] != "别名重复" {
		t.Errorf("expected git_repos preserved, got %v", body["git_repos"])
	}
	if body["oauth_bound"] != false {
		t.Errorf("expected oauth_bound=false preserved, got %v", body["oauth_bound"])
	}
	if body["error"] != "别名重复" {
		t.Errorf("expected error=别名重复, got %v", body["error"])
	}
	// writeErrorMap 兜底补 status/message，前端 message 优先读取方不破
	if body["status"] != "error" {
		t.Errorf("expected status=error, got %v", body["status"])
	}
	if body["message"] != "别名重复" {
		t.Errorf("expected message=别名重复, got %v", body["message"])
	}
}

func TestWriteErrorMap_NoTraceHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	rec := httptest.NewRecorder()

	writeErrorMap(rec, req, http.StatusNotFound, map[string]interface{}{
		"error": "not found",
	})

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, has := body["trace_id"]; has {
		t.Errorf("expected no trace_id without X-Trace-Id header, got %v", body["trace_id"])
	}
	if body["error"] != "not found" {
		t.Errorf("expected error=not found, got %v", body["error"])
	}
	if body["message"] != "not found" {
		t.Errorf("expected message fallback=not found, got %v", body["message"])
	}
}

func TestWriteErrorMap_PreservesExistingMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("X-Trace-Id", "tid-wem-2")
	rec := httptest.NewRecorder()

	// 调用方显式给出 message 时不覆盖（writeError 双键语义兼容）
	writeErrorMap(rec, req, http.StatusInternalServerError, map[string]interface{}{
		"error":   "inner",
		"message": "outer 语义",
	})

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["message"] != "outer 语义" {
		t.Errorf("expected message preserved as outer 语义, got %v", body["message"])
	}
	if body["trace_id"] != "tid-wem-2" {
		t.Errorf("expected trace_id=tid-wem-2, got %v", body["trace_id"])
	}
}
