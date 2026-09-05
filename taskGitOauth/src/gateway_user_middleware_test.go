package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gatewayauth"
	"taskGitOauth/infrastructure"
)

// Regression for OPT-20260812-044: APISIX forward-auth injects only X-User-Id
// (+ gateway verified headers), NOT X-Auth-User-Id. gatewayUserMiddleware must
// promote the verified identity into X-Auth-User-Id so effectiveUserID resolves
// the user even when X-Auth-User-Id is absent on the wire.
func TestGatewayUserMiddlewarePromotesVerifiedXUserID(t *testing.T) {
	app := &App{Cfg: &infrastructure.Config{GatewayInternalSecret: "secret"}}

	called := false
	h := app.gatewayUserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.Header.Get(gatewayauth.HeaderAuthUserID); got != "u123" {
			t.Fatalf("X-Auth-User-Id = %q, want %q", got, "u123")
		}
		if eff := app.effectiveUserID(r); eff != "u123" {
			t.Fatalf("effectiveUserID = %q, want %q", eff, "u123")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/git-oauth/providers/", nil)
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, "secret")
	req.Header.Set(gatewayauth.HeaderUserID, "u123")
	// 刻意不设置 X-Auth-User-Id：验证「仅 X-User-Id + verified」也能解析用户。
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called {
		t.Fatal("inner handler not called")
	}
}

// Without verified gateway headers the middleware must NOT inject X-Auth-User-Id.
// Unauthenticated requests pass through; handlers decide authorization.
func TestGatewayUserMiddlewareSkipsUnverified(t *testing.T) {
	app := &App{Cfg: &infrastructure.Config{GatewayInternalSecret: "secret"}}

	called := false
	h := app.gatewayUserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.Header.Get(gatewayauth.HeaderAuthUserID); got != "" {
			t.Fatalf("X-Auth-User-Id = %q, want empty", got)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/git-oauth/providers/", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u123") // no verified/secret headers
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called {
		t.Fatal("inner handler not called")
	}
}
