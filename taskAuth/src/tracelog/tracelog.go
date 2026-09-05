package tracelog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const Header = "X-Trace-Id"

type ctxKey struct{}

var safeTraceID = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,256}$`)

var serviceName = "task-auth"

// Init configures JSON slog output for centralized log collection.
func Init(service string) {
	if strings.TrimSpace(service) != "" {
		serviceName = strings.TrimSpace(service)
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
}

func normalizeTraceID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if len(s) > 256 {
		s = s[:256]
	}
	if !safeTraceID.MatchString(s) {
		return ""
	}
	return s
}

// NormalizeTraceID validates and returns a safe trace id, or empty string.
func NormalizeTraceID(raw string) string {
	return normalizeTraceID(raw)
}

// ContextWithTraceID stores trace id on ctx for downstream domain event publish.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	tid := normalizeTraceID(traceID)
	if tid == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, tid)
}

func newTraceID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "task-auth-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

// NewTraceID returns a new random trace id suitable for domain event payloads.
func NewTraceID() string {
	return newTraceID()
}

// TraceIDFromContext returns the trace id stored on ctx, if any.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

// Middleware assigns or propagates X-Trace-Id and logs each HTTP request as JSON.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		tid := normalizeTraceID(r.Header.Get(Header))
		if tid == "" {
			tid = newTraceID()
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, tid)
		w.Header().Set(Header, tid)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		ctx, endSpan := startOtelSpan(ctx, r, tid)
		defer endSpan()
		next.ServeHTTP(rec, r.WithContext(ctx))
		slog.InfoContext(ctx, "http_request",
			"service", serviceName,
			"trace_id", tid,
			"otel_trace_id", OtelTraceIDHex(tid),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}
