package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"tracelog"
)

// OPT-20260901-025: logWarn must emit JSON with the request ctx trace_id so
// Loki {job="ai-provider"} | json | trace_id="..." reaches WARN lines, not just
// http_request.
func TestLogWarnEmitsTraceIDFromContext(t *testing.T) {
	const want = "opt-test-trace"
	ctx := tracelog.ContextWithTraceID(context.Background(), want)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	logWarn(ctx, "event=TestTraceWarn err=%s", "boom")
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	var payload map[string]string
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON log line, got %q: %v", buf.String(), err)
	}
	if payload["trace_id"] != want {
		t.Fatalf("trace_id=%q, want %q", payload["trace_id"], want)
	}
	if payload["level"] != "warn" {
		t.Fatalf("level=%q, want warn", payload["level"])
	}
	if payload["msg"] == "" {
		t.Fatal("msg is empty")
	}
}
