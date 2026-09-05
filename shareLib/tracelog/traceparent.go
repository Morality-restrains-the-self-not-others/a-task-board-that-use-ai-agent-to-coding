package tracelog

import (
	"regexp"
	"strings"
)

const TraceParentHeader = "traceparent"

var traceParentRE = regexp.MustCompile(`(?i)^00-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`)

// ParseTraceParent parses W3C traceparent (version 00).
func ParseTraceParent(raw string) (traceID32, parentSpanID string, ok bool) {
	s := strings.TrimSpace(raw)
	m := traceParentRE.FindStringSubmatch(s)
	if m == nil {
		return "", "", false
	}
	parent := strings.ToLower(m[2])
	if parent == "0000000000000000" {
		parent = ""
	}
	return strings.ToLower(m[1]), parent, true
}

// FormatTraceParent builds a W3C traceparent value from correlation (sampled).
func FormatTraceParent(c Correlation) string {
	if c.TraceID == "" {
		return ""
	}
	traceHex := OtelTraceIDHex(c.TraceID)
	parent := c.SpanID
	if parent == "" {
		parent = "0000000000000000"
	}
	return "00-" + traceHex + "-" + parent + "-01"
}

// CorrelationFromTraceParentHeader reads traceparent when custom headers are absent.
func CorrelationFromTraceParentHeader(raw string) (traceID string, parentSpanID string) {
	traceHex, parent, ok := ParseTraceParent(raw)
	if !ok {
		return "", ""
	}
	return "tp-" + traceHex, parent
}
