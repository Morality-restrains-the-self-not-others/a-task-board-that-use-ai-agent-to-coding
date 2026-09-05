package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

func resetAuthContextServiceURLs(t *testing.T) {
	t.Helper()
	prevTask := cfg.TaskServiceURL
	prevProj := cfg.ProjectServiceURL
	prevOauth := cfg.GitOauthBaseURL
	cfg.TaskServiceURL = ""
	cfg.ProjectServiceURL = ""
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.ProjectServiceURL = prevProj
		cfg.GitOauthBaseURL = prevOauth
	})
}

func TestFetchGitOauthCredentialSummary_UsesCanonicalGitOauthPath(t *testing.T) {
	var gotPath string
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"connected": true})
	}))
	defer oauth.Close()
	prev := cfg.GitOauthBaseURL
	cfg.GitOauthBaseURL = oauth.URL
	t.Cleanup(func() { cfg.GitOauthBaseURL = prev })

	ok, errDetail := fetchGitOauthCredentialSummary("877397583960502272", "gitlab", "gitlab:tencent-sh-1")
	if errDetail != "" {
		t.Fatalf("errDetail=%q", errDetail)
	}
	if !ok {
		t.Fatal("want connected")
	}
	if gotPath != "/api/internal/git-oauth/gitlab-credential-summary/" {
		t.Fatalf("path=%q (legacy summary-for-user 404s on taskGitOauth)", gotPath)
	}
}

func TestAuthContext_EmptyRepos(t *testing.T) {
	cfg.InternalSecret = ""
	resetAuthContextServiceURLs(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/layer-git-push/auth-context?tenant_id=1&workspace_id=w1&task_id=task1&user_id=42", nil)
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushAuthContext(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["has_github_repo_projects"] != false {
		t.Fatalf("has_github=%v", out["has_github_repo_projects"])
	}
	if out["git_oauth_ready_for_push"] != true {
		t.Fatalf("ready=%v", out["git_oauth_ready_for_push"])
	}
	repos, _ := out["repos"].([]any)
	if len(repos) != 0 {
		t.Fatalf("repos=%v", repos)
	}
	if out["message"] != "当前工作空间未发现 GitHub/GitLab 仓库项目" {
		t.Fatalf("message=%v", out["message"])
	}
}

func TestAuthContext_WithHTTPReposRequiresOAuth(t *testing.T) {
	cfg.InternalSecret = ""
	resetAuthContextServiceURLs(t)

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "task1",
			"projects": []map[string]any{
				{"id": "p1", "project_name": "Demo", "stored_repo_address": "https://github.com/Acme/Demo.git"},
			},
		})
	}))
	defer taskSrv.Close()
	cfg.TaskServiceURL = taskSrv.URL

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/layer-git-push/auth-context?tenant_id=1&workspace_id=w1&task_id=task1&user_id=42", nil)
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushAuthContext(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["has_github_repo_projects"] != true {
		t.Fatalf("has_github=%v body=%s", out["has_github_repo_projects"], rec.Body.String())
	}
	if out["git_oauth_ready_for_push"] != false {
		t.Fatalf("ready should be false without oauth, got %v", out["git_oauth_ready_for_push"])
	}
}

func TestAuthContext_MissingParams(t *testing.T) {
	cfg.InternalSecret = ""
	req := httptest.NewRequest(http.MethodGet, "/api/internal/layer-git-push/auth-context?tenant_id=1", nil)
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushAuthContext(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestAuthContext_ForbiddenWithoutSecret(t *testing.T) {
	prev := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prev })
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/layer-git-push/auth-context?tenant_id=1&workspace_id=w1&task_id=t&user_id=1", nil)
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushAuthContext(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestGithubRepoSlugFromURLStrict(t *testing.T) {
	if got := githubRepoSlugFromURLStrict("https://github.com/Acme/Demo.git"); got != "acme/demo" {
		t.Fatalf("got=%q", got)
	}
	if got := githubRepoSlugFromURLStrict("git@github.com:Acme/Demo.git"); got != "acme/demo" {
		t.Fatalf("scp got=%q", got)
	}
	if got := githubRepoSlugFromURLStrict("https://gitlab.com/a/b.git"); got != "" {
		t.Fatalf("gitlab should be empty, got=%q", got)
	}
}

func TestAuthContext_WithOAuthConnected(t *testing.T) {
	cfg.InternalSecret = ""
	resetAuthContextServiceURLs(t)

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "task1",
			"projects": []map[string]any{
				{"id": "p1", "project_name": "Demo", "stored_repo_address": "https://github.com/Acme/Demo.git"},
			},
		})
	}))
	defer taskSrv.Close()
	cfg.TaskServiceURL = taskSrv.URL

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = raw
		_ = json.NewEncoder(w).Encode(map[string]any{"connected": true})
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/layer-git-push/auth-context?tenant_id=1&workspace_id=w1&task_id=task1&user_id=42", nil)
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushAuthContext(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["git_oauth_ready_for_push"] != true {
		t.Fatalf("ready=%v body=%s", out["git_oauth_ready_for_push"], rec.Body.String())
	}
}

// TestFetchProjectReposHTTPUsesNewRouteConvention 回归测试：
// taskProjectService 删除 /api/tenant/{tid}/projects/{pid}/ 旧路由后，
// fetchProjectReposHTTP 必须使用 /api/projects/tenant_id/{tid}/{pid}，
// 否则 git push 授权上下文取不到项目仓库（404）。
func TestFetchProjectReposHTTPUsesNewRouteConvention(t *testing.T) {
	resetAuthContextServiceURLs(t)

	var gotPath, gotTraceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTraceID = r.Header.Get("X-Trace-Id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":       "proj1",
			"workspaces": []string{"ws1"},
			"git_repos":  []string{"https://git.example/repo.git"},
		})
	}))
	defer srv.Close()
	cfg.ProjectServiceURL = srv.URL

	// OPT-20260821-012: 入站 X-Trace-Id 必须透传到 taskProjectService。
	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{TraceID: "trace-opt-012-1"})
	meta, repos, err := fetchProjectReposHTTP(ctx, "t1", "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/projects/tenant_id/t1/p1" {
		t.Fatalf("expected new route /api/projects/tenant_id/t1/p1, got %s", gotPath)
	}
	if gotTraceID != "trace-opt-012-1" {
		t.Fatalf("X-Trace-Id=%q want trace-opt-012-1 (OPT-20260821-012 propagation)", gotTraceID)
	}
	if meta == nil || meta.name != "proj1" {
		t.Fatalf("unexpected meta: %#v", meta)
	}
	if len(repos) != 1 || repos[0] != "https://git.example/repo.git" {
		t.Fatalf("unexpected repos: %#v", repos)
	}
}
