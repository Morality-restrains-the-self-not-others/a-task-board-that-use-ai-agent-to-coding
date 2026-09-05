package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

// testAppMinimal creates an App with in-memory OIDC store,
// but without a real database (DB-reliant handlers will be skipped).
func testAppMinimal(t *testing.T) *App {
	t.Helper()
	cfg := &infrastructure.Config{
		SecretKey:        infrastructure.DefaultSecretKey,
		SSOJwtSecret:     "test-sso-secret-for-bridge-jwt",
		SSOJwtIssuer:     "task2app-sso",
		SSOAudience:      "saas-ai-provider",
		OIDCRpIssuer:     "https://oidc.example.com",
		OIDCClientID:     "ai-provider-test",
		OIDCClientSecret: "test-oidc-secret",
		OIDCScopes:       []string{"openid", "email", "profile"},
		OIDCPublicOrigin: "https://ai-provider.daydaymoney.com",
		JWTTTLSeconds:    28800,
	}
	return &App{
		Cfg:  cfg,
		DB:   nil, // No DB for these tests
		OIDC: &oidcStore{data: map[string]oidcSession{}},
		JWKS: nil,
	}
}

// ── handleSSOOnly ──

func TestHandleSSOOnly(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	paths := []string{
		"/api/vendor/auth/register/",
		"/api/vendor/auth/login/",
		"/api/admin/auth/login/",
	}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != 403 {
			t.Fatalf("%s: expected 403, got %d", p, rec.Code)
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if out["code"] != "sso_only" {
			t.Fatalf("%s: expected code=sso_only, got %v", p, out)
		}
	}
}

// ── handleSSOExchange validation tests ──

func TestHandleSSOExchangeMethodNotAllowed(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// PUT is not allowed — only GET (browser redirect) and POST (API call) are supported
	req := httptest.NewRequest(http.MethodPut, "/api/auth/sso/exchange/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSSOExchangeGetMissingBridge(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// GET with missing bridge should redirect to /?error=...
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "缺少+bridge") && !strings.Contains(loc, "bridge") {
		t.Fatalf("expected redirect with bridge error, got %s", loc)
	}
}

func TestHandleSSOExchangeGetInvalidJWT(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/?bridge=not-a-valid-jwt", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// GET with invalid bridge should redirect to /?error=...
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=") {
		t.Fatalf("expected redirect with error param, got %s", loc)
	}
}

func TestHandleSSOExchangeGetExpiredJWT(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	now := time.Now().Add(-2 * time.Hour).Unix()
	claims := map[string]any{
		"iss":   app.Cfg.SSOJwtIssuer,
		"aud":   app.Cfg.SSOAudience,
		"sub":   float64(42),
		"typ":   "vendor_bridge",
		"iat":   now,
		"exp":   now - 3600,
		"email": "test@example.com",
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/?bridge="+bridge, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "%E8%BF%87%E6%9C%9F") && !strings.Contains(loc, "过期") && !strings.Contains(loc, "expired") {
		t.Fatalf("expected redirect about expiration, got %s", loc)
	}
}

func TestHandleSSOExchangeGetMissingTyp(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	now := time.Now().Unix()
	claims := map[string]any{
		"iss": app.Cfg.SSOJwtIssuer,
		"aud": app.Cfg.SSOAudience,
		"sub": float64(42),
		"iat": now,
		"exp": now + 3600,
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/?bridge="+bridge, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=") {
		t.Fatalf("expected redirect with error param, got %s", loc)
	}
}

func TestHandleSSOExchangeMissingBridge(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing bridge, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSSOExchangeInvalidJWT(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"bridge":"not-a-valid-jwt"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 for invalid JWT, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSSOExchangeExpiredJWT(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// Create expired JWT
	now := time.Now().Add(-2 * time.Hour).Unix()
	claims := map[string]any{
		"iss": app.Cfg.SSOJwtIssuer,
		"aud": app.Cfg.SSOAudience,
		"sub": float64(42),
		"typ": "staff_bridge",
		"iat": now,
		"exp": now - 3600, // expired 1 hour ago
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"bridge":"` + bridge + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 for expired JWT, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSSOExchangeMissingTyp(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	now := time.Now().Unix()
	claims := map[string]any{
		"iss": app.Cfg.SSOJwtIssuer,
		"aud": app.Cfg.SSOAudience,
		"sub": float64(42),
		// typ intentionally missing
		"iat": now,
		"exp": now + 3600,
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"bridge":"` + bridge + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 for missing typ, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ── handleOIDCAuthorize tests ──

func TestHandleOIDCAuthorizeMissingRole(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/authorize/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing role, got %d", rec.Code)
	}
}

func TestHandleOIDCAuthorizeInvalidRole(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/authorize/?role=invalid", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for invalid role, got %d", rec.Code)
	}
}

func TestHandleOIDCAuthorizeVendorRole(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/authorize/?role=vendor", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// Should return 302 redirect to OIDC provider
	if rec.Code != 302 {
		t.Fatalf("expected 302 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header")
	}
	if !strings.Contains(loc, app.Cfg.OIDCRpIssuer+"/api/oidc/authorize") {
		t.Fatalf("Location should point to OIDC authorize URL, got %s", loc)
	}
	if !strings.Contains(loc, "response_type=code") {
		t.Fatalf("Location should contain response_type=code, got %s", loc)
	}
	if !strings.Contains(loc, "code_challenge=") {
		t.Fatalf("Location should contain code_challenge PKCE param, got %s", loc)
	}
	if !strings.Contains(loc, "code_challenge_method=S256") {
		t.Fatalf("Location should contain code_challenge_method=S256, got %s", loc)
	}
	if !strings.Contains(loc, "state=") {
		t.Fatalf("Location should contain state param, got %s", loc)
	}
}

func TestHandleOIDCAuthorizeAdminRole(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/authorize/?role=admin", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 302 {
		t.Fatalf("expected 302 redirect, got %d", rec.Code)
	}
}

func TestHandleOIDCAuthorizeStoresSession(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/authorize/?role=vendor", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify session was stored
	app.OIDC.mu.Lock()
	count := len(app.OIDC.data)
	app.OIDC.mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 OIDC session stored, got %d", count)
	}
}

