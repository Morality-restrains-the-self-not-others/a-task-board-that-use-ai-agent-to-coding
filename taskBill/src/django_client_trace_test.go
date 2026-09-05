package main

import (
	"context"
	"testing"

	"tracelog"
)

func TestBillingCorrelationCtx_GeneratesSpanWhenTraceOnly(t *testing.T) {
	ctx := billingCorrelationCtx(context.Background(), "legacy-trace-only12345")
	corr := tracelog.CorrelationFromContext(ctx)
	if corr.TraceID != "legacy-trace-only12345" {
		t.Fatalf("trace = %q", corr.TraceID)
	}
	if corr.SpanID == "" {
		t.Fatal("expected generated span id")
	}
}
