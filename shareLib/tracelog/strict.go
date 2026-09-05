package tracelog

import (
	"errors"
	"net/http"
)

// ErrTraceIdOnlyRejected is returned when X-Trace-Id is present without span propagation headers.
var ErrTraceIdOnlyRejected = errors.New("X-Trace-Id without X-Parent-Span-Id or traceparent is not allowed")

// IsTraceIdOnlyRequest reports legacy callers that send trace id without parent span context.
func IsTraceIdOnlyRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	tid := normalizeTraceID(r.Header.Get(Header))
	if tid == "" {
		return false
	}
	if normalizeSpanID(r.Header.Get(ParentSpanHeader)) != "" {
		return false
	}
	if _, ok := r.Header[TraceParentHeader]; ok {
		return false
	}
	if r.Header.Get(TraceParentHeader) != "" {
		return false
	}
	return true
}

// RejectTraceIdOnlyHTTP writes 400 when inbound request uses trace-id-only propagation.
func RejectTraceIdOnlyHTTP(w http.ResponseWriter, r *http.Request) bool {
	if !IsTraceIdOnlyRequest(r) {
		return false
	}
	if tid := normalizeTraceID(r.Header.Get(Header)); tid != "" {
		w.Header().Set(Header, tid)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"detail":"trace propagation incomplete: require X-Parent-Span-Id or traceparent with X-Trace-Id"}`))
	return true
}
