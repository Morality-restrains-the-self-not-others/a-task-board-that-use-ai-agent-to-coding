package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func TestCatalogCompanyIDFromQueryAndHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/git-oauth/providers/?company_id=co-q", nil)
	if got := catalogCompanyID(req); got != "co-q" {
		t.Fatalf("query company_id=%q", got)
	}
	reqH := httptest.NewRequest(http.MethodGet, "/api/git-oauth/providers/", nil)
	reqH.Header.Set("X-Tenant-Id", "co-h")
	if got := catalogCompanyID(reqH); got != "co-h" {
		t.Fatalf("X-Tenant-Id=%q", got)
	}
}

func TestGitOauthProvidersCatalogIncludesTenantPathA(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

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

	req := httptest.NewRequest(http.MethodGet, "/api/git-oauth/providers/?company_id="+tid, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Providers []map[string]any `json:"providers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range body.Providers {
		if p["provider_key"] == "gitlab:tenant-"+tid {
			found = true
			if p["website"] != "http://115.29.110.74" {
				t.Fatalf("website=%v", p["website"])
			}
			if p["service_provider"] != "tenant-"+tid {
				t.Fatalf("service_provider=%v", p["service_provider"])
			}
			if _, ok := p["client_secret"]; ok {
				t.Fatal("catalog must not leak client_secret")
			}
		}
	}
	if !found {
		t.Fatalf("expected gitlab:tenant-%s in catalog: %v", tid, body.Providers)
	}

	reqBare := httptest.NewRequest(http.MethodGet, "/api/git-oauth/providers/", nil)
	rrBare := httptest.NewRecorder()
	mux.ServeHTTP(rrBare, reqBare)
	var bare struct {
		Providers []map[string]any `json:"providers"`
	}
	_ = json.Unmarshal(rrBare.Body.Bytes(), &bare)
	for _, p := range bare.Providers {
		if p["provider_key"] == "gitlab:tenant-"+tid {
			t.Fatal("catalog without company_id must not leak tenant Path A")
		}
	}
}
