package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

// OPT-20260807-059 回归测试：writeErrorJSONMap 在保留调用方附加字段（path/refer 等）
// 的同时注入 trace_id，使 body 兜底契约对复合错误体同样成立（前端 resolveRequestTraceId
// 可读 body.trace_id）。
func TestWriteErrorJSONMap_AddsTraceIDAndKeepsExtraKeys(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/billing/not-found", nil)
	req = req.WithContext(tracelog.ContextWithTraceID(req.Context(), "tid-wem-1"))
	rec := httptest.NewRecorder()

	writeErrorJSONMap(rec, req, http.StatusNotFound, map[string]string{
		"detail": "not found",
		"path":   "/api/billing/not-found",
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["detail"] != "not found" {
		t.Errorf("expected detail preserved, got %v", body["detail"])
	}
	if body["path"] != "/api/billing/not-found" {
		t.Errorf("expected path preserved, got %v", body["path"])
	}
	if body["trace_id"] != "tid-wem-1" {
		t.Errorf("expected trace_id=tid-wem-1, got %q", body["trace_id"])
	}
}

func TestWriteErrorJSONMap_NoTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	rec := httptest.NewRecorder()

	writeErrorJSONMap(rec, req, http.StatusGone, map[string]string{
		"detail": "pricing_package removed",
		"refer":  "/api/internal/taskbill/resource-pricing/",
	})

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["refer"] != "/api/internal/taskbill/resource-pricing/" {
		t.Errorf("expected refer preserved, got %v", body["refer"])
	}
	if _, has := body["trace_id"]; has {
		t.Errorf("expected no trace_id without context trace, got %q", body["trace_id"])
	}
}
