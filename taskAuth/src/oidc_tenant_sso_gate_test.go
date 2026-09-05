package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskAuth/domain"
)

func startTenantMemberResolveServer(t *testing.T, members map[string]bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/internal/tenant/members/resolve") {
			http.NotFound(w, r)
			return
		}
		uid := r.URL.Query().Get("user_id")
		if members[uid] {
			writeJSON(w, http.StatusOK, map[string]any{"id": "m-" + uid, "is_admin": true})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func startTenantMemberResolveServer500(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOidcTenantSsoAuthorize_MemberGetsCode(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	companyID := "877397588196749312"
	redirect := "http://127.0.0.1:8929/users/auth/openid_connect/callback"
	uris, err := tenantOidcRedirectURIsJSON(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertTenantOidcClient(companyID, "t", uris, "plain-secret"); err != nil {
		t.Fatal(err)
	}
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatal(err)
	}
	cfg.TenantServiceURL = startTenantMemberResolveServer(t, map[string]bool{lm.ObjectID: true}).URL
	token := oidcTestToken(t)
	clientID, _ := domain.TenantGitLabOidcClientID(companyID)
	rec := oidcAuthorizeAsUser(t, token, clientID, redirect)
	assertOidcCodeIssued(t, rec)
}

func TestOidcTenantSsoAuthorize_NonMemberDenied(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	companyID := "877397588196749312"
	redirect := "http://127.0.0.1:8929/users/auth/openid_connect/callback"
	uris, _ := tenantOidcRedirectURIsJSON(redirect)
	if err := insertTenantOidcClient(companyID, "t", uris, "plain-secret"); err != nil {
		t.Fatal(err)
	}
	cfg.TenantServiceURL = startTenantMemberResolveServer(t, map[string]bool{}).URL
	token := oidcTestToken(t)
	clientID, _ := domain.TenantGitLabOidcClientID(companyID)
	rec := oidcAuthorizeAsUser(t, token, clientID, redirect)
	assertOidcAccessDenied(t, rec)
}

func TestOidcTenantSsoAuthorize_LookupFailureDenied(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	companyID := "c1"
	redirect := "http://127.0.0.1:8929/users/auth/openid_connect/callback"
	uris, _ := tenantOidcRedirectURIsJSON(redirect)
	if err := insertTenantOidcClient(companyID, "t", uris, "plain-secret"); err != nil {
		t.Fatal(err)
	}
	cfg.TenantServiceURL = startTenantMemberResolveServer500(t).URL
	token := oidcTestToken(t)
	clientID, _ := domain.TenantGitLabOidcClientID(companyID)
	rec := oidcAuthorizeAsUser(t, token, clientID, redirect)
	assertOidcAccessDenied(t, rec)
}

func TestOidcTenantSsoAuthorize_DoesNotUseRegionGatePrefix(t *testing.T) {
	if domain.RegionGateApplies("gitlab-tenant-x") {
		t.Fatal("tenant client must not use region gate")
	}
	if !domain.RegionGateApplies("gitlab-git-service") {
		t.Fatal("platform client still uses region gate")
	}
}

func TestEnsureOidcClientTenantManagedRowNotOverwritten(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	uris := `["https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"]`
	if _, err := db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, managed_by, owner_company_id, purpose, created_at, updated_at)
		VALUES ('9100', 'gitlab-tenant-keep', 'h', 'n', ?, 'tenant', 'keep', 'tenant_gitlab_sso', NOW(), NOW())`, uris); err != nil {
		t.Fatal(err)
	}
	if err := ensureOidcClient("gitlab-tenant-keep", "other", "n", `["http://conf.example/cb"]`); err != nil {
		t.Fatal(err)
	}
	row, err := loadOidcClient("gitlab-tenant-keep")
	if err != nil || row == nil {
		t.Fatal(err)
	}
	if row.RedirectURIs != uris {
		t.Fatalf("tenant row overwritten: %s", row.RedirectURIs)
	}
}

func TestOidcTenantSsoCRUD_IssueGetRotateDisable(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatal(err)
	}
	tid := "tid-sso-1"
	cfg.TenantServiceURL = startTenantMemberResolveServer(t, map[string]bool{lm.ObjectID: true}).URL
	cfg.GitOauthServiceURL = startPathAServer(t, "http://127.0.0.1:8929").URL
	mux := http.NewServeMux()
	mountRoutes(mux)

	putBody := `{"base_url":"http://127.0.0.1:8929"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/"+tid+"/gitlab-oidc-sso/", strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", lm.ObjectID)
	req.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	req.Header.Set("Idempotency-Key", "enable-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT enable %d %s", rec.Code, rec.Body.String())
	}
	var first map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	secret, _ := first["client_secret"].(string)
	if secret == "" || first["configured"] != true {
		t.Fatalf("expected secret once: %+v", first)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tid+"/gitlab-oidc-sso/", nil)
	req2.Header.Set("X-User-Id", lm.ObjectID)
	req2.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	var got map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &got)
	if _, has := got["client_secret"]; has && strings.TrimSpace(fmtString(got["client_secret"])) != "" {
		t.Fatalf("GET must not return secret: %+v", got)
	}

	req3 := httptest.NewRequest(http.MethodPut, "/api/tenant/"+tid+"/gitlab-oidc-sso/", strings.NewReader(putBody))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-User-Id", lm.ObjectID)
	req3.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	req3.Header.Set("Idempotency-Key", "enable-2")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	var second map[string]any
	_ = json.Unmarshal(rec3.Body.Bytes(), &second)
	if _, has := second["client_secret"]; has && strings.TrimSpace(fmtString(second["client_secret"])) != "" {
		t.Fatalf("second PUT must not mint secret: %+v", second)
	}

	reqR := httptest.NewRequest(http.MethodPost, "/api/tenant/"+tid+"/gitlab-oidc-sso/rotate/", nil)
	reqR.Header.Set("X-User-Id", lm.ObjectID)
	reqR.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	reqR.Header.Set("Idempotency-Key", "rot-1")
	recR := httptest.NewRecorder()
	mux.ServeHTTP(recR, reqR)
	var rotated map[string]any
	_ = json.Unmarshal(recR.Body.Bytes(), &rotated)
	newSecret, _ := rotated["client_secret"].(string)
	if recR.Code != http.StatusOK || newSecret == "" || newSecret == secret {
		t.Fatalf("rotate: %d %+v", recR.Code, rotated)
	}
	reqR2 := httptest.NewRequest(http.MethodPost, "/api/tenant/"+tid+"/gitlab-oidc-sso/rotate/", nil)
	reqR2.Header.Set("X-User-Id", lm.ObjectID)
	reqR2.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	reqR2.Header.Set("Idempotency-Key", "rot-1")
	recR2 := httptest.NewRecorder()
	mux.ServeHTTP(recR2, reqR2)
	var rotatedReplay map[string]any
	_ = json.Unmarshal(recR2.Body.Bytes(), &rotatedReplay)
	if _, has := rotatedReplay["client_secret"]; has && strings.TrimSpace(fmtString(rotatedReplay["client_secret"])) != "" {
		t.Fatalf("rotate replay must not return secret")
	}

	reqD := httptest.NewRequest(http.MethodDelete, "/api/tenant/"+tid+"/gitlab-oidc-sso/", nil)
	reqD.Header.Set("X-User-Id", lm.ObjectID)
	reqD.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	reqD.Header.Set("Idempotency-Key", "del-1")
	recD := httptest.NewRecorder()
	mux.ServeHTTP(recD, reqD)
	if recD.Code != http.StatusNoContent {
		t.Fatalf("DELETE %d %s", recD.Code, recD.Body.String())
	}
}

