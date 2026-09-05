//go:build !otel_enabled

package tracelog

import (
	"context"
	"net/http"
)

func InitOtel(ctx context.Context, service string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	return noop, nil
}

func bindOtelSpan(ctx context.Context, r *http.Request, corr Correlation) (context.Context, func()) {
	if corr.TraceID == "" {
		return ctx, func() {}
	}
	return ContextWithCorrelation(ctx, corr), func() {}
}

// StartSpan records trace/span correlation on ctx when OTEL export is disabled.
func StartSpan(ctx context.Context, name, externalTraceID string, attrs ...Attr) (context.Context, func()) {
	if externalTraceID == "" {
		return ctx, func() {}
	}
	corr := CorrelationFromContext(ctx)
	if corr.TraceID == "" {
		corr.TraceID = externalTraceID
	}
	if corr.SpanID == "" {
		corr.SpanID = newSpanID()
	}
	return ContextWithCorrelation(ctx, corr), func() {}
}
