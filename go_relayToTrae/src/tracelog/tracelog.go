package tracelog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

var serviceName = "go-relay"

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

func newTraceID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "relay-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
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

// ForwardChildLine writes a child process log line for runAll/Promtail collection.
// Structured JSON lines that already include a service field are passed through unchanged.
// Plain text lines are wrapped as JSON with defaultService.
func ForwardChildLine(line, defaultService string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	if forwardStructuredLine(line) {
		return
	}
	svc := strings.TrimSpace(defaultService)
	if svc == "" {
		svc = "onlineServiceJS"
	}
	emitStructuredLine(svc, "info", line, nil)
}

func forwardStructuredLine(line string) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return false
	}
	if _, ok := fields["service"]; !ok {
		return false
	}
	_, _ = os.Stdout.Write([]byte(line))
	_, _ = os.Stdout.Write([]byte("\n"))
	return true
}

func emitStructuredLine(service, level, msg string, extra map[string]string) {
	payload := map[string]string{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   level,
		"service": service,
		"msg":     msg,
	}
	if tid := normalizeTraceID(os.Getenv("TRACE_ID")); tid != "" {
		payload["trace_id"] = tid
		payload["otel_trace_id"] = OtelTraceIDHex(tid)
	}
	for k, v := range extra {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			payload[k] = v
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(payload)
}

// Emit writes a JSON log line with optional trace id from env TRACE_ID.
func Emit(level, msg, component string, fields map[string]string) {
	EmitWithTrace(normalizeTraceID(os.Getenv("TRACE_ID")), level, msg, component, fields)
}

// EmitWithTrace writes a JSON log line with an explicit trace id when valid.
func EmitWithTrace(tid, level, msg, component string, fields map[string]string) {
	payload := map[string]string{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   level,
		"service": serviceName,
		"msg":     msg,
	}
	if component != "" {
		payload["component"] = component
	}
	if normalized := normalizeTraceID(tid); normalized != "" {
		payload["trace_id"] = normalized
		payload["otel_trace_id"] = OtelTraceIDHex(normalized)
	}
	for k, v := range fields {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			payload[k] = v
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(payload)
}

// WithTraceEnv returns env with TRACE_ID set when tid is valid.
func WithTraceEnv(env map[string]string, tid string) map[string]string {
	out := make(map[string]string, len(env)+1)
	for k, v := range env {
		out[k] = v
	}
	if normalized := normalizeTraceID(tid); normalized != "" {
		out["TRACE_ID"] = normalized
	}
	return out
}
