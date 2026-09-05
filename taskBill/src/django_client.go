package main

import (
	"context"
	"strings"

	"tracelog"
)

func billingCorrelationCtx(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	corr := tracelog.CorrelationFromContext(ctx)
	if corr.TraceID == "" {
		corr.TraceID = strings.TrimSpace(traceID)
	}
	if corr.TraceID == "" {
		return ctx
	}
	if corr.SpanID == "" {
		corr.SpanID = tracelog.NewSpanID()
	}
	return tracelog.ContextWithCorrelation(ctx, corr)
}

// djangoEmitBillingEvent now publishes directly to Kafka (no Django bridge).
// Kept for backward compatibility with existing callers.
func djangoEmitBillingEvent(ctx context.Context, outboxID int64, payload map[string]interface{}) {
	publishBillingTransactionCreated(ctx, outboxID, payload)
}

// djangoEmitEvent now publishes directly to Kafka (no Django bridge).
// Kept for backward compatibility with existing callers.
func djangoEmitEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
	publishBillingEvent(ctx, eventType, payload)
}

// emitPaypalLifecycleAsync now publishes directly to Kafka (no Django bridge).
func emitPaypalLifecycleAsync(ctx context.Context, eventType string, payload map[string]interface{}) {
	publishPaypalLifecycleAsync(ctx, eventType, payload)
}

// djangoEnrichTransactions enriches billing rows with project/workspace names.
// Calls taskProjectService + taskTaskService directly (no Django bridge).
func djangoEnrichTransactions(ctx context.Context, tenantID int64, rows []map[string]interface{}) []map[string]interface{} {
	return enrichBillingRows(ctx, tenantID, rows)
}
