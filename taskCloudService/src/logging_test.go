package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"tracelog"
)

func TestBusinessLogUsesSlogJSONFormat(t *testing.T) {
	serviceName = "task-cloud-service-test"
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })

	traceID := "web-test-log-unify-001"
	logInfo("workbench-link: generated url task_id=task1", traceID)

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected log output")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json: %v line=%s", err, line)
	}
	if payload["time"] == nil && payload["ts"] != nil {
		t.Fatalf("expected slog time field, got ts-only payload: %v", payload)
	}
	if payload["msg"] != "workbench-link: generated url task_id=task1" {
		t.Fatalf("msg=%v", payload["msg"])
	}
	if payload["service"] != "task-cloud-service-test" {
		t.Fatalf("service=%v", payload["service"])
	}
	if payload["trace_id"] != traceID {
		t.Fatalf("trace_id=%v", payload["trace_id"])
	}
	wantOtel := tracelog.OtelTraceIDHex(traceID)
	if payload["otel_trace_id"] != wantOtel {
		t.Fatalf("otel_trace_id=%v want %s", payload["otel_trace_id"], wantOtel)
	}
}
