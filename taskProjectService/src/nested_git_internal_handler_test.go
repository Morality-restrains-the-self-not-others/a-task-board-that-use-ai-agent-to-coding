package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleInternalNestedGitRepos(t *testing.T) {
	initProviderTestConfig(t)
	gitmodules := `[submodule "task2app"]
	path = task2app
	url = ../task2app.git
`
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repository/files/.gitmodules/raw") {
			_, _ = w.Write([]byte(gitmodules))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer gitlabSrv.Close()

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	host := strings.TrimPrefix(gitlabSrv.URL, "http://")
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "gitlab", ServiceProvider: "default",
			ProviderKey: "gitlab:default", Host: strings.Split(host, ":")[0], Netloc: host,
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	repoURL := gitlabSrv.URL + "/ljy/ram-work.git"
	req := httptest.NewRequest(http.MethodGet, "/api/internal/nested-git-repos/?repo_url="+
		strings.ReplaceAll(repoURL, ":", "%3A")+"&user_id=1001", nil)
	// simpler: use QueryEscape via URL values in test
	req = httptest.NewRequest(http.MethodGet, "/api/internal/nested-git-repos/", nil)
	q := req.URL.Query()
	q.Set("repo_url", repoURL)
	q.Set("user_id", "1001")
	req.URL.RawQuery = q.Encode()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload nestedGitReposPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.NestedRepos) != 1 || payload.NestedRepos[0].Path != "task2app" {
		t.Fatalf("payload=%#v", payload)
	}
}

func TestHandleInternalNestedGitReposUsesTenantIDQuery(t *testing.T) {
	initProviderTestConfig(t)
	gitmodules := `[submodule "docs"]
	path = docs
	url = ../docs.git
`
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repository/files/.gitmodules/raw") {
			_, _ = w.Write([]byte(gitmodules))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer gitlabSrv.Close()

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "gitlab", ServiceProvider: "default",
			ProviderKey: "gitlab:default", Host: "gitlab.unrelated.example", Netloc: "gitlab.unrelated.example",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}
	prevLookup := lookupTenantGitLabConn
	lookupTenantGitLabConn = func(tenantID string, _ map[string]string) *tenantGitLabConn {
		if tenantID != "co-1" {
			t.Errorf("tenantID=%q", tenantID)
			return nil
		}
		return &tenantGitLabConn{
			Configured:  true,
			Active:      true,
			ProviderKey: "gitlab:tenant-path-a",
			BaseURL:     gitlabSrv.URL,
		}
	}
	t.Cleanup(func() { lookupTenantGitLabConn = prevLookup })

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/nested-git-repos/", nil)
	q := req.URL.Query()
	q.Set("repo_url", gitlabSrv.URL+"/ljy/ram-work.git")
	q.Set("user_id", "1001")
	q.Set("tenant_id", "co-1")
	req.URL.RawQuery = q.Encode()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload nestedGitReposPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error != "" {
		t.Fatalf("error=%q", payload.Error)
	}
	if len(payload.NestedRepos) != 1 || payload.NestedRepos[0].Path != "docs" {
		t.Fatalf("payload=%#v", payload)
	}
}
