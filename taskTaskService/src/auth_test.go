package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveGatewayAuthUser(t *testing.T) {
	cfg.TaskGatewayInternalSecret = "test-gw-secret"

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-TaskGateway-Internal-Secret", "test-gw-secret")
	req.Header.Set("X-User-Id", "850256676127797248")

	if got := resolveGatewayAuthUser(req); got != "850256676127797248" {
		t.Fatalf("expected gateway user id, got %q", got)
	}

	req.Header.Set("X-TaskGateway-Internal-Secret", "wrong")
	if got := resolveGatewayAuthUser(req); got != "" {
		t.Fatalf("expected empty for wrong secret, got %q", got)
	}
}

func TestAuthMiddlewareAcceptsGatewayHeaders(t *testing.T) {
	cfg.TaskGatewayInternalSecret = "test-gw-secret"
	called := false
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-Auth-User-Id") != "u-gateway" {
			t.Fatalf("expected X-Auth-User-Id=u-gateway, got %q", r.Header.Get("X-Auth-User-Id"))
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/1/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-TaskGateway-Internal-Secret", "test-gw-secret")
	req.Header.Set("X-User-Id", "u-gateway")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("handler was not called")
	}
}
