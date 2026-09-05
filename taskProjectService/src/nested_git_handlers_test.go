package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListNestedGitReposNeedsAuth(t *testing.T) {
	initProviderTestConfig(t)
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"detail":"http 502"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "gitlab", ServiceProvider: "default",
			ProviderKey: "gitlab:default", Host: "gitlab.daydaymoney.com", Netloc: "gitlab.daydaymoney.com",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	payload := listNestedGitRepos("1001", "https://gitlab.daydaymoney.com/example-user/ram-work.git", "", "")
	if len(payload.NestedRepos) != 0 {
		t.Errorf("expected empty, got %#v", payload.NestedRepos)
	}
	// 502 from gitoauth is a service outage, not "never bound".
	if !strings.Contains(payload.Error, "授权服务暂时不可用") {
		t.Errorf("error=%q", payload.Error)
	}
}

// Public GitHub parent + OAuth refresh failure must not surface "未检测到可用授权".
// Anonymous Contents API (api.github.com) can still discover an empty nested list.
func TestListNestedGitReposGitHubAnonymousWhenOAuthRefreshFails(t *testing.T) {
	initProviderTestConfig(t)
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/contents/") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Not Found"}`))
			return
		}
		// Repo metadata visible without Authorization (public).
		if r.Header.Get("Authorization") != "" {
			t.Errorf("anonymous path should not send Authorization, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"default_branch":"master","full_name":"ruandao/somanyad"}`))
	}))
	defer githubSrv.Close()
	oldGH := githubAPIBase
	githubAPIBase = githubSrv.URL
	t.Cleanup(func() { githubAPIBase = oldGH })

	oauthHits := 0
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oauthHits++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"detail":"Post \"https://github.com/login/oauth/access_token\": http2: timeout awaiting response headers"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "github", ServiceProvider: "github-official-daydaymoney",
			ProviderKey: "github:github-official-daydaymoney", Host: "github.com", Netloc: "github.com",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	payload := listNestedGitRepos("874599469415428096", "https://github.com/ruandao/somanyad", "", "")
	if payload.Error != "" {
		t.Fatalf("error=%q want empty (public parent, no .gitmodules)", payload.Error)
	}
	if len(payload.NestedRepos) != 0 {
		t.Fatalf("nested=%#v want empty", payload.NestedRepos)
	}
	if oauthHits != 0 {
		t.Fatalf("public GitHub fast path must skip OAuth refresh, oauthHits=%d", oauthHits)
	}
}

// Private/invisible GitHub parent + refresh timeout → timeout copy, not "请先绑定".
func TestListNestedGitReposGitHubPrivateSurfacesRefreshTimeout(t *testing.T) {
	initProviderTestConfig(t)
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer githubSrv.Close()
	oldGH := githubAPIBase
	githubAPIBase = githubSrv.URL
	t.Cleanup(func() { githubAPIBase = oldGH })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"detail":"http2: timeout awaiting response headers"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "github", ServiceProvider: "github-official-daydaymoney",
			ProviderKey: "github:github-official-daydaymoney", Host: "github.com", Netloc: "github.com",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	payload := listNestedGitRepos("1001", "https://github.com/task2money/private-repo", "", "")
	if strings.Contains(payload.Error, "未检测到可用授权") {
		t.Fatalf("error=%q must not claim unbound when refresh timed out", payload.Error)
	}
	if !strings.Contains(payload.Error, "授权刷新超时") {
		t.Fatalf("error=%q want refresh-timeout hint", payload.Error)
	}
}