func TestOidcTenantSsoPut_BaseURLMismatch(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatal(err)
	}
	tid := "tid-mismatch"
	cfg.TenantServiceURL = startTenantMemberResolveServer(t, map[string]bool{lm.ObjectID: true}).URL
	cfg.GitOauthServiceURL = startPathAServer(t, "https://saved.example").URL
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/"+tid+"/gitlab-oidc-sso/", strings.NewReader(`{"base_url":"https://other.example"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", lm.ObjectID)
	req.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	req.Header.Set("Idempotency-Key", "en-x")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func startPathAServer(t *testing.T, baseURL string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"configured": true,
			"base_url":   baseURL,
			"active":     true,
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fmtString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func TestOidcTenantSsoPut_HTTPPublicAllowed(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatal(err)
	}
	tid := "tid-http"
	cfg.TenantServiceURL = startTenantMemberResolveServer(t, map[string]bool{lm.ObjectID: true}).URL
	cfg.GitOauthServiceURL = startPathAServer(t, "http://gitlab.daydaymoney.com").URL
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/"+tid+"/gitlab-oidc-sso/", strings.NewReader(`{"base_url":"http://gitlab.daydaymoney.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", lm.ObjectID)
	req.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	req.Header.Set("Idempotency-Key", "en-http")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for public http, got %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["configured"] != true {
		t.Fatalf("expected configured: %+v", body)
	}
	secret, _ := body["client_secret"].(string)
	if secret == "" {
		t.Fatalf("expected one-time secret: %+v", body)
	}
}
