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

// OPT-20260901-016: EMAIL_SENT / kafka publish lines must carry the request
// trace_id so Loki {job="task-auth"} | json | trace_id="<id>" reaches them.
func TestEmitTraceCarriesRequestTraceID(t *testing.T) {
	const want = "opt-email-trace"
	ctx := tracelog.ContextWithTraceID(context.Background(), want)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	emitTrace(ctx, "info", "EMAIL_SENT: kafka published", map[string]string{"to": "a@b.com"})
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
	if payload["level"] != "info" {
		t.Fatalf("level=%q, want info", payload["level"])
	}
	if payload["msg"] != "EMAIL_SENT: kafka published" {
		t.Fatalf("msg=%q, want EMAIL_SENT: kafka published", payload["msg"])
	}
}