func TestListNestedGitReposFromMockGitLab(t *testing.T) {
	initProviderTestConfig(t)
	gitmodules := `[submodule "task2app"]
	path = task2app
	url = ../task2app.git
[submodule "docs"]
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

	host := strings.TrimPrefix(gitlabSrv.URL, "http://")
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "gitlab", ServiceProvider: "default",
			ProviderKey: "gitlab:default", Host: strings.Split(host, ":")[0], Netloc: host,
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	repoURL := gitlabSrv.URL + "/example-user/ram-work.git"
	payload := listNestedGitRepos("1001", repoURL, "", "")
	if payload.Error != "" {
		t.Fatalf("error=%q", payload.Error)
	}
	if len(payload.NestedRepos) != 2 {
		t.Fatalf("nested=%#v", payload.NestedRepos)
	}
	if payload.NestedRepos[0].Path != "docs" || payload.NestedRepos[1].Path != "task2app" {
		t.Fatalf("order/paths %#v", payload.NestedRepos)
	}
	if payload.NestedRepos[1].Source != nestedSourceGitmodules {
		t.Errorf("source=%q want gitmodules", payload.NestedRepos[1].Source)
	}
	wantURL := gitlabSrv.URL + "/example-user/task2app.git"
	if payload.NestedRepos[1].URL != wantURL {
		t.Errorf("task2app url=%q want %q", payload.NestedRepos[1].URL, wantURL)
	}
}

func TestListNestedGitReposNoGitmodulesNoGitignoreFallback(t *testing.T) {
	initProviderTestConfig(t)
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repository/files/.gitignore/raw") {
			_, _ = w.Write([]byte(`# Nested git repos (tracked in their own repositories)
task2app/
docs/
`))
			return
		}
		// Parent project is visible (default_branch / accessibility probe) but .gitmodules missing.
		if strings.Contains(r.URL.Path, "/api/v4/projects/") && !strings.Contains(r.URL.Path, "/repository/") {
			_, _ = w.Write([]byte(`{"default_branch":"main"}`))
			return
		}
		// .gitmodules missing
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

	payload := listNestedGitRepos("1001", gitlabSrv.URL+"/ljy/ram-work.git", "", "")
	if payload.Error != "" {
		t.Fatalf("error=%q", payload.Error)
	}
	if len(payload.NestedRepos) != 0 {
		t.Fatalf("must not fall back to .gitignore, got %#v", payload.NestedRepos)
	}
}

func TestListNestedGitReposParentInaccessibleNotEmpty(t *testing.T) {
	initProviderTestConfig(t)
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Private repo / no App permissions → GitHub returns 404 for both repo and contents.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer githubSrv.Close()
	oldGH := githubAPIBase
	githubAPIBase = githubSrv.URL
	t.Cleanup(func() { githubAPIBase = oldGH })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "github", ServiceProvider: "github-official-daydaymoney",
			ProviderKey: "github:github-official-daydaymoney", Host: "github.com", Netloc: "github.com",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	payload := listNestedGitRepos("1001", "https://github.com/task2money/ram-work", "", "")
	if len(payload.NestedRepos) != 0 {
		t.Fatalf("expected empty nested, got %#v", payload.NestedRepos)
	}
	if !strings.Contains(payload.Error, "无法访问父仓库") {
		t.Fatalf("error=%q want parent-inaccessible", payload.Error)
	}
}

// github.com/*.git must hit GitHub Contents API (not GitLab), even though the URL ends with ".git".
func TestListNestedGitReposGitHubDotGitPrefersGitHubAPI(t *testing.T) {
	initProviderTestConfig(t)
	ghHits := 0
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ghHits++
		if strings.Contains(r.URL.Path, "/contents/") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Not Found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"default_branch":"main","full_name":"ruandao/somanyad"}`))
	}))
	defer githubSrv.Close()
	oldGH := githubAPIBase
	githubAPIBase = githubSrv.URL
	t.Cleanup(func() { githubAPIBase = oldGH })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"ghu_test_tok"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "github", ServiceProvider: "github-official-daydaymoney",
			ProviderKey: "github:github-official-daydaymoney", Host: "github.com", Netloc: "github.com",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	for _, url := range []string{
		"https://github.com/ruandao/somanyad.git",
		"git@github.com:ruandao/somanyad.git",
	} {
		ghHits = 0
		payload := listNestedGitRepos("1001", url, "", "")
		if strings.Contains(payload.Error, "GitLab") {
			t.Fatalf("url=%q error=%q must not mention GitLab", url, payload.Error)
		}
		if payload.Error != "" {
			t.Fatalf("url=%q error=%q want empty (parent visible, no .gitmodules)", url, payload.Error)
		}
		if ghHits == 0 {
			t.Fatalf("url=%q expected GitHub API hits", url)
		}
	}
}

func TestHandleProjectNestedGitRepos(t *testing.T) {
	setupTestDB(t)
	body := `{"name":"NestedDemo","git_repos":["https://gitlab.daydaymoney.com/example-user/ram-work.git"]}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	recCreate := httptest.NewRecorder()
	handleCreateProject(recCreate, reqCreate)
	var created map[string]interface{}
	_ = json.NewDecoder(recCreate.Body).Decode(&created)
	pid := created["id"].(string)

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not bound"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "gitlab", ServiceProvider: "default",
			ProviderKey: "gitlab:default", Host: "gitlab.daydaymoney.com", Netloc: "gitlab.daydaymoney.com",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/nested-git-repos/tenant_id/t1/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "850256676127797248")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload nestedGitReposPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload.Error, "未检测到可用授权") {
		t.Errorf("error=%q", payload.Error)
	}

	// cross-tenant → 404 (auth user must be provided for tenant membership check)
	req2 := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/nested-git-repos/tenant_id/other/", nil)
	req2.Header.Set("X-Auth-Tenant-Id", "other")
	req2.Header.Set("X-Auth-User-Id", "850256676127797248")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant code=%d", rec2.Code)
	}
}

func TestHandleProjectNestedGitReposNoRepo(t *testing.T) {
	setupTestDB(t)
	body := `{"name":"NoRepo"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	recCreate := httptest.NewRecorder()
	handleCreateProject(recCreate, reqCreate)
	var created map[string]interface{}
	_ = json.NewDecoder(recCreate.Body).Decode(&created)
	pid := created["id"].(string)

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/nested-git-repos/tenant_id/t1/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "850256676127797248")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

// Path A GitLab (tenant connection) must resolve OAuth when yaml host does not match.
// Auto-run used to omit tenant_id → empty providerKey → 「未检测到可用授权」 despite a bound token.
func TestListNestedGitReposTenantPathAWhenYamlUnmatched(t *testing.T) {
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

	var gotPath string
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"access_token":"tok-path-a"}`))
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
		if tenantID != "t-path-a" {
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

	repoURL := gitlabSrv.URL + "/ljy/ram-work.git"
	withoutTenant := listNestedGitRepos("1001", repoURL, "", "")
	if !strings.Contains(withoutTenant.Error, "未检测到可用授权") {
		t.Fatalf("without tenant error=%q want 未检测到可用授权", withoutTenant.Error)
	}

	payload := listNestedGitRepos("1001", repoURL, "", "t-path-a")
	if payload.Error != "" {
		t.Fatalf("with tenant error=%q", payload.Error)
	}
	if len(payload.NestedRepos) != 1 || payload.NestedRepos[0].Path != "task2app" {
		t.Fatalf("payload=%#v", payload)
	}
	if !strings.Contains(gotPath, "/api/internal/gitsite/") {
		t.Fatalf("gitoauth path=%q want gitsite path", gotPath)
	}
}
