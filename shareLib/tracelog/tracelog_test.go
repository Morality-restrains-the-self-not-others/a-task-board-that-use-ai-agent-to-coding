package tracelog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestMiddleware_PropagatesTraceAndSpanHeaders(t *testing.T) {
	Init("task-service-test")
	var got Correlation
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = CorrelationFromContext(r.Context())
		w.WriteHeader(204)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	req.Header.Set(Header, "test-trace-abc12345")
	req.Header.Set(ParentSpanHeader, "a1b2c3d4e5f67890")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get(Header) != "test-trace-abc12345" {
		t.Fatalf("response trace header = %q", rec.Header().Get(Header))
	}
	if rec.Header().Get(SpanHeader) == "" {
		t.Fatalf("missing response %s", SpanHeader)
	}
	if got.TraceID != "test-trace-abc12345" {
		t.Fatalf("context trace = %q", got.TraceID)
	}
	if got.ParentSpanID != "a1b2c3d4e5f67890" {
		t.Fatalf("context parent_span = %q", got.ParentSpanID)
	}
	if got.SpanID == "" {
		t.Fatal("expected generated span id")
	}
}

func TestOutboundHeaders_UsesCurrentSpanAsParent(t *testing.T) {
	ctx := ContextWithCorrelation(context.Background(), Correlation{
		TraceID: "trace-outbound123456",
		SpanID:  "b1b2c3d4e5f67890",
	})
	h := OutboundHeaders(ctx)
	if h[Header] != "trace-outbound123456" {
		t.Fatalf("trace header = %q", h[Header])
	}
	if h[ParentSpanHeader] != "b1b2c3d4e5f67890" {
		t.Fatalf("parent span header = %q", h[ParentSpanHeader])
	}
}

func TestEnsureSpanCorrelationInMap(t *testing.T) {
	ctx := ContextWithCorrelation(context.Background(), Correlation{
		TraceID:      "trace-event12345678",
		SpanID:       "c1c2c3d4e5f67890",
		ParentSpanID: "a1b2c3d4e5f67890",
	})
	data := EnsureTraceInData(ctx, map[string]interface{}{"k": "v"})
	if data["span_id"] != "c1c2c3d4e5f67890" {
		t.Fatalf("span_id = %v", data["span_id"])
	}
	if data["parent_span_id"] != "a1b2c3d4e5f67890" {
		t.Fatalf("parent_span_id = %v", data["parent_span_id"])
	}
}

func TestParseTraceParent(t *testing.T) {
	traceHex, parent, ok := ParseTraceParent("00-" + strings.Repeat("a", 32) + "-b1b2c3d4e5f67890-01")
	if !ok {
		t.Fatal("expected parse ok")
	}
	if traceHex != strings.Repeat("a", 32) {
		t.Fatalf("traceHex = %q", traceHex)
	}
	if parent != "b1b2c3d4e5f67890" {
		t.Fatalf("parent = %q", parent)
	}
}

func TestFormatTraceParent(t *testing.T) {
	tp := FormatTraceParent(Correlation{TraceID: "trace-test12345678", SpanID: "c1c2c3d4e5f67890"})
	if !strings.HasPrefix(tp, "00-") || !strings.HasSuffix(tp, "-01") {
		t.Fatalf("traceparent = %q", tp)
	}
}

func TestMiddleware_RejectsTraceIdOnly(t *testing.T) {
	Init("task-service-test")
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	req.Header.Set(Header, "test-trace-abc12345")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rec.Code)
	}
}

func TestCorrelationFromEnvelopeData(t *testing.T) {
	data := []byte(`{"trace_id":"event-trace-abc12345","span_id":"a1b2c3d4e5f67890"}`)
	corr := CorrelationFromEnvelopeData(data)
	if corr.TraceID != "event-trace-abc12345" {
		t.Fatalf("trace = %q", corr.TraceID)
	}
	if corr.ParentSpanID != "a1b2c3d4e5f67890" {
		t.Fatalf("parent = %q", corr.ParentSpanID)
	}
	if corr.SpanID == "" {
		t.Fatal("expected consumer span id")
	}
}

func TestMiddleware_EmitsHttpRequestJSON(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-service-test")
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
		req.Header.Set(Header, "trace-test12345678")
		req.Header.Set(ParentSpanHeader, "a1b2c3d4e5f67890")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["msg"] != "http_request" {
		t.Fatalf("msg = %v", payload["msg"])
	}
	if payload["level"] != "info" {
		t.Fatalf("level = %v want info (lowercase)", payload["level"])
	}
	if _, ok := payload["ts"]; !ok {
		t.Fatalf("expected ts key, got keys=%v", payload)
	}
	if payload["trace_id"] != "trace-test12345678" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
}

