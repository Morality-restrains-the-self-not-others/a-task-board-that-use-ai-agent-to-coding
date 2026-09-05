package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dbload "dbload"
)

// oidcTestSetup creates an in-memory test environment with DB + signing key + OIDC client + user.
func oidcTestSetup(t *testing.T) (root string, cleanup func()) {
	t.Helper()
	root = repoRoot()

	// Save & restore globals
	oldDB := db
	oldCfg := cfg
	oldSigningKey := signingKey

	t.Cleanup(func() {
		db = oldDB
		cfg = oldCfg
		signingKey = oldSigningKey
	})

	setupAuthTestDB(t)

	// Set up minimal config
	cfg = Config{
		Host:           "127.0.0.1",
		Port:           8003,
		InternalSecret: "",
		// SQLitePath removed — taskAuth uses MySQL only
		GatewayPublicBase:   "http://127.0.0.1:8003",
		OidcAccessTokenTTL:  3600,
		OidcIDTokenTTL:      3600,
		OidcRefreshTokenTTL: 2592000,
		nowUnix:             func() int64 { return time.Now().Unix() },
	}

	if err := loadUserContentTypeID(); err != nil {
		t.Fatalf("content_type: %v", err)
	}

	// Init signing key
	if err := initOidcSigningKey(""); err != nil {
		t.Fatalf("initOidcSigningKey: %v", err)
	}

	// Seed test OIDC client (new multi-client API)
	cfg.OidcBootstrapClients = []OidcBootstrapClient{
		{ClientID: "test-oidc-client", ClientSecret: "test-oidc-secret", RedirectURI: "http://127.0.0.1:8012/users/auth/openid_connect/callback"},
	}
	if err := seedOidcBootstrapClients(); err != nil {
		t.Fatalf("seedOidcBootstrapClients: %v", err)
	}

	// Create test user
	userID, _, err := createUserWithEmailLogin("test@example.com", "testpassword123")
	if err != nil {
		t.Fatalf("createUserWithEmailLogin: %v", err)
	}
	_ = userID // used by tests via token lookup

	return root, func() {
		db.Close()
	}
}

func oidcTestToken(t *testing.T) string {
	t.Helper()
	// Find user by email
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatalf("findLoginMethodByEmail: err=%v row=%v", err, lm)
	}
	token, err := getOrCreateToken(lm.ObjectID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("getOrCreateToken: %v", err)
	}
	return token
}

