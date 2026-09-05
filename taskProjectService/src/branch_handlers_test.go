package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseGitLabProjectParts(t *testing.T) {
	parts, err := parseGitLabProjectParts("http://127.0.0.1:8012/example-user/task2app.git")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := "http://127.0.0.1:8012/api/v4/projects/example-user%2Ftask2app"
	if parts.APIBase != want {
		t.Errorf("APIBase = %q, want %q", parts.APIBase, want)
	}
}

func TestListGitHubBranchesAnonymousWhenOAuthRefreshFails(t *testing.T) {
	initProviderTestConfig(t)
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("anonymous branch list should not send Authorization")
		}
		if !strings.Contains(r.URL.Path, "/branches") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"master"},{"name":"dev"}]`))
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

	payload := listGitHubBranches("1001", "https://github.com/ruandao/somanyad", gitoauthSrv.URL)
	if payload.Error != "" {
		t.Fatalf("error=%q want empty for public repo anonymous list", payload.Error)
	}
	if len(payload.Branches) != 2 || payload.Branches[0] != "master" {
		t.Fatalf("branches=%v", payload.Branches)
	}
}

func TestListProjectRepoBranchesGitLabNeedsAuth(t *testing.T) {
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
			Provider: "gitlab", ServiceProvider: "tencent-gitlab",
			ProviderKey: "gitlab:tencent-gitlab", Host: "1.117.67.121", Netloc: "1.117.67.121:8012",
			GitoauthBase: gitoauthSrv.URL,
		}},
	}

	payload := listProjectRepoBranches("1001", "http://1.117.67.121:8012/group-a/demo-repo", "", "")
	if len(payload.Branches) != 0 {
		t.Errorf("expected empty branches, got %v", payload.Branches)
	}
	if !strings.Contains(payload.Error, "未检测到可用授权") {
		t.Errorf("expected auth hint, got %q", payload.Error)
	}
	if strings.Contains(payload.Error, "generic Git repositories") {
		t.Errorf("unexpected generic git error: %q", payload.Error)
	}
	if payload.Gitlab["resolve_error"] != "http 502" {
		t.Errorf("resolve_error = %q", payload.Gitlab["resolve_error"])
	}
}

func TestListProjectRepoBranchesPathAUsesTenantProviderKey(t *testing.T) {
	initProviderTestConfig(t)
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); !strings.Contains(got, "Bearer path-a-token") {
			t.Errorf("Authorization=%q, want Path A bearer", got)
		}
		_, _ = w.Write([]byte(`[{"name":"main"}]`))
	}))
	defer gitlabSrv.Close()

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OPT-20260827-036: 换票统一走 gitsite 路径，按 repo host 寻址，不再带 provider_key。
		if !strings.Contains(r.URL.Path, "/api/internal/gitsite/") || !strings.Contains(r.URL.Path, "/oauth/access-for-user/") {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, has := body["provider_key"]; has {
			t.Fatalf("gitsite body must omit provider_key, got %v", body["provider_key"])
		}
		writeJSON(w, 200, map[string]interface{}{"access_token": "path-a-token"})
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	oldLookup := lookupTenantGitLabConn
	oldResolver := providerResolver
	t.Cleanup(func() {
		lookupTenantGitLabConn = oldLookup
		providerResolver = oldResolver
	})
	providerResolver = &ProviderResolver{}
	pathAOrigin := strings.TrimSuffix(gitlabSrv.URL, "/")
	lookupTenantGitLabConn = func(tenantID string, trace map[string]string) *tenantGitLabConn {
		return &tenantGitLabConn{
			Configured:  true,
			Active:      true,
			ProviderKey: "gitlab:tenant-877397588196749312",
			BaseURL:     pathAOrigin,
		}
	}

	repoURL := pathAOrigin + "/example-user/somanyad.git"
	payload := listProjectRepoBranches("1001", repoURL, "", "877397588196749312")
	if payload.Error != "" {
		t.Fatalf("error=%q", payload.Error)
	}
	if len(payload.Branches) != 1 || payload.Branches[0] != "main" {
		t.Fatalf("branches=%v", payload.Branches)
	}
}

func TestListProjectRepoBranchesGitLabSessionCookie(t *testing.T) {
	initProviderTestConfig(t)
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); !strings.Contains(got, "_gitlab_session=abc123") {
			t.Errorf("cookie = %q", got)
		}
		if !strings.Contains(r.URL.Path, "/repository/branches") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"main"},{"name":"dev"}]`))
	}))
	defer gitlabSrv.Close()

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not bound"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{}
	repoURL := strings.TrimSuffix(gitlabSrv.URL, "/") + "/group-a/demo-repo.git"

	payload := listProjectRepoBranches("1001", repoURL, "abc123", "")
	if payload.Gitlab != nil {
		t.Fatalf("expected gitlab meta cleared on success, got %+v", payload.Gitlab)
	}
	if len(payload.Branches) != 2 || payload.Branches[0] != "main" {
		t.Fatalf("branches = %v", payload.Branches)
	}
}

func TestHandleProjectBranchesRoute(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"Branch Preview Project"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	recCreate := httptest.NewRecorder()
	handleCreateProject(recCreate, reqCreate)
	var created map[string]interface{}
	_ = json.NewDecoder(recCreate.Body).Decode(&created)
	pid := created["id"].(string)

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"http 502"}`))
	}))
	defer gitoauthSrv.Close()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider: "gitlab", ServiceProvider: "default",
			ProviderKey: "gitlab:default", Host: "127.0.0.1", Netloc: "127.0.0.1:8012",
		}},
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/branches/tenant_id/t1/?repo_url="+
		"http%3A%2F%2F127.0.0.1%3A8012%2Fexample-user%2Ftask2app.git", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "850256676127797248")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rawBody := rec.Body.String()
	if rawBody == "" {
		t.Fatal("empty body")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(rawBody), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, rawBody)
	}
	if _, ok := payload["branches"]; !ok {
		t.Fatalf("missing branches: %v", payload)
	}
	if errText, _ := payload["error"].(string); !strings.Contains(errText, "未检测到可用授权") {
		t.Errorf("expected auth hint, got %q", errText)
	}
}

func TestHandleProjectBranchesNot501(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	body := `{"name":"No501"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	recCreate := httptest.NewRecorder()
	handleCreateProject(recCreate, reqCreate)
	var created map[string]interface{}
	_ = json.NewDecoder(recCreate.Body).Decode(&created)
	pid := created["id"].(string)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/branches/tenant_id/t1/?repo_url=http%3A%2F%2Fexample.com%2Fa.git", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotImplemented {
		t.Fatalf("branches endpoint still returns 501")
	}
}
