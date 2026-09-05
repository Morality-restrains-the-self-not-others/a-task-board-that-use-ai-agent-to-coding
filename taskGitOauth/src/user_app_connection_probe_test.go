package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func getUserAppConnection(t *testing.T, app *App, path string, userID string) (int, map[string]any, string) {
	t.Helper()
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	return rec.Code, body, rec.Body.String()
}

func insertActiveCredential(t *testing.T, app *App, provider, userID, cipher string) {
	t.Helper()
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, ?, 'remote-1', 'octocat', 'repo', 'active', '', NOW(), NOW())`,
		provider, userID, cipher)
	if err != nil {
		t.Fatal(err)
	}
}

func assertNoAccessTokenLeak(t *testing.T, raw string, body map[string]any) {
	t.Helper()
	if _, ok := body["access_token"]; ok {
		t.Fatalf("probe response must not include access_token: %s", raw)
	}
	lower := strings.ToLower(raw)
	for _, needle := range []string{"gho_", "ghp_", "ghu_", "ghr_", "github_pat_", "glrt_"} {
		if strings.Contains(lower, needle) {
			t.Fatalf("probe response leaked token material %q: %s", needle, raw)
		}
	}
}

func stubGitLabHTTP(app *App, status int, err error, probes *atomic.Int32) {
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if probes != nil {
			probes.Add(1)
		}
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}
}

func TestUserAppConnectionGetWithoutProbeOmitsAccessTokenValid(t *testing.T) {
	app := providerTestApp(t)
	const userID = "873093522473906176"
	insertActiveCredential(t, app, "github:github-official-daydaymoney", userID, "cipher")

	code, body, _ := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo",
		userID)
	if code != http.StatusOK {
		t.Fatalf("status=%d", code)
	}
	if body["connected"] != true {
		t.Fatalf("connected=%v", body["connected"])
	}
	if _, ok := body["access_token_valid"]; ok {
		t.Fatalf("without probe_access_token, access_token_valid must be omitted: %v", body)
	}
}

func TestUserAppConnectionProbeNotConnectedSkipsRefresh(t *testing.T) {
	app := providerTestApp(t)
	var refreshCalls atomic.Int32
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls.Add(1)
		return map[string]any{"access_token": "gho_should_not_issue"}, nil
	}

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo&probe_access_token=1",
		"873093522473906176")
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if body["connected"] != false {
		t.Fatalf("connected=%v", body["connected"])
	}
	if body["access_token_valid"] != false {
		t.Fatalf("access_token_valid=%v", body["access_token_valid"])
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("unconnected probe must not refresh, calls=%d", refreshCalls.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeCacheHitValidWithoutRefresh(t *testing.T) {
	app := providerTestApp(t)
	const userID = "873093522473906176"
	insertActiveCredential(t, app, "github:github-official-daydaymoney", userID, "cipher")
	app.Cache.Put(cacheKeyID("github:github-official-daydaymoney", "remote-1"), userID, "gho_cached_secret_do_not_leak", 3600)

	var refreshCalls atomic.Int32
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls.Add(1)
		return nil, fmt.Errorf("refresh should not run on cache hit")
	}

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo&probe_access_token=true",
		userID)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if body["connected"] != true {
		t.Fatalf("connected=%v", body["connected"])
	}
	if body["access_token_valid"] != true {
		t.Fatalf("access_token_valid=%v", body["access_token_valid"])
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("cache hit must not refresh, calls=%d", refreshCalls.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeDirectGithubTokenValidWithoutRefresh(t *testing.T) {
	app := providerTestApp(t)
	const userID = "873093522473906176"
	cipher, err := app.Fernet.Encrypt("ghp_probe_direct_secret")
	if err != nil {
		t.Fatal(err)
	}
	insertActiveCredential(t, app, "github:github-official-daydaymoney", userID, cipher)

	var refreshCalls atomic.Int32
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls.Add(1)
		return nil, fmt.Errorf("direct github token must not refresh")
	}

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo&probe_access_token=1",
		userID)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if body["access_token_valid"] != true {
		t.Fatalf("access_token_valid=%v", body["access_token_valid"])
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("direct github token must not refresh, calls=%d", refreshCalls.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeRefreshFailReturns200ValidFalse(t *testing.T) {
	app := providerTestApp(t)
	const userID = "827923618451263488"
	cipher, err := app.Fernet.Encrypt("glrt_expired_refresh")
	if err != nil {
		t.Fatal(err)
	}
	insertActiveCredential(t, app, "gitlab:gitlab-local", userID, cipher)

	var refreshCalls atomic.Int32
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls.Add(1)
		if refreshPlain != "glrt_expired_refresh" {
			t.Errorf("unexpected refresh plain")
		}
		return nil, fmt.Errorf("gitlab refresh http 400")
	}
	stubGitLabHTTP(app, 200, nil, nil)

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=http%3A%2F%2Flocalhost%3A8012%2Fgroup%2Frepo&probe_access_token=1",
		userID)
	if code != http.StatusOK {
		t.Fatalf("probe refresh failure must stay 200, status=%d body=%s", code, raw)
	}
	if body["connected"] != true {
		t.Fatalf("connected remains DB truth, got %v", body["connected"])
	}
	if body["access_token_valid"] != false {
		t.Fatalf("access_token_valid=%v", body["access_token_valid"])
	}
	if refreshCalls.Load() != 1 {
		t.Fatalf("expected one refresh, calls=%d", refreshCalls.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeGitHubIssueHangReturnsWithinTimeout(t *testing.T) {
	oldTimeout := probeIssueTokenTimeout
	probeIssueTokenTimeout = 150 * time.Millisecond
	t.Cleanup(func() { probeIssueTokenTimeout = oldTimeout })

	app := providerTestApp(t)
	const userID = "873093522473906176"
	cipher, err := app.Fernet.Encrypt("glrt_should_refresh_hang")
	if err != nil {
		t.Fatal(err)
	}
	insertActiveCredential(t, app, "github:github-official-daydaymoney", userID, cipher)

	// OPT-20260902-010：GitHub refresh 卡住（github.com 不可达 dial 数十秒）时，
	// probe 必须在显式短超时内返回 issue_failed，而不是等 refresh 自然失败。
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		time.Sleep(800 * time.Millisecond)
		return nil, errors.New("github refresh hang")
	}

	start := time.Now()
	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo&probe_access_token=1",
		userID)
	elapsed := time.Since(start)

	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if body["access_token_valid"] != false {
		t.Fatalf("access_token_valid=%v body=%s", body["access_token_valid"], raw)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("probe must return within ~150ms timeout, elapsed=%v", elapsed)
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeGitLabUnreachableDespiteCache(t *testing.T) {
	app := providerTestApp(t)
	const userID = "873093522473906176"
	insertActiveCredential(t, app, "gitlab:gitlab-local", userID, "cipher")
	app.Cache.Put(cacheKeyID("gitlab:gitlab-local", "remote-1"), userID, "glpat_cached_do_not_leak", 3600)

	var probes, refreshCalls atomic.Int32
	stubGitLabHTTP(app, 0, errors.New("dial tcp 127.0.0.1:8012: i/o timeout"), &probes)
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls.Add(1)
		return map[string]any{"access_token": "glpat_should_not_issue"}, nil
	}

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=http%3A%2F%2Flocalhost%3A8012%2Fgroup%2Frepo&probe_access_token=1",
		userID)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if body["connected"] != true {
		t.Fatalf("connected remains DB truth, got %v", body["connected"])
	}
	if body["network_status"] != domain.ReachabilityUnreachable {
		t.Fatalf("network_status=%v", body["network_status"])
	}
	if body["access_token_valid"] != false {
		t.Fatalf("access_token_valid=%v", body["access_token_valid"])
	}
	if probes.Load() < 1 {
		t.Fatalf("must probe GitLab even on cache hit, probes=%d", probes.Load())
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("unreachable must not refresh, calls=%d", refreshCalls.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeIntranetGitLabSkipsUnreachable(t *testing.T) {
	app := providerTestApp(t)
	const userID = "873093522473906176"
	_, err := app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       "co-intranet-oauth",
		BaseURL:         "http://localhost:8012",
		ClientID:        "cid",
		ClientSecretEnc: "enc",
		RedirectURI:     "http://localhost/cb",
		Scope:           domain.DefaultTenantGitLabScope,
		Active:          true,
		Intranet:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	insertActiveCredential(t, app, "gitlab:gitlab-local", userID, "cipher")

	var probes atomic.Int32
	stubGitLabHTTP(app, 0, errors.New("dial tcp: i/o timeout"), &probes)

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=http%3A%2F%2Flocalhost%3A8012%2Fgroup%2Frepo&probe_access_token=1",
		userID)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if body["network_status"] != domain.ReachabilitySkippedIntranet {
		t.Fatalf("network_status=%v body=%s", body["network_status"], raw)
	}
	if body["access_token_valid"] != true {
		t.Fatalf("intranet must keep token valid, got %v", body["access_token_valid"])
	}
	if probes.Load() != 0 {
		t.Fatalf("intranet must not live-probe GitLab, probes=%d", probes.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestUserAppConnectionProbeGithubOmitsNetworkStatus(t *testing.T) {
	app := providerTestApp(t)
	const userID = "873093522473906176"
	insertActiveCredential(t, app, "github:github-official-daydaymoney", userID, "cipher")
	app.Cache.Put(cacheKeyID("github:github-official-daydaymoney", "remote-1"), userID, "gho_cached", 3600)

	var probes atomic.Int32
	stubGitLabHTTP(app, 0, errors.New("should not probe github.com"), &probes)

	code, body, raw := getUserAppConnection(t, app,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo&probe_access_token=1",
		userID)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, raw)
	}
	if _, ok := body["network_status"]; ok {
		t.Fatalf("github must omit network_status: %v", body)
	}
	if body["access_token_valid"] != true {
		t.Fatalf("access_token_valid=%v", body["access_token_valid"])
	}
	if probes.Load() != 0 {
		t.Fatalf("must not GitLab-probe GitHub, probes=%d", probes.Load())
	}
	assertNoAccessTokenLeak(t, raw, body)
}

func TestOpenAPIUserAppConnectionProbeAccessTokenOptional(t *testing.T) {
	doc := openAPIDocument()
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("missing paths")
	}
	entry, ok := paths["/api/git-oauth/user-app-connection/"].(map[string]any)
	if !ok {
		t.Fatal("missing user-app-connection path")
	}
	op, ok := entry["get"].(map[string]any)
	if !ok {
		t.Fatal("missing get")
	}
	params, ok := op["parameters"].([]any)
	if !ok {
		t.Fatal("missing parameters")
	}
	found := false
	for _, p := range params {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if pm["name"] != "probe_access_token" {
			continue
		}
		found = true
		if req, _ := pm["required"].(bool); req {
			t.Fatal("probe_access_token must be optional")
		}
	}
	if !found {
		t.Fatal("GET missing probe_access_token parameter")
	}
}
