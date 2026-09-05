//go:build otel_enabled

package tracelog

import (
	"context"
	"crypto/rand"
	"net/http"
	"os"
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
)

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

func bindOtelSpan(ctx context.Context, r *http.Request, corr Correlation) (context.Context, func()) {
	if corr.TraceID == "" {
		return ctx, func() {}
	}
	ctx = ContextWithCorrelation(ctx, corr)
	if !otelActive || tracer == nil {
		return ctx, func() {}
	}
	return StartSpan(ctx, r.Method+" "+r.URL.Path, corr.TraceID,
		String("http.method", r.Method),
		String("http.target", r.URL.Path),
	)
}

// StartSpan creates an OTEL span aligned with external X-Trace-Id when export is enabled.
func StartSpan(ctx context.Context, name, externalTraceID string, attrs ...Attr) (context.Context, func()) {
	if externalTraceID == "" {
		return ctx, func() {}
	}
	corr := CorrelationFromContext(ctx)
	if corr.TraceID == "" {
		corr.TraceID = externalTraceID
	}
	ctx = ContextWithCorrelation(ctx, corr)
	if !otelActive || tracer == nil {
		return ctx, func() {}
	}
	traceID, err := trace.TraceIDFromHex(OtelTraceIDHex(corr.TraceID))
	if err != nil {
		return ctx, func() {}
	}
	var parentSpanID trace.SpanID
	if corr.ParentSpanID != "" {
		if sid, err := trace.SpanIDFromHex(corr.ParentSpanID); err == nil {
			parentSpanID = sid
		}
	}
	var spanID trace.SpanID
	if corr.SpanID != "" {
		if sid, err := trace.SpanIDFromHex(corr.SpanID); err == nil {
			spanID = sid
		}
	}
	if !spanID.IsValid() {
		if _, err := rand.Read(spanID[:]); err != nil {
			return ctx, func() {}
		}
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     parentSpanID.IsValid(),
	})
	parentCtx := trace.ContextWithSpanContext(ctx, sc)
	all := make([]attribute.KeyValue, 0, len(attrs)+1)
	all = append(all, attribute.String("trace_id", corr.TraceID))
	for _, a := range attrs {
		all = append(all, attribute.String(a.Key, a.Value))
	}
	ctx, span := tracer.Start(parentCtx, name, trace.WithAttributes(all...))
	return ctx, func() { span.End() }
}
