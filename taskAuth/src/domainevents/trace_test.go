package domainevents

import (
	"context"
	"testing"

	"tracelog"
)

func TestEnsureTraceInData_preservesExplicit(t *testing.T) {
	data := map[string]interface{}{"trace_id": "explicit-trace-id-12345", "x": 1}
	out := EnsureTraceInData(context.Background(), data)
	if out["trace_id"] != "explicit-trace-id-12345" {
		t.Fatalf("trace_id = %v", out["trace_id"])
	}
}

func TestEnsureTraceInData_fromContext(t *testing.T) {
	ctx := tracelog.ContextWithTraceID(context.Background(), "ctx-trace-id-abcdef12")
	out := EnsureTraceInData(ctx, map[string]interface{}{"k": "v"})
	if out["trace_id"] != "ctx-trace-id-abcdef12" {
		t.Fatalf("trace_id = %v", out["trace_id"])
	}
}

func TestEnsureTraceInData_bgFallback(t *testing.T) {
	out := EnsureTraceInData(context.Background(), nil)
	raw, ok := out["trace_id"].(string)
	if !ok || len(raw) < 10 {
		t.Fatalf("expected bg- trace_id, got %v", out["trace_id"])
	}
}
