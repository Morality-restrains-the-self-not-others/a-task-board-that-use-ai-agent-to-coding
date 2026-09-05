package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayUserMiddlewarePromotesXUserId(t *testing.T) {
	cfg.GatewayInternalSecret = "test-gw-secret"
	called := false
	handler := gatewayUserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := getAuthUser(r); got != "u-gateway" {
			t.Fatalf("expected getAuthUser=u-gateway, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-TaskGateway-Internal-Secret", "test-gw-secret")
	req.Header.Set("X-User-Id", "u-gateway")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestGatewayUserMiddlewareRejectsUnverifiedXUserId(t *testing.T) {
	cfg.GatewayInternalSecret = "test-gw-secret"
	handler := gatewayUserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := getAuthUser(r); got != "" {
			t.Fatalf("expected empty getAuthUser for unverified X-User-Id, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/", nil)
	req.Header.Set("X-User-Id", "spoofed")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (middleware is non-blocking), got %d", rec.Code)
	}
}

func TestGetAuthUserIgnoresBareXUserId(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Id", "only-x-user")
	if got := getAuthUser(req); got != "" {
		t.Fatalf("getAuthUser must not trust bare X-User-Id, got %q", got)
	}
	req.Header.Set("X-Auth-User-Id", "canonical")
	if got := getAuthUser(req); got != "canonical" {
		t.Fatalf("expected canonical, got %q", got)
	}
}
