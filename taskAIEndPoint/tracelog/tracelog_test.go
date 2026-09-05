package tracelog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestMiddleware_PropagatesTraceHeader(t *testing.T) {
	Init("task-container-gateway-test")
	var got string
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = TraceIDFromContext(r.Context())
		w.WriteHeader(204)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	req.Header.Set(Header, "test-trace-abc12345")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get(Header) != "test-trace-abc12345" {
		t.Fatalf("response header = %q", rec.Header().Get(Header))
	}
	if got != "test-trace-abc12345" {
		t.Fatalf("context trace = %q", got)
	}
}

func TestMiddleware_EmitsHttpRequestJSON(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-container-gateway-test")
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/1/task/2/cloud/compute/container-layer-git-commit/", nil)
		req.Header.Set(Header, "trace-test12345678")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["msg"] != "http_request" {
		t.Fatalf("msg = %v", payload["msg"])
	}
	if payload["trace_id"] != "trace-test12345678" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
	if payload["service"] != "task-container-gateway-test" {
		t.Fatalf("service = %v", payload["service"])
	}
}

func TestLogForwardStage_IncludesForwardStage(t *testing.T) {
	ctx := contextWithTrace("trace-forward123456")
	out := captureStdout(t, func() {
		Init("task-container-gateway-test")
		LogForwardStage(ctx, "django_validate", map[string]any{
			"django_status": 502,
			"duration_ms":   30001,
		})
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["msg"] != "forward_stage" {
		t.Fatalf("msg = %v", payload["msg"])
	}
	if payload["forward_stage"] != "django_validate" {
		t.Fatalf("forward_stage = %v", payload["forward_stage"])
	}
}

func contextWithTrace(tid string) context.Context {
	return context.WithValue(context.Background(), ctxKey{}, tid)
}
