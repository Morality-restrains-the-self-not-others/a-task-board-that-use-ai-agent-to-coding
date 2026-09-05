package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleValidateGitReposGoNativeTokenOnly(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				Provider:        "gitlab",
				ServiceProvider: "default",
				ProviderKey:     "gitlab:default",
				Host:            "gitlab.daydaymoney.com",
				Netloc:          "gitlab.daydaymoney.com",
				WebsiteOrigin:   "https://gitlab.daydaymoney.com",
				GitoauthBase:    "",
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "gitlab-tenant-connection") {
			writeJSON(w, 200, map[string]interface{}{"configured": false})
			return
		}
		if !strings.Contains(r.URL.Path, "/api/internal/gitsite/") || !strings.Contains(r.URL.Path, "/oauth/access-for-user/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, has := body["provider_key"]; has {
			t.Fatalf("gitsite body must omit provider_key, got %v", body["provider_key"])
		}
		// Simulate bound for DaydaymoneyGrafana user flow: return token for any request
		writeJSON(w, 200, map[string]interface{}{"access_token": "tok-xyz"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })
	providerResolver.entries[0].GitoauthBase = gitoauthSrv.URL

	body := `{"urls":["https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git","https://gitlab.daydaymoney.com/g/docs.git","https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git"],"probe_access":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/validate-git-repos/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Id", "1001")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleValidateGitRepos(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	results, ok := resp["results"].([]interface{})
	if !ok || len(results) != 2 {
		t.Fatalf("results = %#v, want 2 deduped", resp["results"])
	}
	first, _ := results[0].(map[string]interface{})
	if first["token_status"] != tokenStatusAvailable {
		t.Fatalf("results[0].token_status = %#v", first["token_status"])
	}
	if _, has := first["is_accessible"]; !has {
		t.Fatalf("expected is_accessible field in batch result")
	}
}

func TestHandleValidateGitReposProbeAccessUsesRemote(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/private" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer tok" {
			t.Fatalf("Authorization=%q", auth)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer githubAPI.Close()

	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "tok"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	body := `{"urls":["https://github.com/o/private.git"],"probe_access":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/validate-git-repos/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Id", "1001")
	rec := httptest.NewRecorder()
	handleValidateGitRepos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	results, _ := resp["results"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("results=%#v", resp["results"])
	}
	row, _ := results[0].(map[string]interface{})
	if row["is_accessible"] != false {
		t.Fatalf("is_accessible=%#v want false", row["is_accessible"])
	}
	// Provider token must not keep the row "已授权" when this repo is 404/inaccessible
	// (e.g. project git URL switched to another person's private repo).
	if row["token_status"] != tokenStatusTokenError {
		t.Fatalf("token_status=%#v want token_error", row["token_status"])
	}
	msg, _ := row["message"].(string)
	if !strings.Contains(msg, "not found") {
		t.Fatalf("message=%q", msg)
	}
}

func TestHandleValidateGitRepoGoNativeEmptyURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/projects/validate-git-repo/", nil)
	req.Header.Set("X-Auth-User-Id", "1001")
	rec := httptest.NewRecorder()
	handleValidateGitRepo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleValidateGitReposRejectsEmptyAndWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/projects/validate-git-repos/", nil)
	rec := httptest.NewRecorder()
	handleValidateGitRepos(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET expected 405, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/validate-git-repos/", strings.NewReader(`{"urls":[]}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleValidateGitRepos(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("empty urls expected 400, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestResolveTokenStatus(t *testing.T) {
	if resolveTokenStatus("abc", "") != tokenStatusAvailable {
		t.Fatal("token available")
	}
	if resolveTokenStatus("", "boom") != tokenStatusTokenError {
		t.Fatal("token error")
	}
	if resolveTokenStatus("", "") != tokenStatusNotBound {
		t.Fatal("not bound")
	}
}
