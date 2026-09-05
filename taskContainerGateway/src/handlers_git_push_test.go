package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tracelog"
)

func TestHandleContainerLayerGitPush_PrepareAndUpstream(t *testing.T) {
	var sawComplete atomic.Bool
	var upstreamPath string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Access-Token") != "tok" {
			http.Error(w, "token", http.StatusUnauthorized)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if body["target_branch"] != "feature/x" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pushed": true})
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		case r.URL.Path == "/api/internal/layer-git-push/prepare" || r.URL.Path == "/api/internal/layer-git-push/prepare/":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			if body["user_id"] != "42" || body["layer_id"] != "layer-1" {
				http.Error(w, "bad prepare", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":                    true,
				"use_oauth_access_push": false,
				"push_body": map[string]any{
					"target_branch": "feature/x",
				},
			})
		case r.URL.Path == "/api/internal/layer-git-push/complete" || r.URL.Path == "/api/internal/layer-git-push/complete/":
			sawComplete.Store(true)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-push/"
	body := `{"layer_id":"layer-1","target_branch":"feature/x","container_page_url":"` + upstream.URL + `/ui/x"}`
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	req.Header.Set(tracelog.Header, "trace-gitpush1234567890")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	wantPath := "/api/tenant/1/workspace/2/task/3/layers/layer-1/git/push"
	if upstreamPath != wantPath {
		t.Fatalf("upstream path=%q want %q", upstreamPath, wantPath)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sawComplete.Load() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawComplete.Load() {
		t.Fatal("expected cloud complete-layer-git-push to be called")
	}
}

func TestHandleContainerLayerGitPush_UsesOauthAccessPushPath(t *testing.T) {
	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{"company_member_id": "m1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/layer-git-push/prepare"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":                    true,
				"use_oauth_access_push": true,
				"push_body": map[string]any{
					"github_auth_by_repo": map[string]string{"acme/repo": "ghu_x"},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/layer-git-push/complete"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-push/"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"layer_id":"L1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	want := "/api/tenant/1/workspace/2/task/3/layers/L1/git/oauth-access-push"
	if upstreamPath != want {
		t.Fatalf("upstream path=%q want %q", upstreamPath, want)
	}
}

func TestHandleContainerLayerGitPushAuthContext(t *testing.T) {
	authURL, _ := startAuthCloudHotPathMocks(t, "http://127.0.0.1:9", "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": "http://127.0.0.1:9", "access_token": "tok",
			})
		case r.URL.Path == "/api/internal/layer-git-push/auth-context" ||
			r.URL.Path == "/api/internal/layer-git-push/auth-context/":
			if r.Method != http.MethodGet {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			if r.URL.Query().Get("user_id") != "42" {
				http.Error(w, "user", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"git_oauth_ready_for_push": true,
				"message":                  "ok",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-push-auth-context/"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var parsed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["git_oauth_ready_for_push"] != true {
		t.Fatalf("body=%v", parsed)
	}
}

func TestGitPushReadTimeoutSecDefault(t *testing.T) {
	t.Setenv("CONTAINER_LAYER_GIT_PUSH_READ_TIMEOUT", "")
	if got := gitPushReadTimeoutSec(); got != 45 {
		t.Fatalf("got=%v", got)
	}
	t.Setenv("CONTAINER_LAYER_GIT_PUSH_READ_TIMEOUT", "60")
	if got := gitPushReadTimeoutSec(); got != 60 {
		t.Fatalf("got=%v", got)
	}
}

func TestGithubPullRequestSummaryFromOauthRepos(t *testing.T) {
	upstream := map[string]any{
		"ok": true,
		"github_oauth_multirepo": map[string]any{
			"repos": []any{
				map[string]any{
					"push_ok":     true,
					"github_slug": "acme/demo",
					"pr": map[string]any{
						"html_url": "https://github.com/acme/demo/pull/7",
						"number":   7,
						"state":    "open",
					},
				},
			},
		},
	}
	got := githubPullRequestSummaryFromOauthRepos(upstream)
	if got == nil {
		t.Fatal("expected summary")
	}
	if got["html_url"] != "https://github.com/acme/demo/pull/7" {
		t.Fatalf("html_url=%v", got["html_url"])
	}
}

func TestHandleContainerLayerGitPush_EnrichesGithubPullRequest(t *testing.T) {
	var sawRemember atomic.Bool
	var rememberBody atomic.Value
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/git/remember-pr-html-url") {
			sawRemember.Store(true)
			raw, _ := io.ReadAll(r.Body)
			rememberBody.Store(string(raw))
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "remembered": true})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"github_oauth_multirepo": map[string]any{
				"repos": []any{
					map[string]any{
						"push_ok":     true,
						"github_slug": "acme/demo",
						"pr": map[string]any{
							"html_url": "https://github.com/acme/demo/pull/9",
							"number":   9,
						},
					},
				},
			},
		})
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{"company_member_id": "m1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/layer-git-push/prepare"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":                    true,
				"use_oauth_access_push": true,
				"push_body": map[string]any{
					"github_auth_by_repo": map[string]string{"acme/demo": "ghu_x"},
					"pr_base_branch":      "main",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/layer-git-push/complete"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-push/"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"layer_id":"L1","target_branch":"feat/x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	pr, _ := out["github_pull_request"].(map[string]any)
	if pr == nil || pr["html_url"] != "https://github.com/acme/demo/pull/9" {
		t.Fatalf("github_pull_request=%v", out["github_pull_request"])
	}
	gr, _ := out["git_remote"].(map[string]any)
	if gr == nil || gr["pr_html_url"] != "https://github.com/acme/demo/pull/9" {
		t.Fatalf("git_remote.pr_html_url=%v", out["git_remote"])
	}
	// OPT-20260817-042: 网关应兜底调用容器 remember-pr-html-url 端点持久化 PR 链接。
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sawRemember.Load() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawRemember.Load() {
		t.Fatal("expected container remember-pr-html-url to be called")
	}
	if body := rememberBody.Load(); body == nil || !strings.Contains(body.(string), "https://github.com/acme/demo/pull/9") {
		t.Fatalf("remember body=%v", body)
	}
}

func TestPrHtmlURLFromGitRemote(t *testing.T) {
	if got := prHtmlURLFromGitRemote(nil); got != "" {
		t.Fatalf("nil got=%q", got)
	}
	if got := prHtmlURLFromGitRemote(map[string]any{"git_remote": map[string]any{"pr_html_url": "  "}}); got != "" {
		t.Fatalf("blank got=%q", got)
	}
	up := map[string]any{
		"git_remote": map[string]any{"pr_html_url": "https://gitlab.daydaymoney.com/a/b/-/merge_requests/3"},
	}
	if got := prHtmlURLFromGitRemote(up); got != "https://gitlab.daydaymoney.com/a/b/-/merge_requests/3" {
		t.Fatalf("got=%q", got)
	}
	if got := prHtmlURLFromGitRemote(map[string]any{"ok": true}); got != "" {
		t.Fatalf("no git_remote got=%q", got)
	}
}
