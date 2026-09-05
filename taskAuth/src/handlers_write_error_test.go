package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

// OPT-20260808-010: 错误响应统一注入 trace_id（X-Trace-Id 头优先，
// 其次 context trace id），并保持 FE 读取顺序契约（detail 优先）。

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body not json: %v (%s)", err, rec.Body.String())
	}
	return body
}

func TestWriteErrorInjectTraceIDFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	req.Header.Set("X-Trace-Id", "trace-from-header-1")
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusBadRequest, "boom")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-from-header-1" {
		t.Errorf("trace_id = %v, want trace-from-header-1", body["trace_id"])
	}
	if body["error"] != "boom" || body["message"] != "boom" {
		t.Errorf("error/message = %v/%v, want boom/boom", body["error"], body["message"])
	}
	if body["status"] != "error" {
		t.Errorf("status field = %v, want error", body["status"])
	}
}

func TestWriteErrorInjectTraceIDFromContext(t *testing.T) {
	ctx := tracelog.ContextWithTraceID(context.Background(), "trace-from-ctx-2")
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusInternalServerError, "db error")

	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-from-ctx-2" {
		t.Errorf("trace_id = %v, want trace-from-ctx-2", body["trace_id"])
	}
}

func TestWriteErrorNoTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	rec := httptest.NewRecorder()
	writeError(rec, req, http.StatusForbidden, "forbidden")

	body := decodeBody(t, rec)
	if _, ok := body["trace_id"]; ok {
		t.Errorf("trace_id present without any trace source: %v", body["trace_id"])
	}
}

func TestWriteErrorDetailContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	req.Header.Set("X-Trace-Id", "trace-detail-3")
	rec := httptest.NewRecorder()
	writeErrorDetail(rec, req, http.StatusNotFound, "user not found")

	body := decodeBody(t, rec)
	// FE 契约：data.detail 优先读取（useLoginVerificationCode.js 读 detail||error||message）
	if body["detail"] != "user not found" || body["error"] != "user not found" || body["message"] != "user not found" {
		t.Errorf("detail/error/message = %v/%v/%v, want uniform", body["detail"], body["error"], body["message"])
	}
	if body["trace_id"] != "trace-detail-3" {
		t.Errorf("trace_id = %v, want trace-detail-3", body["trace_id"])
	}
}

func TestWriteErrorMapPreservesExtraKeys(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	req.Header.Set("X-Trace-Id", "trace-map-4")
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"error":         "该邮箱已被注册，请直接登录",
		"user_existed":  true,
		"is_active":     false,
		"custom_number": 42,
	})

	body := decodeBody(t, rec)
	if body["trace_id"] != "trace-map-4" {
		t.Errorf("trace_id = %v, want trace-map-4", body["trace_id"])
	}
	if body["user_existed"] != true || body["custom_number"] != float64(42) {
		t.Errorf("extra keys lost: %v", body)
	}
	// status/message 自动补齐（源自 error），且不覆盖已有 status
	if body["status"] != "error" || body["message"] != "该邮箱已被注册，请直接登录" {
		t.Errorf("status/message = %v/%v, want error/该邮箱已被注册，请直接登录", body["status"], body["message"])
	}
}

func TestWriteErrorMapRespectsExistingStatusAndMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test/", nil)
	rec := httptest.NewRecorder()
	writeErrorMap(rec, req, http.StatusBadRequest, map[string]interface{}{
		"status":  "custom",
		"message": "激活链接无效",
	})
	body := decodeBody(t, rec)
	if body["status"] != "custom" || body["message"] != "激活链接无效" {
		t.Errorf("existing status/message overwritten: %v", body)
	}
}
