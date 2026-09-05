package domainevents

import (
	"context"

	"tracelog"
)

// EnsureTraceInData injects trace_id into domain event data per monorepo contract.
func EnsureTraceInData(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	if raw, ok := data["trace_id"].(string); ok {
		if tid := tracelog.NormalizeTraceID(raw); tid != "" {
			return data
		}
	}
	if tid := tracelog.TraceIDFromContext(ctx); tid != "" {
		data["trace_id"] = tid
		return data
	}
	data["trace_id"] = "bg-" + tracelog.NewTraceID()
	return data
}
