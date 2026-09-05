package main

import (
	"net/http"
	"strings"
)

// statusRecorder wraps http.ResponseWriter to capture the status code written
// by a handler. It preserves http.Flusher so SSE/streaming endpoints keep
// working under the trace-logging middleware.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying writer when it supports http.Flusher
// (SSE progress streams rely on it).
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// uiRequestTraceLogging wraps the Status-UI mux so every request carrying an
// X-Trace-Id produces a structured log entry with that trace_id. The entry
// flows through the same tee→Promtail→Loki pipeline as service logs, so an
// error banner's data-traceId becomes queryable end-to-end (OPT-20260820-009).
//
// To avoid flooding Loki with progress-poll noise, only errors (status >= 400)
// and mutating requests (POST/PUT/PATCH/DELETE) are logged; plain GET/HEAD
// success and requests without a trace id are skipped.
func uiRequestTraceLogging(runner *Runner) func(http.Handler) http.Handler {
	if runner == nil {
		return func(h http.Handler) http.Handler { return h }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			tid := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
			if tid == "" {
				return
			}
			method := r.Method
			status := rec.status
			isMutating := method != http.MethodGet && method != http.MethodHead
			if status < 400 && !isMutating {
				return
			}
			level := "info"
			if status >= 400 {
				level = "error"
			}
			runner.appendStructuredLog("runall", level, "ui_request", map[string]interface{}{
				"trace_id": tid,
				"method":   method,
				"path":     r.URL.Path,
				"status":   status,
			})
		})
	}
}
