package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Regression: SPA opens GitLab sync without gitlab_host; provider fallback must
// still exchange OAuth via WebsiteOrigin host (not "missing repo host").
func TestHandleGitlabRemoteReposEmptyHostUsesWebsiteOrigin(t *testing.T) {
	initProviderTestConfig(t)

	var gotPath string
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"access_token":"tok-sync-1"}`))
	}))
	defer gitoauthSrv.Close()

	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api/v4/user"):
			_, _ = w.Write([]byte(`{"username":"sync-user"}`))
		case strings.Contains(r.URL.Path, "/api/v4/projects"):
			_, _ = w.Write([]byte(`[{"id":1,"name":"demo","path_with_namespace":"g/demo","http_url_to_repo":"https://gitlab.daydaymoney.com/g/demo.git","description":"","last_activity_at":"2026-09-01T00:00:00Z"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer gitlabSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider:        "gitlab",
			ServiceProvider: "default",
			ProviderKey:     "gitlab:default",
			Host:            "gitlab.daydaymoney.com",
			Netloc:          "gitlab.daydaymoney.com",
			WebsiteOrigin:   gitlabSrv.URL,
			GitoauthBase:    gitoauthSrv.URL,
		}},
	}

	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/gitlab-remote-repos/tenant_id/t1/", nil)
	req.Header.Set("X-Auth-User-Id", "1001")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleGitlabRemoteRepos(rec, req, "t1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "missing repo host") {
		t.Fatalf("must not surface missing repo host: %s", rec.Body.String())
	}
	// gitsite path uses host from WebsiteOrigin (test server host:port)
	host, _ := extractHostNetloc(gitlabSrv.URL)
	wantPrefix := "/api/internal/gitsite/" + host + "/oauth/access-for-user"
	if !strings.HasPrefix(gotPath, wantPrefix) && !strings.Contains(gotPath, host) {
		t.Fatalf("oauth path=%q want host %q from WebsiteOrigin", gotPath, host)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["oauth_bound"] != true {
		t.Fatalf("oauth_bound=%v want true", body["oauth_bound"])
	}
	if body["error"] != nil && body["error"] != "" {
		t.Fatalf("error=%v want empty", body["error"])
	}
}

func TestHandleGitlabRemoteReposExplicitHostStillUsed(t *testing.T) {
	initProviderTestConfig(t)

	var gotPath string
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not_found"}`))
	}))
	defer gitoauthSrv.Close()

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider:        "gitlab",
			ServiceProvider: "default",
			ProviderKey:     "gitlab:default",
			Host:            "gitlab.daydaymoney.com",
			Netloc:          "gitlab.daydaymoney.com",
			WebsiteOrigin:   "https://gitlab.daydaymoney.com",
			GitoauthBase:    gitoauthSrv.URL,
		}},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/gitlab-remote-repos/tenant_id/t1/?gitlab_host=gitlab.daydaymoney.com",
		nil,
	)
	req.Header.Set("X-Auth-User-Id", "1001")
	rec := httptest.NewRecorder()
	handleGitlabRemoteRepos(rec, req, "t1")

	if !strings.Contains(gotPath, "gitlab.daydaymoney.com") {
		t.Fatalf("oauth path=%q want explicit gitlab.daydaymoney.com", gotPath)
	}
	if strings.Contains(rec.Body.String(), "missing repo host") {
		t.Fatalf("must not be missing repo host: %s", rec.Body.String())
	}
}
