package tracelog

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	otelActive = false
	tracer     trace.Tracer
	hex32      = regexp.MustCompile(`(?i)^[0-9a-f]{32}$`)
)

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

// InitOtel configures OTLP gRPC export when OTEL_EXPORTER_OTLP_ENDPOINT is set.
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

func startOtelSpan(ctx context.Context, r *http.Request, externalTraceID string) (context.Context, func()) {
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
	spanName := r.Method + " " + r.URL.Path
	ctx, span := tracer.Start(parentCtx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
			attribute.String("trace_id", externalTraceID),
		),
	)
	return ctx, func() { span.End() }
}
