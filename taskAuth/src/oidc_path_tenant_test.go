package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"taskAuth/domain"
)

func TestOidcTenantPathAuthorize_MatchIssuesCode(t *testing.T) {
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
	rec := oidcAuthorizeAsUserOnPath(t, "/api/oidc/"+companyID+"/authorize", token, clientID, redirect)
	assertOidcCodeIssued(t, rec)
}

func TestOidcTenantPathAuthorize_WrongTenantRejected(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	companyID := "877397588196749312"
	redirect := "http://127.0.0.1:8929/users/auth/openid_connect/callback"
	uris, _ := tenantOidcRedirectURIsJSON(redirect)
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
	rec := oidcAuthorizeAsUserOnPath(t, "/api/oidc/111111111111111111/authorize", token, clientID, redirect)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unauthorized_client") {
		t.Fatalf("expected unauthorized_client, body=%s", rec.Body.String())
	}
}

func TestOidcTenantSsoCRUD_SnippetHasTenantEndpoints(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatal(err)
	}
	tid := "877397588196749312"
	cfg.TenantServiceURL = startTenantMemberResolveServer(t, map[string]bool{lm.ObjectID: true}).URL
	cfg.GitOauthServiceURL = startPathAServer(t, "http://127.0.0.1:8929").URL
	mux := http.NewServeMux()
	mountRoutes(mux)

	putBody := `{"base_url":"http://127.0.0.1:8929"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/"+tid+"/gitlab-oidc-sso/", strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", lm.ObjectID)
	req.Header.Set("X-Tenant-Perms", tid+":region:settings.gitlab.main:operate")
	req.Header.Set("Idempotency-Key", "enable-path-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT enable %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	want := "/api/oidc/" + tid + "/authorize"
	if !strings.Contains(body, want) {
		t.Fatalf("snippet missing %q in %s", want, body)
	}
	if strings.Contains(body, `"https://`) && strings.Contains(body, `/api/oidc/authorize"`) && !strings.Contains(body, "/api/oidc/"+tid+"/authorize") {
		t.Fatalf("global authorize path leaked: %s", body)
	}
}

func oidcAuthorizeAsUserOnPath(t *testing.T, path, token, clientID, redirectURI string) *httptest.ResponseRecorder {
	t.Helper()
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "s1")
	params.Set("nonce", "n1")
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path+"?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
