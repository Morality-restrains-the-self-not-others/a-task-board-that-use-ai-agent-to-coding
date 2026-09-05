package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectProjectGitRepoURLsFromStringSlice(t *testing.T) {
	detail := map[string]interface{}{
		"git_repos": []string{"http://example.com/group/somanyad.git", "  ", ""},
	}
	got := collectProjectGitRepoURLs(detail)
	if len(got) != 1 || got[0] != "http://example.com/group/somanyad.git" {
		t.Fatalf("collectProjectGitRepoURLs() = %#v, want single trimmed repo URL", got)
	}
}

func TestCollectProjectGitRepoURLsFromInterfaceSlice(t *testing.T) {
	detail := map[string]interface{}{
		"git_repos": []interface{}{"https://github.com/o/r"},
	}
	got := collectProjectGitRepoURLs(detail)
	if len(got) != 1 || got[0] != "https://github.com/o/r" {
		t.Fatalf("collectProjectGitRepoURLs() = %#v", got)
	}
}

func TestEnrichProjectGitReposStatusGoNative(t *testing.T) {
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]interface{}{"error": "not bound"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	detail := map[string]interface{}{
		"git_repos": []string{"https://github.com/o/private.git"},
	}
	enrichProjectGitReposStatus(detail, "1001", "")
	raw, ok := detail["git_repos_status"].([]map[string]interface{})
	if !ok || len(raw) != 1 {
		t.Fatalf("git_repos_status=%#v", detail["git_repos_status"])
	}
	if raw[0]["token_status"] != tokenStatusNotBound {
		t.Fatalf("token_status=%#v want not_bound", raw[0]["token_status"])
	}
	if raw[0]["oauth_provider"] != "github" {
		t.Fatalf("oauth_provider=%#v", raw[0]["oauth_provider"])
	}
	if raw[0]["repo_url"] != "https://github.com/o/private.git" {
		t.Fatalf("repo_url=%#v", raw[0]["repo_url"])
	}
}

func TestRepoAccessCheckUsesPrimaryGitRepoFromProjectDetail(t *testing.T) {
	setupTestDB(t)

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]interface{}{"error": "not bound"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	body := `{"name":"把 somanyad 运行起来","git_repos":["https://github.com/org/somanyad-emailD.git"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatalf("missing project id: %#v", created)
	}

	reqCheck := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/repo-access-check/tenant_id/t1/", nil)
	reqCheck.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCheck.Header.Set("X-Auth-User-Id", "1001")
	recCheck := httptest.NewRecorder()
	handleRepoAccessCheck(recCheck, reqCheck, "t1", pid)

	if recCheck.Code != http.StatusOK {
		t.Fatalf("repo-access-check: expected 200, got %d: %s", recCheck.Code, recCheck.Body.String())
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(recCheck.Body).Decode(&resp)
	if resp["access_status"] != accessStatusNeedsAuth {
		t.Fatalf("access_status = %#v, want needs_auth", resp["access_status"])
	}
}