func TestRejectTraceIdOnlyHTTP_EchoesTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/public/catalog/", nil)
	req.Header.Set(Header, "71c09823-3d45-4777-88a9-9f57610805a1")
	rec := httptest.NewRecorder()
	if !RejectTraceIdOnlyHTTP(rec, req) {
		t.Fatal("expected reject")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get(Header); got != "71c09823-3d45-4777-88a9-9f57610805a1" {
		t.Fatalf("echo X-Trace-Id = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "trace propagation incomplete") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestRejectTraceIdOnlyHTTP_AllowsParentSpan(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/public/catalog/", nil)
	req.Header.Set(Header, "71c09823-3d45-4777-88a9-9f57610805a1")
	req.Header.Set(ParentSpanHeader, "a1b2c3d4e5f67890")
	rec := httptest.NewRecorder()
	if RejectTraceIdOnlyHTTP(rec, req) {
		t.Fatal("expected allow")
	}
}

func TestLogForwardStage_IncludesForwardStage(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "trace-forward123456")
	out := captureStdout(t, func() {
		Init("task-service-test")
		LogForwardStage(ctx, "django_validate", map[string]any{
			"django_status": 502,
			"duration_ms":   30001,
		})
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["msg"] != "forward_stage" {
		t.Fatalf("msg = %v", payload["msg"])
	}
	if payload["forward_stage"] != "django_validate" {
		t.Fatalf("forward_stage = %v", payload["forward_stage"])
	}
}

func TestEnsureTraceInData_UsesContext(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "ctx-trace-id-abcdef12")
	data := EnsureTraceInData(ctx, map[string]interface{}{"k": "v"})
	if data["trace_id"] != "ctx-trace-id-abcdef12" {
		t.Fatalf("trace_id = %v", data["trace_id"])
	}
}

func TestEnsureTraceInData_PreservesExisting(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "ctx-trace-id-abcdef12")
	data := EnsureTraceInData(ctx, map[string]interface{}{"trace_id": "web-existing-trace123"})
	if data["trace_id"] != "web-existing-trace123" {
		t.Fatalf("trace_id = %v", data["trace_id"])
	}
	if data["otel_trace_id"] != OtelTraceIDHex("web-existing-trace123") {
		t.Fatalf("otel_trace_id = %v", data["otel_trace_id"])
	}
}

func TestEnsureTraceInData_NoInventWhenMissing(t *testing.T) {
	data := EnsureTraceInData(context.Background(), map[string]interface{}{"k": "v"})
	if _, ok := data["trace_id"]; ok {
		t.Fatalf("unexpected trace_id = %v", data["trace_id"])
	}
}

func TestLogEventConsume_IncludesTraceID(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "web-trace-consume123")
	out := captureStdout(t, func() {
		InitConsumer("task-events-test")
		LogEventConsume(ctx, "dispatch_ok", "CLOUD_SERVER_START_AUTO", map[string]any{
			"task_id": "task_1",
		})
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["msg"] != "event_consume" {
		t.Fatalf("msg = %v", payload["msg"])
	}
	if payload["trace_id"] != "web-trace-consume123" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
	if payload["consume_stage"] != "dispatch_ok" {
		t.Fatalf("consume_stage = %v", payload["consume_stage"])
	}
}

func TestLogEventPublish_IncludesTraceID(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "web-trace-publish123")
	out := captureStdout(t, func() {
		Init("task-cloud-service-test")
		LogEventPublish(ctx, "CLOUD_SERVER_START_AUTO", "cloud-server-start-auto", map[string]any{
			"task_id": "task_1",
		})
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["msg"] != "event_publish" {
		t.Fatalf("msg = %v", payload["msg"])
	}
	if payload["trace_id"] != "web-trace-publish123" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
}

func TestOtelTraceIDHex_UUID(t *testing.T) {
	got := OtelTraceIDHex("b7906027-3792-4215-9fab-6b044565a272")
	want := "b7906027379242159fab6b044565a272"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveTraceID(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "from-context12345678")
	if got := ResolveTraceID(ctx, "from-header123456789"); got != "from-header123456789" {
		t.Fatalf("header preferred: %q", got)
	}
	if got := ResolveTraceID(ctx, ""); got != "from-context12345678" {
		t.Fatalf("context fallback: %q", got)
	}
}

func lastJSONLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			return line
		}
	}
	return strings.TrimSpace(out)
}

func TestMiddleware_ImpersonationHeadersInHTTPLog(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-service-test")
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		}))
		req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
		req.Header.Set(Header, "trace-impersonate1234")
		req.Header.Set(ParentSpanHeader, "a1b2c3d4e5f67890")
		req.Header.Set(HeaderImpersonatorID, "admin-1")
		req.Header.Set("X-User-Id", "user-2")
		req.Header.Set(HeaderImpersonationSessionID, "99")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(lastJSONLine(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["impersonating"] != true {
		t.Fatalf("impersonating=%v payload=%v", payload["impersonating"], payload)
	}
	if payload["impersonator_user_id"] != "admin-1" {
		t.Fatalf("impersonator=%v", payload["impersonator_user_id"])
	}
	if payload["impersonated_user_id"] != "user-2" {
		t.Fatalf("impersonated=%v", payload["impersonated_user_id"])
	}
	if payload["impersonation_session_id"] != "99" {
		t.Fatalf("session=%v", payload["impersonation_session_id"])
	}
}

func TestMiddleware_NoImpersonationFieldsWhenHeadersAbsent(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-service-test")
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		}))
		req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
		req.Header.Set(Header, "trace-plain12345678")
		req.Header.Set(ParentSpanHeader, "a1b2c3d4e5f67890")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(lastJSONLine(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if _, ok := payload["impersonating"]; ok {
		t.Fatalf("unexpected impersonating in %v", payload)
	}
}

func TestMiddleware_SetImpersonationSeenInHTTPLog(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-service-test")
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetImpersonation(r.Context(), Impersonation{
				ImpersonatorUserID: "admin-9",
				ImpersonatedUserID: "user-8",
				SessionID:          "77",
			})
			w.WriteHeader(204)
		}))
		req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
		req.Header.Set(Header, "trace-setimperson12")
		req.Header.Set(ParentSpanHeader, "a1b2c3d4e5f67890")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(lastJSONLine(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["impersonator_user_id"] != "admin-9" || payload["impersonation_session_id"] != "77" {
		t.Fatalf("payload=%v", payload)
	}
}
