package main

// Regression (nightly sweep 2026-08-06): commit 41d83e7 renamed the routes in
// RegisterRoutes from the legacy audience-prefixed paths (/api/auth/…,
// /api/vendor/…, /api/admin/…, /api/public/…) to the /api/ai-provider/…
// convention without updating the consumers of the legacy contract — openapi.go,
// the frontend (frontend/src/views/*, api.js), the e2e scripts and this test
// suite. Every legacy request fell through to the SPA catch-all and 404'd.
// Fix: register BOTH families (the gateway forwards /api/ai-provider/* without
// rewrite, the frontend still calls the legacy paths). This test pins that
// dual contract so a future rename must migrate all consumers in the same
// commit.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// DB-free inputs only: these request shapes reach their handler response
// before any database access (mirrors auth_handlers_test.go's minimal app).
func TestRouteContractLegacyAndConvention(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	cases := []struct {
		path string
		want int
	}{
		// legacy family — documented in openapi.go, consumed by frontend & e2e
		{"/api/auth/sso/exchange/", http.StatusFound},
		{"/api/auth/oidc/authorize/?role=vendor", http.StatusFound},
		{"/api/auth/oidc/callback/?code=x", http.StatusBadRequest},
		{"/api/vendor/auth/register/", http.StatusForbidden},
		{"/api/vendor/auth/login/", http.StatusForbidden},
		{"/api/admin/auth/login/", http.StatusForbidden},
		{"/api/vendor/auth/me/", http.StatusUnauthorized},
		{"/api/admin/auth/me/", http.StatusUnauthorized},
		// convention family — forwarded by taskGateway without path rewrite
		{"/api/ai-provider/sso-exchange/", http.StatusFound},
		{"/api/ai-provider/oidc-authorize/?role=vendor", http.StatusFound},
		{"/api/ai-provider/oidc-callback/?code=x", http.StatusBadRequest},
		{"/api/ai-provider/vendor-auth-register/", http.StatusForbidden},
		{"/api/ai-provider/vendor-auth-login/", http.StatusForbidden},
		{"/api/ai-provider/admin-auth-login/", http.StatusForbidden},
		{"/api/ai-provider/vendor-me/", http.StatusUnauthorized},
		{"/api/ai-provider/vendor-status/", http.StatusUnauthorized},
		{"/api/ai-provider/vendor-application/phone-status/", http.StatusUnauthorized},
		{"/api/ai-provider/admin-me/", http.StatusUnauthorized},
		{"/api/vendor/status/", http.StatusUnauthorized},
		{"/api/vendor/application/phone-status/", http.StatusUnauthorized},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: expected %d, got %d (route missing?)", c.path, c.want, rec.Code)
		}
	}
}
