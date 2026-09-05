package cloudcommon

import (
	"context"
	"testing"

	"tracelog"
)

func TestEnsureTraceInStatusDataAddsOtelTraceID(t *testing.T) {
	ctx := tracelog.ContextWithTraceID(context.Background(), "web-trace-test12345678")
	out := ensureTraceInStatusData(ctx, map[string]interface{}{"status": "processing"})
	if out["trace_id"] != "web-trace-test12345678" {
		t.Fatalf("trace_id=%v", out["trace_id"])
	}
	otel, ok := out["otel_trace_id"].(string)
	if !ok || otel != tracelog.OtelTraceIDHex("web-trace-test12345678") {
		t.Fatalf("otel_trace_id=%v", out["otel_trace_id"])
	}
}

func TestSSEEventPayloadIncludesCorrelationFields(t *testing.T) {
	status := map[string]interface{}{
		"trace_id":       "web-trace-test12345678",
		"otel_trace_id":  tracelog.OtelTraceIDHex("web-trace-test12345678"),
		"status":         "error",
	}
	payload := sseEventPayload("task-1", status)
	if payload["trace_id"] != "web-trace-test12345678" {
		t.Fatalf("payload trace_id=%v", payload["trace_id"])
	}
	if payload["otel_trace_id"] != status["otel_trace_id"] {
		t.Fatalf("payload otel_trace_id=%v", payload["otel_trace_id"])
	}
}

func TestPublishSSENilPublisherReturnsError(t *testing.T) {
	err := PublishSSE(context.Background(), nil, "task-1", map[string]interface{}{"status": "ok"})
	if err == nil {
		t.Fatal("expected error for nil publisher")
	}
}
