package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
)

func TestEmitStructuredError_WritesJSONLevelError(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	emitStructuredError("Config error", errors.New(`field "frontend.x.y": provider "frontend" not in runAll config`))
	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()

	var payload map[string]string
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &payload); err != nil {
		t.Fatalf("expected JSON log line, got %q: %v", buf.String(), err)
	}
	if payload["level"] != "error" {
		t.Fatalf("level=%q, want error", payload["level"])
	}
	if payload["service"] != "value-stream" {
		t.Fatalf("service=%q, want value-stream", payload["service"])
	}
	if payload["msg"] == "" || payload["ts"] == "" {
		t.Fatalf("missing required fields: %+v", payload)
	}
	tid, ok := payload["trace_id"]
	if !ok || tid == "" {
		t.Fatalf("trace_id must be non-empty for startup errors, got %+v", payload)
	}
	if len(tid) < 8 {
		t.Fatalf("trace_id too short: %q", tid)
	}
}
