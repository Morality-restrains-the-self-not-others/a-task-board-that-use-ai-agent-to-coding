package tracelog

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOtelTraceIDHex_uuid(t *testing.T) {
	got := OtelTraceIDHex("1cd1a1cc-e64d-4325-8b31-caabdd8aa74d")
	want := "1cd1a1cce64d43258b31caabdd8aa74d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMiddleware_propagates_incoming_trace_id(t *testing.T) {
	var got string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = TraceIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	req.Header.Set(Header, "1cd1a1cc-e64d-4325-8b31-caabdd8aa74d")
	rec := httptest.NewRecorder()
	Middleware(next).ServeHTTP(rec, req)

	if got != "1cd1a1cc-e64d-4325-8b31-caabdd8aa74d" {
		t.Fatalf("context trace_id = %q", got)
	}
	if rec.Header().Get(Header) != "1cd1a1cc-e64d-4325-8b31-caabdd8aa74d" {
		t.Fatalf("response header = %q", rec.Header().Get(Header))
	}
}

func TestMiddleware_skips_options(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/health/", nil)
	rec := httptest.NewRecorder()
	Middleware(next).ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected inner handler to run for OPTIONS")
	}
	if strings.Contains(rec.Body.String(), "http_request") {
		t.Fatal("did not expect access log body for OPTIONS")
	}
}
