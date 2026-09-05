package tracelog

import (
	"context"
	"encoding/json"
	"strings"
)

// EnsureTraceCorrelationInMap writes trace_id, span_id, parent_span_id and derived otel_trace_id when missing.
func EnsureTraceCorrelationInMap(m map[string]interface{}, traceID string) {
	if m == nil {
		return
	}
	tid := normalizeTraceID(traceID)
	if tid == "" {
		if raw, ok := m["trace_id"].(string); ok {
			tid = normalizeTraceID(raw)
		}
	}
	if tid == "" {
		return
	}
	if raw, ok := m["trace_id"].(string); !ok || normalizeTraceID(raw) == "" {
		m["trace_id"] = tid
	}
	if raw, ok := m["otel_trace_id"].(string); !ok || strings.TrimSpace(raw) == "" {
		m["otel_trace_id"] = OtelTraceIDHex(tid)
	}
}

// EnsureSpanCorrelationInMap writes span_id and parent_span_id from ctx when missing.
func EnsureSpanCorrelationInMap(ctx context.Context, m map[string]interface{}) {
	if m == nil {
		return
	}
	c := CorrelationFromContext(ctx)
	if c.SpanID != "" {
		if raw, ok := m["span_id"].(string); !ok || normalizeSpanID(raw) == "" {
			m["span_id"] = c.SpanID
		}
	}
	if c.ParentSpanID != "" {
		if raw, ok := m["parent_span_id"].(string); !ok || normalizeSpanID(raw) == "" {
			m["parent_span_id"] = c.ParentSpanID
		}
	}
}

// EnsureTraceInData injects trace_id into event payload maps when missing.
func EnsureTraceInData(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	if raw, ok := data["trace_id"].(string); ok {
		if tid := normalizeTraceID(raw); tid != "" {
			EnsureTraceCorrelationInMap(data, tid)
			EnsureSpanCorrelationInMap(ctx, data)
			return data
		}
	}
	if tid := TraceIDFromContext(ctx); tid != "" {
		EnsureTraceCorrelationInMap(data, tid)
		EnsureSpanCorrelationInMap(ctx, data)
		return data
	}
	// Do not invent background trace ids when publishing from an HTTP request path;
	// callers that truly lack correlation should pass explicit trace_id in event data.
	return data
}

// TraceIDFromEnvelopeData reads trace_id / traceId from event JSON data.
func TraceIDFromEnvelopeData(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	for _, key := range []string{"trace_id", "traceId"} {
		raw, ok := m[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if tid := normalizeTraceID(s); tid != "" {
				return tid
			}
		}
	}
	return ""
}

func envelopeStringField(data json.RawMessage, keys ...string) string {
	if len(data) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// CorrelationFromEnvelopeData builds consumer span correlation from event payload.
// Publisher span_id becomes parent_span_id for the consumer hop.
func CorrelationFromEnvelopeData(data json.RawMessage) Correlation {
	tid := TraceIDFromEnvelopeData(data)
	if tid == "" {
		return Correlation{}
	}
	parent := normalizeSpanID(envelopeStringField(data, "span_id"))
	if parent == "" {
		parent = normalizeSpanID(envelopeStringField(data, "parent_span_id"))
	}
	return Correlation{
		TraceID:      tid,
		SpanID:       newSpanID(),
		ParentSpanID: parent,
	}
}
