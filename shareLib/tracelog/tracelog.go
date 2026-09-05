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

var (
	safeTraceID            = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,256}$`)
	hex32                  = regexp.MustCompile(`(?i)^[0-9a-f]{32}$`)
	serviceName            = "task-service"
	generateTraceIfMissing = true
	baseJSONHandler        slog.Handler
)

// Init configures JSON slog output for runAll/Promtail collection.
func Init(service string) {
	generateTraceIfMissing = true
	initService(service)
}

// InitConsumer is for Kafka/event consumers that should not invent trace ids on HTTP probes.
func InitConsumer(service string) {
	generateTraceIfMissing = false
	initService(service)
}

func initService(service string) {
	if strings.TrimSpace(service) != "" {
		serviceName = strings.TrimSpace(service)
	}
	// Business processes must not inherit shell HTTP(S)_PROXY (dev-only accel).
	disableDefaultEnvProxy()
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "ts"
			case slog.MessageKey:
				a.Key = "msg"
			case slog.LevelKey:
				a.Key = "level"
				a.Value = slog.StringValue(NormalizeLevel(a.Value.String()))
			}
			return a
		},
	})
	baseJSONHandler = impersonationHandler{Handler: h}
	if strings.TrimSpace(aidevServiceID) == "" {
		SetDaydaymoneyMeta(serviceName, nil)
	} else {
		applyAidevToDefaultLogger()
	}
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

// ContextWithTraceID stores trace id on ctx for downstream calls and events.
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
		return "trace-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

// NewTraceID returns a new random trace id.
func NewTraceID() string {
	return newTraceID()
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

// ResolveTraceID prefers explicit header/value, then context, then empty.
func ResolveTraceID(ctx context.Context, header string) string {
	if tid := normalizeTraceID(header); tid != "" {
		return tid
	}
	return TraceIDFromContext(ctx)
}

// Middleware assigns or propagates trace/span headers and logs each HTTP request as JSON.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if RejectTraceIdOnlyHTTP(w, r) {
			return
		}
		corr := ResolveInboundCorrelation(r, generateTraceIfMissing)
		ctx := r.Context()
		if corr.TraceID != "" || corr.SpanID != "" {
			ctx = ContextWithCorrelation(ctx, corr)
		}
		ctx = ContextWithImpersonation(ctx, ImpersonationFromRequest(r))
		if corr.TraceID != "" {
			w.Header().Set(Header, corr.TraceID)
		}
		if corr.SpanID != "" {
			w.Header().Set(SpanHeader, corr.SpanID)
		}
		if tp := FormatTraceParent(corr); tp != "" {
			w.Header().Set(TraceParentHeader, tp)
		}
		var endSpan func()
		if corr.TraceID != "" {
			ctx, endSpan = bindOtelSpan(ctx, r, corr)
		}
		if endSpan != nil {
			defer endSpan()
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r.WithContext(ctx))
		emitHTTPLog(ctx, r.Method, r.URL.Path, rec.status, start)
	})
}

func emitHTTPLog(ctx context.Context, method, path string, status int, start time.Time) {
	args := []any{
		"service", serviceName,
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", time.Since(start).Milliseconds(),
	}
	args = AppendCorrelationLogArgs(args, ctx)
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
