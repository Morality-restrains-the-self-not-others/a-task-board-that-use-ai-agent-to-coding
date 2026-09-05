package tracelog

import (
	"bytes"
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

func TestForwardChildLine_PassthroughStructuredJSON(t *testing.T) {
	line := `{"ts":"2026-05-28T10:00:00Z","level":"info","service":"onlineServiceJS","trace_id":"trace-test12345678","msg":"http_request"}`
	out := captureStdout(t, func() {
		ForwardChildLine(line, "onlineServiceJS")
	})
	if !strings.Contains(out, line) {
		t.Fatalf("expected passthrough line, got %q", out)
	}
	var wrapped map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &wrapped); err == nil && wrapped["service"] == "go-relay" {
		t.Fatalf("structured line was re-wrapped: %q", out)
	}
}

func TestForwardChildLine_WrapsPlainTextWithService(t *testing.T) {
	out := captureStdout(t, func() {
		ForwardChildLine("[onlineServiceJS] server listening", "onlineServiceJS")
	})
	var payload map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["service"] != "onlineServiceJS" {
		t.Fatalf("service = %q", payload["service"])
	}
	if !strings.Contains(payload["msg"], "server listening") {
		t.Fatalf("msg = %q", payload["msg"])
	}
}

func TestForwardChildLine_WrapsMultilineBlock(t *testing.T) {
	block := "[onlineServiceJS] reachability 失败: Error: HTTP 500\n    at postJson (file:///app/src/saasTaskCloud.mjs:154:23)\n    at async registerReachability (file:///app/src/reachability.mjs:265:3)"
	out := captureStdout(t, func() {
		ForwardChildLine(block, "onlineServiceJS")
	})
	var payload map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["service"] != "onlineServiceJS" {
		t.Fatalf("service = %q", payload["service"])
	}
	if !strings.Contains(payload["msg"], "at postJson") || !strings.Contains(payload["msg"], "reachability") {
		t.Fatalf("msg should contain full stack: %q", payload["msg"])
	}
	if strings.Count(payload["msg"], "\n") < 2 {
		t.Fatalf("msg should preserve newlines: %q", payload["msg"])
	}
}

func TestMiddleware_PropagatesTraceHeader(t *testing.T) {
	Init("go-relay-test")
	var got string
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = TraceIDFromContext(r.Context())
		w.WriteHeader(204)
	}))
	req := httptest.NewRequest("GET", "/health", nil)
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

func TestEmitWithTraceIncludesTraceFields(t *testing.T) {
	out := captureStdout(t, func() {
		EmitWithTrace("trace-test12345678", "info", "[relayToTrae] start", "relay", nil)
	})
	var payload map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["trace_id"] != "trace-test12345678" {
		t.Fatalf("trace_id = %q", payload["trace_id"])
	}
	if payload["otel_trace_id"] == "" {
		t.Fatalf("expected otel_trace_id")
	}
}

func TestWithTraceEnv(t *testing.T) {
	out := WithTraceEnv(map[string]string{"A": "1"}, "trace-xyz12345678")
	if out["TRACE_ID"] != "trace-xyz12345678" {
		t.Fatalf("TRACE_ID = %q", out["TRACE_ID"])
	}
}
