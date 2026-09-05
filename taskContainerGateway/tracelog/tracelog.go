package tracelog

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
var hex32 = regexp.MustCompile(`(?i)^[0-9a-f]{32}$`)

var serviceName = "task-container-gateway"

// Init configures JSON slog output for runAll/Promtail collection.
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

func newTraceID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "tcgw-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

// OtelTraceIDHex maps X-Trace-Id to a Tempo-compatible 32-char hex trace id.
func OtelTraceIDHex(external string) string {
	raw := strings.TrimSpace(external)
	compact := strings.ReplaceAll(raw, "-", "")
	if hex32.MatchString(compact) {
		return strings.ToLower(compact)
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:16])
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
		tid := normalizeTraceID(r.Header.Get(Header))
		if tid == "" {
			tid = newTraceID()
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, tid)
		w.Header().Set(Header, tid)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
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

// LogForwardStage emits a structured forward_stage line for Loki trace correlation.
func LogForwardStage(ctx context.Context, stage string, attrs map[string]any) {
	tid := TraceIDFromContext(ctx)
	args := []any{
		"service", serviceName,
		"trace_id", tid,
		"otel_trace_id", OtelTraceIDHex(tid),
		"forward_stage", stage,
	}
	for k, v := range attrs {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		args = append(args, k, v)
	}
	slog.InfoContext(ctx, "forward_stage", args...)
}
