package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func TestTenantGitlabConnectionPutGetDelete(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	tid := "company-42"
	path := "/api/git-oauth/tenant-connection/" + tid + "/gitlab-oauth-connection/"

	// GET empty (member)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET empty status %d body %s", rr.Code, rr.Body.String())
	}
	var empty map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if empty["configured"] != false {
		t.Fatalf("configured=%v", empty["configured"])
	}
	if empty["provider_key"] != "gitlab:tenant-"+tid {
		t.Fatalf("provider_key=%v", empty["provider_key"])
	}
	wantRedirect := "http://localhost:8002/api/accounts/tenant-" + tid + "/oauth/callback/"
	if asString(empty["redirect_uri"]) != wantRedirect {
		t.Fatalf("redirect_uri=%v want %v", empty["redirect_uri"], wantRedirect)
	}
	if strings.Contains(asString(empty["redirect_uri"]), "/api/accounts/tenant-gitlab/") {
		t.Fatal("shared tenant-gitlab redirect_uri must not be returned")
	}
	if _, hasSecret := empty["client_secret"]; hasSecret {
		t.Fatal("GET must not return client_secret")
	}
	if _, hasEnc := empty["client_secret_enc"]; hasEnc {
		t.Fatal("GET must not return client_secret_enc")
	}

	// PUT as non-admin → 403
	putBody := `{"base_url":"https://gitlab.daydaymoney.com","client_id":"cid","client_secret":"sekrit","remark":"demo"}`
	req = httptest.NewRequest(http.MethodPut, path, strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "u1")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("PUT member status %d body %s", rr.Code, rr.Body.String())
	}

	// PUT as admin
	req = httptest.NewRequest(http.MethodPut, path, strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin1")
	req.Header.Set("X-Is-Admin", "1")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT admin status %d body %s", rr.Code, rr.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved["configured"] != true {
		t.Fatalf("configured=%v", saved["configured"])
	}
	if saved["base_url"] != "https://gitlab.daydaymoney.com" {
		t.Fatalf("base_url=%v", saved["base_url"])
	}
	if saved["client_id"] != "cid" {
		t.Fatalf("client_id=%v", saved["client_id"])
	}
	if asString(saved["redirect_uri"]) != wantRedirect {
		t.Fatalf("PUT redirect_uri=%v want %v", saved["redirect_uri"], wantRedirect)
	}
	if _, has := saved["client_secret"]; has {
		t.Fatal("PUT response must not echo client_secret")
	}

	// Upsert uniqueness: second PUT updates same row
	putBody2 := `{"base_url":"https://gitlab.daydaymoney.com/","client_id":"cid2","client_secret":"sekrit2","remark":"v2"}`
	req = httptest.NewRequest(http.MethodPut, path, strings.NewReader(putBody2))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin1")
	req.Header.Set("X-Is-Admin", "1")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT update status %d", rr.Code)
	}
	row, err := app.DB.GetTenantGitLabConnection(tid)
	if err != nil || row == nil {
		t.Fatalf("row missing: %v", err)
	}
	if row.ClientID != "cid2" {
		t.Fatalf("client_id after update=%s", row.ClientID)
	}
	if row.BaseURL != "https://gitlab.daydaymoney.com" {
		t.Fatalf("normalized base_url=%s", row.BaseURL)
	}

	// GET returns updated, still no secret
	req = httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET after put %d", rr.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["client_id"] != "cid2" || got["remark"] != "v2" {
		t.Fatalf("got=%v", got)
	}

	// Cascade: insert credential then DELETE
	pk := domain.ProviderKeyForCompany(tid)
	_, err = app.DB.UpsertCredential(&infrastructure.CredentialRow{
		Provider:           pk,
		Task2appUserID:     "99",
		RefreshTokenCipher: "cipher",
		RemoteUserID:       "gl-1",
		RemoteLogin:        "alice",
		Scope:              "api",
		BindStatus:         "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	cred, err := app.DB.FindActiveCredential(pk, "99", "")
	if err != nil || cred == nil {
		t.Fatalf("cred missing before delete: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("X-User-Id", "admin1")
	req.Header.Set("X-Is-Admin", "1")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DELETE status %d body %s", rr.Code, rr.Body.String())
	}
	row, err = app.DB.GetTenantGitLabConnection(tid)
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatal("connection should be deleted")
	}
	cred, err = app.DB.FindActiveCredential(pk, "99", "")
	if err != nil {
		t.Fatal(err)
	}
	if cred != nil {
		t.Fatal("credential should be cascade-deleted")
	}
}

func TestResolveByServiceProviderLoadsTenantConnection(t *testing.T) {
	app := testApp(t)
	tid := "co-99"
	cipher, err := app.Fernet.Encrypt("super-secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       tid,
		BaseURL:         "https://git.corp.example",
		ClientID:        "app-id",
		ClientSecretEnc: cipher,
		Remark:          "corp",
		RedirectURI:     domain.DefaultRedirectURI(app.Cfg.PublicBaseURL, tid),
		Scope:           domain.DefaultTenantGitLabScope,
		Active:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	sp := domain.TenantServiceProvider(tid)
	pc, err := app.resolveProviderByServiceProvider(sp)
	if err != nil {
		t.Fatal(err)
	}
	if pc == nil {
		t.Fatal("expected provider config from DB")
	}
	if pc.ClientID != "app-id" {
		t.Fatalf("client_id=%s", pc.ClientID)
	}
	if pc.ClientSecret != "super-secret" {
		t.Fatalf("secret not decrypted")
	}
	if pc.Website != "https://git.corp.example" {
		t.Fatalf("website=%s", pc.Website)
	}
	if pc.ProviderKey != domain.ProviderKeyForCompany(tid) {
		t.Fatalf("provider_key=%s", pc.ProviderKey)
	}

	routeCtx, reason := app.resolveGitLabAuthorizeContext(sp, "")
	if routeCtx == nil {
		t.Fatalf("resolve context failed: %s", reason)
	}
	if routeCtx["client_id"] != "app-id" {
		t.Fatalf("route client_id=%v", routeCtx["client_id"])
	}
	if routeCtx["origin"] != "https://git.corp.example" {
		t.Fatalf("origin=%v", routeCtx["origin"])
	}
	wantRedirect := domain.DefaultRedirectURI(app.Cfg.PublicBaseURL, tid)
	if routeCtx["redirect_uri"] != wantRedirect {
		t.Fatalf("authorize redirect_uri=%v want %v", routeCtx["redirect_uri"], wantRedirect)
	}
}

func TestInternalTenantGitlabOAuthConnection(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	tid := "co-internal"
	cipher, _ := app.Fernet.Encrypt("x")
	_, _ = app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID: tid, BaseURL: "https://gl.test", ClientID: "c",
		ClientSecretEnc: cipher, RedirectURI: domain.DefaultRedirectURI(app.Cfg.PublicBaseURL, tid),
		Scope: domain.DefaultTenantGitLabScope, Active: true, UpdatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/gitlab-tenant-connection/?company_id="+tid, nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["configured"] != true {
		t.Fatalf("body=%v", body)
	}
	if _, ok := body["client_secret"]; ok {
		t.Fatal("internal GET must not leak secret")
	}
}

func TestGetTenantGitLabConnectionByHostPathA(t *testing.T) {
	app := testApp(t)
	tid := "877397588196749312"
	cipher, err := app.Fernet.Encrypt("sekrit")
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID: tid, BaseURL: "http://115.29.110.74", ClientID: "path-a-client",
		ClientSecretEnc: cipher, RedirectURI: domain.DefaultRedirectURI(app.Cfg.PublicBaseURL, tid),
		Scope: domain.DefaultTenantGitLabScope, Active: true, UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	row, err := app.DB.GetTenantGitLabConnectionByHost("115.29.110.74")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil || row.CompanyID != tid {
		t.Fatalf("by host=%v", row)
	}
	pc, err := app.Cfg.ResolveProviderByGitsiteWithDB(app.DB, app.Fernet, "115.29.110.74")
	if err != nil {
		t.Fatal(err)
	}
	if pc == nil || pc.ProviderKey != "gitlab:tenant-"+tid {
		t.Fatalf("gitsite resolve=%v", pc)
	}
	if other, err := app.DB.GetTenantGitLabConnectionByHost("gitlab.daydaymoney.com"); err != nil || other != nil {
		t.Fatalf("unrelated host must miss: %v %v", other, err)
	}
}
