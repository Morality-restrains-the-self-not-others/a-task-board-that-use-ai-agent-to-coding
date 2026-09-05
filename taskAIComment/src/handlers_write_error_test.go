package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260808-010d: 错误响应统一注入 trace_id（X-Trace-Id 头，
// 本服务无 tracelog context 包，仅 header 提取），保持 FE 读取顺序契约（detail 优先）。

func decodeBodyAI(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body not json: %v (%s)", err, rec.Body.String())
	}
	return body
}

func TestWriteErrorInjectTraceIDFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/w1/workspace/w2/task-detail/t3/ai-comments", nil)
	req.Header.Set("X-Trace-Id", "trace-ai-1")
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusForbidden, "forbidden")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	body := decodeBodyAI(t, rec)
	if body["trace_id"] != "trace-ai-1" {
		t.Errorf("trace_id = %v, want trace-ai-1", body["trace_id"])
	}
	if body["error"] != "forbidden" || body["message"] != "forbidden" {
		t.Errorf("error/message = %v/%v", body["error"], body["message"])
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

func TestWriteErrorNilRequest(t *testing.T) {
	// 无 r 作用域的内部 helper 传 nil：不注入、不 panic
	rec := httptest.NewRecorder()
	writeError(rec, nil, http.StatusBadGateway, "upstream failed")
	body := decodeBodyAI(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present with nil request: %v", body["trace_id"])
	}
}

func TestWriteErrorDetailContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/w1/workspace/w2/task-detail/t3/ai-comments", nil)
	req.Header.Set("X-Trace-Id", "trace-ai-detail-2")
	rec := httptest.NewRecorder()
	writeErrorDetail(rec, req, http.StatusNotFound, "任务不存在")

	body := decodeBodyAI(t, rec)
	if body["detail"] != "任务不存在" || body["error"] != "任务不存在" || body["message"] != "任务不存在" {
		t.Errorf("detail/error/message = %v/%v/%v", body["detail"], body["error"], body["message"])
	}
	if body["trace_id"] != "trace-ai-detail-2" {
		t.Errorf("trace_id = %v, want trace-ai-detail-2", body["trace_id"])
	}
}

func TestWriteErrorMapPreservesExtraKeys(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/w1/workspace/w2/task-detail/t3/ai-comments", nil)
	req.Header.Set("X-Trace-Id", "trace-ai-map-3")
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadGateway, map[string]interface{}{
		"detail": "validation service unavailable",
		"error":  "upstream 502",
		"code":   "VALIDATION_UNAVAILABLE",
	})
	body := decodeBodyAI(t, rec)
	if body["trace_id"] != "trace-ai-map-3" {
		t.Errorf("trace_id = %v, want trace-ai-map-3", body["trace_id"])
	}
	if body["code"] != "VALIDATION_UNAVAILABLE" {
		t.Errorf("extra key lost: %v", body)
	}
	if body["message"] != "upstream 502" {
		t.Errorf("message = %v, want upstream 502 (error first)", body["message"])
	}
}

func TestWriteErrorMapRespectsExistingStatusAndMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/w1/workspace/w2/task-detail/t3/ai-comments", nil)
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"status":  "fail",
		"message": "keep me",
		"detail":  "do not overwrite",
	})
	body := decodeBodyAI(t, rec)
	if body["status"] != "fail" {
		t.Errorf("status = %v, want existing fail", body["status"])
	}
	if body["message"] != "keep me" {
		t.Errorf("message = %v, want existing keep me", body["message"])
	}
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present without header: %v", body["trace_id"])
	}
}
