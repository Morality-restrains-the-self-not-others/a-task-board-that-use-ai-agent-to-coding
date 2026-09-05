package tracelog

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const Header = "X-Trace-Id"

type ctxKey struct{}

var (
	safeTraceID = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,256}$`)
	hex32       = regexp.MustCompile(`(?i)^[0-9a-f]{32}$`)
	serviceName = "task-events"
	otelActive  = false
	tracer      trace.Tracer
)

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

func OtelTraceIDHex(external string) string {
	raw := strings.TrimSpace(external)
	compact := strings.ReplaceAll(raw, "-", "")
	if hex32.MatchString(compact) {
		return strings.ToLower(compact)
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:16])
}

func InitOtel(ctx context.Context, service string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return noop, nil
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(grpcHostPort(endpoint)),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return noop, err
	}
	svc := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	if svc == "" {
		svc = service
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
		)),
	)
	otel.SetTracerProvider(tp)
	tracer = otel.Tracer(svc + ".http")
	otelActive = true
	return tp.Shutdown, nil
}

func grpcHostPort(raw string) string {
	raw = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "http://"), "https://")
	if !strings.Contains(raw, ":") {
		return raw + ":4317"
	}
	return raw
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	tid := normalizeTraceID(traceID)
	if tid == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, tid)
}

// EnsureTraceInData injects trace_id into event payload maps (see domain-events trace design).
func EnsureTraceInData(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	if raw, ok := data["trace_id"].(string); ok {
		if tid := normalizeTraceID(raw); tid != "" {
			return data
		}
	}
	if tid := TraceIDFromContext(ctx); tid != "" {
		data["trace_id"] = tid
		return data
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		data["trace_id"] = "bg-task-events-" + time.Now().UTC().Format("20060102150405")
		return data
	}
	data["trace_id"] = "bg-" + hex.EncodeToString(b)
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

func StartSpan(ctx context.Context, name, externalTraceID string, attrs ...attribute.KeyValue) (context.Context, func()) {
	if externalTraceID == "" {
		return ctx, func() {}
	}
	ctx = ContextWithTraceID(ctx, externalTraceID)
	if !otelActive || tracer == nil {
		return ctx, func() {}
	}
	traceID, err := trace.TraceIDFromHex(OtelTraceIDHex(externalTraceID))
	if err != nil {
		return ctx, func() {}
	}
	var spanID trace.SpanID
	if _, err := rand.Read(spanID[:]); err != nil {
		return ctx, func() {}
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	parentCtx := trace.ContextWithSpanContext(ctx, sc)
	all := append([]attribute.KeyValue{
		attribute.String("trace_id", externalTraceID),
	}, attrs...)
	ctx, span := tracer.Start(parentCtx, name, trace.WithAttributes(all...))
	return ctx, func() { span.End() }
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		tid := normalizeTraceID(r.Header.Get(Header))
		ctx, end := StartSpan(r.Context(), r.Method+" "+r.URL.Path, tid,
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
		)
		defer end()
		if tid != "" {
			w.Header().Set(Header, tid)
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r.WithContext(ctx))
		emitHTTPLog(ctx, tid, r.Method, r.URL.Path, rec.status, start)
	})
}

func emitHTTPLog(ctx context.Context, tid, method, path string, status int, start time.Time) {
	tid = normalizeTraceID(tid)
	if tid == "" {
		tid = TraceIDFromContext(ctx)
	}
	args := []any{
		"service", serviceName,
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", time.Since(start).Milliseconds(),
	}
	if tid != "" {
		args = append(args, "trace_id", tid, "otel_trace_id", OtelTraceIDHex(tid))
	}
	slog.InfoContext(ctx, "http_request", args...)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func Emit(level, msg, component string, traceID string, fields map[string]string) {
	tid := normalizeTraceID(traceID)
	payload := map[string]string{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   level,
		"service": serviceName,
		"msg":     msg,
	}
	if component != "" {
		payload["component"] = component
	}
	if tid != "" {
		payload["trace_id"] = tid
		payload["otel_trace_id"] = OtelTraceIDHex(tid)
	}
	for k, v := range fields {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			payload[k] = v
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(payload)
}