// ── handleOIDCCallback tests ──

func TestHandleOIDCCallbackMissingState(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback/?code=test", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// Returns 400 JSON for missing state param
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing state, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["detail"] != "缺少 state 参数" {
		t.Fatalf("expected '缺少 state 参数', got %v", out)
	}
}

func TestHandleOIDCCallbackMissingCode(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback/?state=test-state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing code, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleOIDCCallbackUnknownState(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback/?code=some-code&state=unknown-state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for unknown state, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if !strings.Contains(fmt.Sprint(out["detail"]), "过期") {
		t.Fatalf("expected session expired message, got %v", out)
	}
}

func TestHandleOIDCCallbackExpiredSession(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// Store an expired session directly
	app.OIDC.put("exp-state", oidcSession{
		CodeVerifier: "test",
		Role:         "vendor",
		Expires:      time.Now().Add(-1 * time.Hour),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback/?code=test-code&state=exp-state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for expired session, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ── handleVendorMe / handleStaffMe auth tests ──

func TestHandleVendorMeUnauthenticated(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/vendor/auth/me/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleStaffMeUnauthenticated(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auth/me/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// ── pkceChallenge ──

func TestPkceChallengeKnown(t *testing.T) {
	// Test with known verifier
	verifier := "test-verifier-value-1234567890"
	challenge := pkceChallenge(verifier)
	if challenge == "" || challenge == verifier {
		t.Fatalf("pkceChallenge should produce a hashed value, got %q", challenge)
	}
	// Should be base64url-encoded SHA256
	if len(challenge) != 43 {
		t.Fatalf("expected 43-char base64url SHA256, got %d chars: %q", len(challenge), challenge)
	}
}

func TestPkceChallengeEmpty(t *testing.T) {
	challenge := pkceChallenge("")
	if challenge == "" {
		t.Fatal("pkceChallenge of empty string should still produce a hash")
	}
}

// ── oidcStore tests ──

func TestOidcStorePutTake(t *testing.T) {
	s := &oidcStore{data: map[string]oidcSession{}}
	s.put("state-1", oidcSession{
		CodeVerifier: "verifier-1",
		Role:         "vendor",
		Expires:      time.Now().Add(10 * time.Minute),
	})

	session, ok := s.take("state-1")
	if !ok {
		t.Fatal("expected to find session")
	}
	if session.CodeVerifier != "verifier-1" {
		t.Fatalf("expected verifier-1, got %s", session.CodeVerifier)
	}
	if session.Role != "vendor" {
		t.Fatalf("expected role vendor, got %s", session.Role)
	}

	// Second take should fail (one-time use)
	_, ok = s.take("state-1")
	if ok {
		t.Fatal("second take should fail (session consumed)")
	}
}

func TestOidcStoreExpired(t *testing.T) {
	s := &oidcStore{data: map[string]oidcSession{}}
	s.put("expired-state", oidcSession{
		CodeVerifier: "expired-verifier",
		Role:         "admin",
		Expires:      time.Now().Add(-1 * time.Minute), // already expired
	})

	_, ok := s.take("expired-state")
	if ok {
		t.Fatal("expired session should not be returned")
	}
}

func TestOidcStoreUnknownState(t *testing.T) {
	s := &oidcStore{data: map[string]oidcSession{}}
	_, ok := s.take("nonexistent")
	if ok {
		t.Fatal("unknown state should not be found")
	}
}

func TestOidcStoreThreadSafety(t *testing.T) {
	s := &oidcStore{data: map[string]oidcSession{}}
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			s.put("key", oidcSession{Expires: time.Now().Add(time.Hour)})
			s.take("key")
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			s.put("key", oidcSession{Expires: time.Now().Add(time.Hour)})
			s.take("key")
		}
		done <- true
	}()

	<-done
	<-done
	// No race detector errors = pass
}
