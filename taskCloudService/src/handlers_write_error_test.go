package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260808-010b: 错误响应统一注入 trace_id（X-Trace-Id 头）。
// writeErrorJSON 归一化 detail 体；writeErrorMapJSON 保留复合体键。

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body not json: %v (%s)", err, rec.Body.String())
	}
	return body
}

func TestWriteErrorJSONInjectTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	req.Header.Set("X-Trace-Id", "trace-cloud-1")
	rec := httptest.NewRecorder()
	writeErrorJSON(rec, req, http.StatusBadRequest, "bad request")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-cloud-1" {
		t.Errorf("trace_id = %v, want trace-cloud-1", body["trace_id"])
	}
	if body["detail"] != "bad request" {
		t.Errorf("detail = %v, want bad request", body["detail"])
	}
}

func TestWriteErrorJSONNilRequest(t *testing.T) {
	// 无 r 作用域的 helper（如 requireBudgetDB）传 nil 不 panic、无 trace_id
	rec := httptest.NewRecorder()
	writeErrorJSON(rec, nil, http.StatusServiceUnavailable, "budget db unavailable")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := decodeBody(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present with nil request: %v", body["trace_id"])
	}
	if body["detail"] != "budget db unavailable" {
		t.Errorf("detail = %v", body["detail"])
	}
}

func TestWriteErrorMapJSONPreservesKeysAndInjectsTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	req.Header.Set("X-Trace-Id", "trace-cloud-map-2")
	rec := httptest.NewRecorder()
	writeErrorMapJSON(rec, req, http.StatusForbidden, map[string]interface{}{
		"status":    "error",
		"message":   "platform_type 与授权不匹配",
		"error_code": "PLATFORM_MISMATCH",
		"retryable":  false,
	})

	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-cloud-map-2" {
		t.Errorf("trace_id = %v, want trace-cloud-map-2", body["trace_id"])
	}
	if rec.Header().Get("X-Trace-Id") != "trace-cloud-map-2" {
		t.Errorf("X-Trace-Id header = %q, want trace-cloud-map-2", rec.Header().Get("X-Trace-Id"))
	}
	if body["error_code"] != "PLATFORM_MISMATCH" || body["retryable"] != false {
		t.Errorf("extra keys lost: %v", body)
	}
	if body["status"] != "error" || body["message"] != "platform_type 与授权不匹配" {
		t.Errorf("status/message = %v/%v", body["status"], body["message"])
	}
}

func TestWriteErrorMapJSONNilRequest(t *testing.T) {
	rec := httptest.NewRecorder()
	writeErrorMapJSON(rec, nil, http.StatusBadGateway, map[string]interface{}{
		"status": "error", "message": "container gateway proxy failed",
	})
	body := decodeBody(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present with nil request: %v", body["trace_id"])
	}
	if body["message"] != "container gateway proxy failed" {
		t.Errorf("message = %v", body["message"])
	}
}

func TestWriteErrorJSONNoTraceHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	rec := httptest.NewRecorder()
	writeErrorJSON(rec, req, http.StatusNotFound, "not found")
	body := decodeBody(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present without header: %v", body["trace_id"])
	}
}