func TestOidcDiscoveryEndpoint(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	handleOidcDiscovery(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	issuer, _ := doc["issuer"].(string)
	if issuer == "" {
		t.Fatal("issuer missing")
	}
	authEP, _ := doc["authorization_endpoint"].(string)
	if authEP == "" {
		t.Fatal("authorization_endpoint missing")
	}
	tokenEP, _ := doc["token_endpoint"].(string)
	if tokenEP == "" {
		t.Fatal("token_endpoint missing")
	}
	jwksURI, _ := doc["jwks_uri"].(string)
	if jwksURI == "" {
		t.Fatal("jwks_uri missing")
	}

	t.Logf("discovery issuer=%s auth=%s token=%s jwks=%s", issuer, authEP, tokenEP, jwksURI)
}

func TestOidcJWKSEndpoint(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/jwks", nil)
	rec := httptest.NewRecorder()
	handleOidcJWKS(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	keys, ok := doc["keys"].([]interface{})
	if !ok || len(keys) == 0 {
		t.Fatalf("keys missing or empty: %v", doc)
	}

	key := keys[0].(map[string]interface{})
	if key["kty"] != "RSA" {
		t.Fatalf("expected kty=RSA, got %v", key["kty"])
	}
	if key["alg"] != "RS256" {
		t.Fatalf("expected alg=RS256, got %v", key["alg"])
	}
}

// OPT-20260825-032 回归：租户路径 JWKS 与全局 JWKS 返回相同签名密钥集
// （URL 已按租户分片，但密钥材料尚未分片——过早拆钥会让已签发 token 验签失败）。
func TestOidcJWKSTenantPathSharesGlobalKeys(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	globalReq := httptest.NewRequest(http.MethodGet, "/api/oidc/jwks", nil)
	globalRec := httptest.NewRecorder()
	handleOidcJWKS(globalRec, globalReq)
	if globalRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for global jwks, got %d", globalRec.Code)
	}

	tenantReq := httptest.NewRequest(http.MethodGet, "/api/oidc/tenant-abc/jwks", nil)
	tenantRec := httptest.NewRecorder()
	handleOidcJWKS(tenantRec, tenantReq)
	if tenantRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for tenant jwks, got %d", tenantRec.Code)
	}

	var globalDoc map[string]interface{}
	if err := json.Unmarshal(globalRec.Body.Bytes(), &globalDoc); err != nil {
		t.Fatalf("unmarshal global jwks: %v", err)
	}
	var tenantDoc map[string]interface{}
	if err := json.Unmarshal(tenantRec.Body.Bytes(), &tenantDoc); err != nil {
		t.Fatalf("unmarshal tenant jwks: %v", err)
	}

	globalKeys, _ := globalDoc["keys"].([]interface{})
	tenantKeys, _ := tenantDoc["keys"].([]interface{})
	if len(globalKeys) == 0 || len(tenantKeys) == 0 {
		t.Fatal("both jwks must expose signing keys")
	}
	if len(globalKeys) != len(tenantKeys) {
		t.Fatalf("key count mismatch: global=%d tenant=%d", len(globalKeys), len(tenantKeys))
	}
	// 同一把签名钥：kid 必须一致（租户路径不得引入独立 kid 导致验签失败）
	globalKid := globalKeys[0].(map[string]interface{})["kid"]
	tenantKid := tenantKeys[0].(map[string]interface{})["kid"]
	if globalKid != tenantKid {
		t.Fatalf("kid mismatch: global=%v tenant=%v — tenant path must share the global signing key", globalKid, tenantKid)
	}
}

func TestOidcAuthorizeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "somestate")
	params.Set("nonce", "somenonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	// Should redirect to login — 302 status
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect to login, got %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}
	t.Logf("redirect to login: %s", location)
}

func TestOidcAuthorizeUnauthenticatedPrefersWwwLoginBase(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GatewayPublicBase = "https://api.daydaymoney.com"
	cfg.OidcIssuer = "https://api.daydaymoney.com"
	// Resolved postLogoutRedirectOrigins (confload must expand ${scheme}/${subdomains.www}).
	cfg.PostLogoutOrigins = []string{
		"https://www.daydaymoney.com",
		"https://example.com",
		"http://127.0.0.1:4000",
		"http://localhost:4000",
	}

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "somestate")
	params.Set("nonce", "somenonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect to login, got %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "https://www.daydaymoney.com/auth/login/?next=") {
		t.Fatalf("expected www login redirect from resolved PostLogoutOrigins, got %q", location)
	}
	// OPT-20260826-018：next 用相对路径，登录页 sanitizeOidcResumeNext 接受
	// /api/oidc/authorize 形态并拼回 gateway，不依赖 www/apex/api 精确 host。
	if !strings.Contains(location, url.QueryEscape("/api/oidc/authorize")) {
		t.Fatalf("expected next to be relative authorize URL, got %q", location)
	}
}

func TestOidcAuthorizeUnauthenticatedTenantPathUsesRelativeNext(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GatewayPublicBase = "https://api.daydaymoney.com"
	cfg.OidcIssuer = "https://api.daydaymoney.com"
	cfg.PostLogoutOrigins = []string{"https://www.daydaymoney.com"}

	// Tenant-scoped client: gitlab-tenant-{companyID} → ownerCompanyID equals path tenant,
	// so rejectOidcPathTenantMismatch passes and we reach the unauthenticated redirect.
	const tid = "tenant-abc123"
	cfg.OidcBootstrapClients = []OidcBootstrapClient{
		{ClientID: "gitlab-tenant-" + tid, ClientSecret: "test-oidc-secret", RedirectURI: "http://127.0.0.1:8012/users/auth/openid_connect/callback"},
	}
	if err := seedOidcBootstrapClients(); err != nil {
		t.Fatalf("seedOidcBootstrapClients: %v", err)
	}

	params := url.Values{}
	params.Set("client_id", "gitlab-tenant-"+tid)
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "somestate")
	params.Set("nonce", "somenonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/"+tid+"/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect to login, got %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "https://www.daydaymoney.com/auth/login/?next=") {
		t.Fatalf("expected www login redirect, got %q", location)
	}
	if !strings.Contains(location, url.QueryEscape("/api/oidc/"+tid+"/authorize")) {
		t.Fatalf("expected next to be relative tenant authorize URL, got %q", location)
	}
}

