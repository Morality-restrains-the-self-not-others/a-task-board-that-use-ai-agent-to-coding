package tracelog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
)

const (
	SpanHeader       = "X-Span-Id"
	ParentSpanHeader = "X-Parent-Span-Id"
)

var hex16 = regexp.MustCompile(`(?i)^[0-9a-f]{16}$`)

type spanCtxKey struct{}

// Correlation holds distributed trace identifiers for a single request span.
type Correlation struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
}

func normalizeSpanID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if len(s) > 32 {
		s = s[:32]
	}
	if !hex16.MatchString(s) {
		return ""
	}
	return strings.ToLower(s)
}

func newSpanID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// NormalizeSpanID validates a 16-char hex span id.
func NormalizeSpanID(raw string) string {
	return normalizeSpanID(raw)
}

// NewSpanID returns a new random 16-char hex span id.
func NewSpanID() string {
	return newSpanID()
}

// ContextWithCorrelation stores trace/span correlation on ctx.
func ContextWithCorrelation(ctx context.Context, c Correlation) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	c.TraceID = normalizeTraceID(c.TraceID)
	c.SpanID = normalizeSpanID(c.SpanID)
	c.ParentSpanID = normalizeSpanID(c.ParentSpanID)
	if c.TraceID == "" && c.SpanID == "" && c.ParentSpanID == "" {
		return ctx
	}
	if c.TraceID != "" {
		ctx = context.WithValue(ctx, ctxKey{}, c.TraceID)
	}
	return context.WithValue(ctx, spanCtxKey{}, c)
}

// CorrelationFromContext reads trace/span correlation from ctx.
func CorrelationFromContext(ctx context.Context) Correlation {
	if ctx == nil {
		return Correlation{}
	}
	c, _ := ctx.Value(spanCtxKey{}).(Correlation)
	if c.TraceID == "" {
		c.TraceID = TraceIDFromContext(ctx)
	}
	return c
}

// ResolveInboundCorrelation builds correlation from HTTP headers, generating ids when allowed.
func ResolveInboundCorrelation(r *http.Request, generateIfMissing bool) Correlation {
	tid := normalizeTraceID(r.Header.Get(Header))
	parent := normalizeSpanID(r.Header.Get(ParentSpanHeader))
	if tid == "" {
		if tpTrace, tpParent, ok := ParseTraceParent(r.Header.Get(TraceParentHeader)); ok {
			tid = "tp-" + tpTrace
			if parent == "" {
				parent = tpParent
			}
		}
	}
	if tid == "" && generateIfMissing {
		tid = newTraceID()
	}
	span := newSpanID()
	return Correlation{
		TraceID:      tid,
		SpanID:       span,
		ParentSpanID: parent,
	}
}

// OutboundHeaders returns headers for downstream HTTP calls using ctx correlation.
// The current span id becomes the parent span id for the callee.
func OutboundHeaders(ctx context.Context) map[string]string {
	c := CorrelationFromContext(ctx)
	out := make(map[string]string)
	if c.TraceID != "" {
		out[Header] = c.TraceID
	}
	if c.SpanID != "" {
		out[ParentSpanHeader] = c.SpanID
	}
	if tp := FormatTraceParent(c); tp != "" {
		out[TraceParentHeader] = tp
	}
	return out
}

// ApplyOutboundHeaders sets trace propagation headers on an outbound request.
func ApplyOutboundHeaders(req *http.Request, ctx context.Context) {
	if req == nil {
		return
	}
	for k, v := range OutboundHeaders(ctx) {
		req.Header.Set(k, v)
	}
}

// AppendCorrelationLogArgs adds trace_id, span_id, parent_span_id fields to slog args.
func AppendCorrelationLogArgs(args []any, ctx context.Context) []any {
	c := CorrelationFromContext(ctx)
	if c.TraceID != "" {
		args = append(args, "trace_id", c.TraceID, "otel_trace_id", OtelTraceIDHex(c.TraceID))
	}
	if c.SpanID != "" {
		args = append(args, "span_id", c.SpanID)
	}
	if c.ParentSpanID != "" {
		args = append(args, "parent_span_id", c.ParentSpanID)
	}
	return args
}
