package tracelog

import (
	"bytes"
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

func TestMiddleware_PropagatesTraceHeader(t *testing.T) {
	Init("task-cloud-service-test")
	var got string
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = TraceIDFromContext(r.Context())
		w.WriteHeader(204)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	req.Header.Set(Header, "web-1783499653748-fnkl3zqin2e")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get(Header) != "web-1783499653748-fnkl3zqin2e" {
		t.Fatalf("response header = %q", rec.Header().Get(Header))
	}
	if got != "web-1783499653748-fnkl3zqin2e" {
		t.Fatalf("context trace = %q", got)
	}
}

func TestMiddleware_EmitsHttpRequestJSON(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-cloud-service-test")
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/", nil)
		req.Header.Set(Header, "web-1783499653748-fnkl3zqin2e")
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
	if payload["trace_id"] != "web-1783499653748-fnkl3zqin2e" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
	if payload["otel_trace_id"] != OtelTraceIDHex("web-1783499653748-fnkl3zqin2e") {
		t.Fatalf("otel_trace_id = %v", payload["otel_trace_id"])
	}
}

func TestOtelTraceIDHex_UUID(t *testing.T) {
	got := OtelTraceIDHex("b7906027-3792-4215-9fab-6b044565a272")
	want := "b7906027379242159fab6b044565a272"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
