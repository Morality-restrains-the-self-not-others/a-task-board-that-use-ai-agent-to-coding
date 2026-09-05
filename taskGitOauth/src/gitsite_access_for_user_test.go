package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// gitsiteTestApp 配置两个带 website 的 provider（github 官方 + gitlab tencent-sh-1），
// 供内部 gitsite 换票路径解析。
func gitsiteTestApp(t *testing.T) *App {
	t.Helper()
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {{
			Provider: "github", ServiceProvider: "github-official",
			ProviderKey: "github:github-official",
			Website:     "https://github.com",
			ClientID:    "cid", ClientSecret: "sec",
		}},
		"gitlab": {{
			Provider: "gitlab", ServiceProvider: "tencent-sh-1",
			ProviderKey: "gitlab:tencent-sh-1",
			Website:     "https://gitlab-tencent-sh-1.daydaymoney.com",
			ClientID:    "cid", ClientSecret: "sec",
		}, {
			Provider: "gitlab", ServiceProvider: "localhost",
			ProviderKey: "gitlab:localhost",
			Website:     "http://localhost:8012",
			ClientID:    "cid", ClientSecret: "sec",
		}},
	}
	return app
}

// TestGitsiteAccessForUserPathReachesHandler OPT-20260822-036 回归：
// /api/internal/gitsite/{site}/oauth/access-for-user/ 按 site 解析 provider 后到达
// handleAccessForUser（空 user_id → 400 bad user_id，而非 404）。
func TestGitsiteAccessForUserPathReachesHandler(t *testing.T) {
	app := gitsiteTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	for _, path := range []string{
		"/api/internal/gitsite/gitlab-tencent-sh-1.daydaymoney.com/oauth/access-for-user/",
		"/api/internal/gitsite/github.com/oauth/access-for-user/",
		// 带端口 site URL 编码：hostname 回退匹配 localhost。
		"/api/internal/gitsite/localhost%3A8012/oauth/access-for-user/",
	} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"user_id":""}`))
		req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("path %s: 404 — gitsite 未解析到 provider 或路径未注册", path)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s: expected 400 bad user_id, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

// TestGitsiteAccessForUserUnknownSite 未知 site → 404 not_found + errDetail。
func TestGitsiteAccessForUserUnknownSite(t *testing.T) {
	app := gitsiteTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/gitsite/unknown.example.com/oauth/access-for-user/",
		bytes.NewBufferString(`{"user_id":"u1"}`))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown site status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestCacheKeyIDIncludesRemoteUserID OPT-20260822-036(4)：cache 键含 remote_user_id，
// 同一 provider+user 下多远端账号互不共享缓存。
func TestCacheKeyIDIncludesRemoteUserID(t *testing.T) {
	if got := cacheKeyID("gitlab:tencent-sh-1", "rid-1"); got != "gitlab:tencent-sh-1|r:rid-1" {
		t.Fatalf("cacheKeyID=%q", got)
	}
	if got := cacheKeyID("github", ""); got != "github" {
		t.Fatalf("cacheKeyID empty rid=%q", got)
	}
	if cacheKeyID("g", "a") == cacheKeyID("g", "b") {
		t.Fatalf("cacheKeyID must distinguish remote_user_id")
	}

	c := infrastructure.NewAccessTokenCache()
	c.Put(cacheKeyID("gitlab:tencent-sh-1", "rid-1"), "u1", "token-1", 100)
	c.Put(cacheKeyID("gitlab:tencent-sh-1", "rid-2"), "u1", "token-2", 100)
	if got := c.Get(cacheKeyID("gitlab:tencent-sh-1", "rid-1"), "u1"); got != "token-1" {
		t.Fatalf("rid-1 token=%q want token-1", got)
	}
	if got := c.Get(cacheKeyID("gitlab:tencent-sh-1", "rid-2"), "u1"); got != "token-2" {
		t.Fatalf("rid-2 token=%q want token-2", got)
	}
	if got := c.Get(cacheKeyID("gitlab:tencent-sh-1", "rid-3"), "u1"); got != "" {
		t.Fatalf("rid-3 should miss cache, got %q", got)
	}
}

// TestGitsiteHostnameFallback 带端口 site 的 hostname 提取（localhost:8012 → localhost）。
func TestGitsiteHostnameFallback(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"localhost:8012", "localhost"},
		{"gitlab.daydaymoney.com", "gitlab.daydaymoney.com"},
		{"https://gitlab-tencent-sh-1.daydaymoney.com", "gitlab-tencent-sh-1.daydaymoney.com"},
	} {
		if got := gitsiteHostname(tc.in); got != tc.want {
			t.Fatalf("gitsiteHostname(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestGitsiteAccessForUserPathAIPHostReachesHandler(t *testing.T) {
	app := gitsiteTestApp(t)
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

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/gitsite/115.29.110.74/oauth/access-for-user/",
		bytes.NewBufferString(`{"user_id":""}`))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatalf("Path A IP must resolve via tenant connection, got 404 body=%s", rec.Body.String())
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 bad user_id after Path A resolve, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "user_id") && rec.Code != http.StatusBadRequest {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