func TestOidcAuthorizeUnauthenticatedSkipsUnresolvedTemplates(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GatewayPublicBase = "https://api.daydaymoney.com"
	cfg.OidcIssuer = "https://api.daydaymoney.com"
	cfg.PostLogoutOrigins = []string{
		"${scheme}://${subdomains.www}",
		"http://127.0.0.1:4000",
	}

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "somestate")
	params.Set("nonce", "somenonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	// Without resolved www origin, fall back to GatewayPublicBase (no gateway→www invent).
	if !strings.HasPrefix(location, "https://api.daydaymoney.com/auth/login/?next=") {
		t.Fatalf("expected gateway login fallback when templates unresolved, got %q", location)
	}
}

func TestOidcAuthorizeSuccess(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	token := oidcTestToken(t)

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "somestate")
	params.Set("nonce", "somenonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}
	if !strings.Contains(location, "code=") {
		t.Fatalf("expected code in redirect: %s", location)
	}
	if !strings.Contains(location, "state=somestate") {
		t.Fatalf("expected state in redirect: %s", location)
	}
	t.Logf("authorize redirect: %s", location)
}

func TestOidcTokenExchange(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	token := oidcTestToken(t)

	// Step 1: Authorize to get a code
	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "teststate")
	params.Set("nonce", "testnonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("authorize: expected 302, got %d", rec.Code)
	}
	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := location.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in redirect: %s", location)
	}
	state := location.Query().Get("state")
	if state != "teststate" {
		t.Fatalf("state mismatch: %q", state)
	}
	t.Logf("got code: %s", code[:16]+"...")

	// Step 2: Exchange code for tokens
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)
	tokenForm.Set("client_id", "test-oidc-client")
	tokenForm.Set("client_secret", "test-oidc-secret")
	tokenForm.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")

	req2 := httptest.NewRequest(http.MethodPost, "/api/oidc/token", strings.NewReader(tokenForm.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	handleOidcToken(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("token: expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal token resp: %v", err)
	}

	idToken, _ := resp["id_token"].(string)
	accessToken, _ := resp["access_token"].(string)
	tokenType, _ := resp["token_type"].(string)

	if idToken == "" {
		t.Fatal("id_token missing")
	}
	if accessToken == "" {
		t.Fatal("access_token missing")
	}
	if tokenType != "Bearer" {
		t.Fatalf("token_type: %q", tokenType)
	}

	// Verify id_token is valid JWT
	claims, err := verifyRS256Signature(idToken)
	if err != nil {
		t.Fatalf("verify id_token: %v", err)
	}
	if sub, _ := claims["sub"].(string); sub == "" {
		t.Fatal("id_token sub missing")
	}
	if aud, _ := claims["aud"].(string); aud != "test-oidc-client" {
		t.Fatalf("id_token aud: %q", aud)
	}
	if email, _ := claims["email"].(string); email != "test@example.com" {
		t.Fatalf("id_token email: %q", email)
	}
	if nonce, _ := claims["nonce"].(string); nonce != "testnonce" {
		t.Fatalf("id_token nonce: %q", nonce)
	}

	t.Logf("id_token sub=%s email=%s", claims["sub"], claims["email"])
	t.Logf("access_token length=%d", len(accessToken))
}

func TestOidcTokenInvalidClient(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "nonexistent")
	form.Set("client_id", "test-oidc-client")
	form.Set("client_secret", "wrong-secret")
	form.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")

	req := httptest.NewRequest(http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handleOidcToken(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOidcUserInfo(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	token := oidcTestToken(t)

	// Get authorization code
	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "test")
	params.Set("nonce", "test")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	location, _ := url.Parse(rec.Header().Get("Location"))
	code := location.Query().Get("code")

	// Exchange for tokens
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)
	tokenForm.Set("client_id", "test-oidc-client")
	tokenForm.Set("client_secret", "test-oidc-secret")
	tokenForm.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")

	req2 := httptest.NewRequest(http.MethodPost, "/api/oidc/token", strings.NewReader(tokenForm.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	handleOidcToken(rec2, req2)

	var tokenResp map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &tokenResp)
	accessToken, _ := tokenResp["access_token"].(string)

	// Call userinfo
	req3 := httptest.NewRequest(http.MethodGet, "/api/oidc/userinfo", nil)
	req3.Header.Set("Authorization", "Bearer "+accessToken)
	rec3 := httptest.NewRecorder()
	handleOidcUserInfo(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("userinfo: expected 200, got %d body=%s", rec3.Code, rec3.Body.String())
	}

	var info map[string]interface{}
	if err := json.Unmarshal(rec3.Body.Bytes(), &info); err != nil {
		t.Fatalf("unmarshal userinfo: %v", err)
	}

	if email, _ := info["email"].(string); email != "test@example.com" {
		t.Fatalf("userinfo email: %q", email)
	}
	if sub, _ := info["sub"].(string); sub == "" {
		t.Fatal("userinfo sub missing")
	}
	t.Logf("userinfo: sub=%s email=%s", info["sub"], info["email"])
}

func TestOidcJWTSignAndVerify(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	issuer := "http://test-issuer"
	clientID := "test-client"
	userID := "test-user-123"
	nonce := "test-nonce-456"

	idToken, err := makeIDToken(userID, "test@example.com", "testuser", issuer, clientID, nonce, []string{"openid", "email", "profile"})
	if err != nil {
		t.Fatalf("makeIDToken: %v", err)
	}

	claims, err := verifyRS256Signature(idToken)
	if err != nil {
		t.Fatalf("verifyRS256Signature: %v", err)
	}

	if sub, _ := claims["sub"].(string); sub != userID {
		t.Fatalf("subject: expected %s, got %s", userID, sub)
	}
	if iss, _ := claims["iss"].(string); iss != issuer {
		t.Fatalf("issuer: expected %s, got %s", issuer, iss)
	}
	if aud, _ := claims["aud"].(string); aud != clientID {
		t.Fatalf("audience: expected %s, got %s", aud, clientID)
	}

	t.Logf("JWT valid: sub=%s iss=%s aud=%s", claims["sub"], claims["iss"], claims["aud"])
}

func TestOidcAuthorizeInvalidClient(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	params := url.Values{}
	params.Set("client_id", "nonexistent-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	// Invalid client — should return JSON error (not redirect to untrusted URI)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var errResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp["error"] != "unauthorized_client" {
		t.Fatalf("expected unauthorized_client error: %v", errResp)
	}
}

func TestOidcCodeSingleUse(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	token := oidcTestToken(t)

	// Get a code
	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "test")
	params.Set("nonce", "test")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	location, _ := url.Parse(rec.Header().Get("Location"))
	code := location.Query().Get("code")

	exchangeForm := func() *strings.Reader {
		f := url.Values{}
		f.Set("grant_type", "authorization_code")
		f.Set("code", code)
		f.Set("client_id", "test-oidc-client")
		f.Set("client_secret", "test-oidc-secret")
		f.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
		return strings.NewReader(f.Encode())
	}

	// First exchange — should succeed
	req1 := httptest.NewRequest(http.MethodPost, "/api/oidc/token", exchangeForm())
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec1 := httptest.NewRecorder()
	handleOidcToken(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first exchange: expected 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}

	// Second exchange — should fail (code already used)
	req2 := httptest.NewRequest(http.MethodPost, "/api/oidc/token", exchangeForm())
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	handleOidcToken(rec2, req2)

	if rec2.Code == http.StatusOK {
		t.Fatalf("second exchange: expected error, got 200 body=%s", rec2.Body.String())
	}
	t.Logf("code single-use: second exchange returned %d (expected non-200)", rec2.Code)
}

func TestOidcMigrationsAddsOidcTables(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)

	if err := runDataMigrateFromDir(testDSN, repoRoot()); err != nil {
		t.Fatalf("runDataMigrateFromDir: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"auth_oidc_client", "auth_oidc_authorization"} {
		var name string
		err := db.QueryRow(
			`SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("OIDC table %s missing: %v", table, err)
		}
	}
}

func TestEnsureOidcClientIdempotent(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	n, err := countOidcClients()
	if err != nil {
		t.Fatalf("countOidcClients: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 client, got %d", n)
	}

	// Seed again — should be idempotent
	if err := seedOidcBootstrapClients(); err != nil {
		t.Fatalf("second seedOidcBootstrapClients: %v", err)
	}

	n2, _ := countOidcClients()
	if n2 != n {
		t.Fatalf("idempotent check: %d → %d", n, n2)
	}
}

func TestOidcTokenRequiresCode(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "nonexistent-code")
	form.Set("client_id", "test-oidc-client")
	form.Set("client_secret", "test-oidc-secret")
	form.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")

	req := httptest.NewRequest(http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handleOidcToken(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected error for nonexistent code, got 200")
	}
}

func TestOidcAuthorizeInvalidResponseType(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "token") // unsupported
	params.Set("scope", "openid")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	// Should return JSON 400 (no redirect before redirect_uri validated)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var errResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp["error"] != "unsupported_response_type" {
		t.Fatalf("expected unsupported_response_type: %v", errResp)
	}
}

func TestOidcAuthorizeMissingOpenidScope(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "profile email") // no openid

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	// Should return JSON 400 (no redirect for invalid requests before URI validation)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var errResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp["error"] != "invalid_scope" {
		t.Fatalf("expected invalid_scope: %v", errResp)
	}
}

func TestOidcAuthorizeInvalidRedirectURI(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	params := url.Values{}
	params.Set("client_id", "test-oidc-client")
	params.Set("redirect_uri", "http://evil.com/callback") // not in allowlist
	params.Set("response_type", "code")
	params.Set("scope", "openid")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	// Should return JSON 400 (no redirect to untrusted URI)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var errResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp["error"] != "invalid_request" {
		t.Fatalf("expected invalid_request: %v", errResp)
	}
}

func TestOidcUserInfoNoAuth(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/userinfo", nil)
	rec := httptest.NewRecorder()
	handleOidcUserInfo(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestOidcSigningKeyPersistence(t *testing.T) {
	tmp := t.TempDir()
	keyPath := filepath.Join(tmp, "oidc_signing_key.pem")

	// Reset global
	oldKey := signingKey
	signingKey = nil
	t.Cleanup(func() { signingKey = oldKey })

	// First init — should generate and persist
	if err := initOidcSigningKey(keyPath); err != nil {
		t.Fatalf("first initOidcSigningKey: %v", err)
	}
	if signingKey == nil {
		t.Fatal("signingKey nil after first init")
	}
	firstKeyID := signingKey.KeyID
	firstJWKS, _ := jwksJSON()

	// Verify file exists
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("key file not created: %v", err)
	}

	// Reset and reload
	signingKey = nil
	if err := initOidcSigningKey(keyPath); err != nil {
		t.Fatalf("second initOidcSigningKey: %v", err)
	}
	if signingKey == nil {
		t.Fatal("signingKey nil after second init")
	}
	secondKeyID := signingKey.KeyID
	secondJWKS, _ := jwksJSON()

	if firstKeyID != secondKeyID {
		t.Fatalf("key ID changed across restarts: %s → %s", firstKeyID, secondKeyID)
	}

	// JWKS should be identical
	firstKeys := firstJWKS["keys"].([]map[string]interface{})
	secondKeys := secondJWKS["keys"].([]map[string]interface{})
	if firstKeys[0]["n"] != secondKeys[0]["n"] {
		t.Fatal("JWKS modulus changed across restarts")
	}

	t.Logf("key persistence OK: kid=%s file=%s", firstKeyID, keyPath)
}

func TestOidcSigningKeyDefaultPath(t *testing.T) {
	oldCfg := cfg
	cfg = Config{}
	t.Cleanup(func() { cfg = oldCfg })

	got := defaultSigningKeyPath()
	if !strings.Contains(got, filepath.Join("conf-local", "auth", "task-auth", "oidc_signing_key.pem")) {
		t.Fatalf("defaultSigningKeyPath: expected conf-local auth pem, got %s", got)
	}
}

func writeFakeConfigRoot(t *testing.T, tmp string) {
	t.Helper()
	conf := filepath.Join(tmp, "conf")
	if err := os.MkdirAll(conf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conf, "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOidcSigningKeyDeployModeMissingFails(t *testing.T) {
	tmp := t.TempDir()
	writeFakeConfigRoot(t, tmp)
	t.Setenv("DEPLOY_ROOT", tmp)
	t.Setenv("CONF_ROOT", filepath.Join(tmp, "conf"))
	t.Setenv("DEPLOY_MODE", "1")
	oldKey := signingKey
	signingKey = nil
	t.Cleanup(func() { signingKey = oldKey })

	if err := initOidcSigningKey(""); err == nil {
		t.Fatal("expected DEPLOY_MODE missing PEM to fail")
	}
}

func TestOidcSigningKeyMigratesLegacyPath(t *testing.T) {
	tmp := t.TempDir()
	writeFakeConfigRoot(t, tmp)
	t.Setenv("DEPLOY_ROOT", tmp)
	t.Setenv("CONF_ROOT", filepath.Join(tmp, "conf"))
	t.Setenv("DEPLOY_MODE", "0")
	legacy := filepath.Join(tmp, "db", "task-auth", "oidc_signing_key.pem")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	oldKey := signingKey
	signingKey = nil
	t.Cleanup(func() { signingKey = oldKey })
	if err := initOidcSigningKey(legacy); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	signingKey = nil
	if err := initOidcSigningKey(""); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dest := filepath.Join(tmp, "conf-local", "auth", "task-auth", "oidc_signing_key.pem")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected migrated key at %s: %v", dest, err)
	}
}

func BenchmarkOidcJWTMakeAndVerify(b *testing.B) {
	tmp := b.TempDir()
	oldKey := signingKey
	signingKey = nil
	b.Cleanup(func() { signingKey = oldKey })
	if err := initOidcSigningKey(filepath.Join(tmp, "oidc_signing_key.pem")); err != nil {
		b.Fatalf("initOidcSigningKey: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, err := makeIDToken("user-"+fmt.Sprint(i), "user@example.com", "user", "issuer", "aud", "nonce", []string{"openid", "email", "profile"})
		if err != nil {
			b.Fatalf("makeIDToken: %v", err)
		}
		_, err = verifyRS256Signature(token)
		if err != nil {
			b.Fatalf("verify: %v", err)
		}
	}
}

func TestOidcAuthorizeClientNotFoundLogsClientIDNotSecret(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	var buf bytes.Buffer
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(oldDefault)

	const secret = "SUPER-SECRET-CLIENT-SECRET-xyz"
	params := url.Values{}
	params.Set("client_id", "unknown-client-abc")
	// 即使客户端误传 client_secret，WARN 日志也不得记录。
	params.Set("client_secret", secret)
	params.Set("redirect_uri", "http://127.0.0.1:8012/users/auth/openid_connect/callback")
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "somestate")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown client, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "unauthorized_client") {
		t.Fatalf("expected unauthorized_client error, body=%s", body)
	}
	if strings.Contains(body, secret) {
		t.Fatalf("error response leaked client_secret: %s", body)
	}

	logOut := buf.String()
	if !strings.Contains(logOut, "unknown-client-abc") {
		t.Fatalf("expected WARN log to include client_id, log=%s", logOut)
	}
	if strings.Contains(logOut, secret) {
		t.Fatalf("WARN log leaked client_secret: %s", logOut)
	}
}
