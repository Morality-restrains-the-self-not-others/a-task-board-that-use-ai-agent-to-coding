package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func gitlabHTMLURL() string {
	return "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/12"
}

func mergeTestApp(t *testing.T) *App {
	t.Helper()
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"gitlab": {{
			Provider:        "gitlab",
			ServiceProvider: "tencent-sh-1",
			ProviderKey:     "gitlab:tencent-sh-1",
			Website:         "https://gitlab-tencent-sh-1.daydaymoney.com",
		}},
	}
	app.MergeAccessTokenFn = func(userID, htmlURL string) (string, error) {
		return "glpat-test-token", nil
	}
	return app
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func postJSON(mux http.Handler, path, body, userID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func TestMergeRequestStatusOpen(t *testing.T) {
	app := mergeTestApp(t)
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/merge_requests/12") {
			t.Fatalf("path=%s", req.URL.Path)
		}
		return jsonHTTPResponse(http.StatusOK, `{"state":"opened","title":"hello"}`), nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+gitlabHTMLURL()+`"]}`, "u1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	results, _ := out["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results=%v", out)
	}
	row, _ := results[0].(map[string]any)
	if row["state"] != "open" {
		t.Fatalf("state=%v", row["state"])
	}
}

func TestMergeRequestMergeAlreadyMergedNoopAudits(t *testing.T) {
	app := mergeTestApp(t)
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonHTTPResponse(http.StatusOK, `{"state":"merged","title":"done"}`), nil
		}
		t.Fatalf("should not merge when already merged: %s %s", req.Method, req.URL.Path)
		return nil, nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["merged"] != true || out["noop"] != true {
		t.Fatalf("body=%v", out)
	}
	var n int
	err := app.DB.QueryRow(`SELECT COUNT(*) FROM git_oauth_taskcredentialaudit WHERE action='merge_request_merge'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit rows=%d", n)
	}
	if err := app.DB.QueryRow(`SELECT COUNT(*) FROM git_oauth_appaccesstokenuseaudit WHERE action='merge_request_merge'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("access audit rows=%d", n)
	}
}

func TestMergeRequestMergeGitTimeoutReturns504Chinese(t *testing.T) {
	app := mergeTestApp(t)
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonHTTPResponse(http.StatusOK, `{"state":"opened"}`), nil
		}
		if req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/merge") {
			return nil, fmt.Errorf(
				`Put "https://gitlab-tencent-sh-1.daydaymoney.com/api/v4/projects/x/merge": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`,
			)
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL)
		return nil, nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "context deadline exceeded") || strings.Contains(body, "请检查网络") {
		t.Fatalf("must not leak raw timeout or blame network: %s", body)
	}
	if !strings.Contains(body, "Git 站点合并未在时限内完成") {
		t.Fatalf("want Chinese merge timeout, got %s", body)
	}
	if !strings.Contains(body, "gitlab-tencent-sh-1.daydaymoney.com") {
		t.Fatalf("want git host in message: %s", body)
	}
	var n int
	if err := app.DB.QueryRow(`SELECT COUNT(*) FROM git_oauth_taskcredentialaudit WHERE action='merge_request_merge'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit rows=%d", n)
	}
}

func TestIsGitAPITimeout(t *testing.T) {
	if isGitAPITimeout(nil) {
		t.Fatal("nil is not timeout")
	}
	if isGitAPITimeout(fmt.Errorf("git merge http 405: method not allowed")) {
		t.Fatal("http 405 is not timeout")
	}
	raw := fmt.Errorf("Put %q: context deadline exceeded (Client.Timeout exceeded while awaiting headers)", "https://gitlab.example/merge")
	if !isGitAPITimeout(raw) {
		t.Fatalf("want timeout detection for %v", raw)
	}
}

func TestMergeRequestMergePerformsPut(t *testing.T) {
	app := mergeTestApp(t)
	putSeen := false
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonHTTPResponse(http.StatusOK, `{"state":"opened"}`), nil
		}
		if req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/merge") {
			putSeen = true
			return jsonHTTPResponse(http.StatusOK, `{"state":"merged"}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL)
		return nil, nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	if !putSeen {
		t.Fatal("expected PUT merge")
	}
}

func TestMergeRequestRejectsUnlistedHost(t *testing.T) {
	app := mergeTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/1/",
		`{"html_url":"https://evil.example/group/repo/-/merge_requests/1"}`, "u1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestMergeRequestMergeNotConnectedReturnsChinese(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = func(string, string) (string, error) {
		return "", nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusConflict {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "git oauth not connected") {
		t.Fatalf("must not leak English detail: %s", body)
	}
	if !strings.Contains(body, "尚未绑定 Git 网站 OAuth") {
		t.Fatalf("want Chinese bind hint, got %s", body)
	}
}

func TestMergeRequestMergeRefreshFailureNotCollapsedToNotConnected(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = func(string, string) (string, error) {
		return "", fmt.Errorf("refresh timeout")
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "git oauth not connected") {
		t.Fatalf("refresh failure must not be labeled not-connected: %s", body)
	}
	if !strings.Contains(body, "refresh timeout") {
		t.Fatalf("want underlying refresh error, got %s", body)
	}
}

func TestMergeRequestMergeRefresh400ReturnsReauthChinese(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = func(string, string) (string, error) {
		return "", fmt.Errorf("gitlab refresh http 400")
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "git oauth not connected") {
		t.Fatalf("must not leak English not-connected: %s", body)
	}
	if strings.Contains(body, "gitlab refresh http 400") {
		t.Fatalf("must not leak raw refresh status: %s", body)
	}
	if !strings.Contains(body, "Git OAuth 授权已失效") {
		t.Fatalf("want Chinese reauth hint, got %s", body)
	}
}

func TestMergeRequestStatusRefresh400IsChineseReauth(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = func(string, string) (string, error) {
		return "", fmt.Errorf("gitlab refresh http 400")
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+gitlabHTMLURL()+`"]}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	results, _ := out["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results=%v", out)
	}
	row, _ := results[0].(map[string]any)
	errText := fmt.Sprint(row["error"])
	if strings.Contains(errText, "gitlab refresh http 400") {
		t.Fatalf("must not leak raw refresh status: %v", row)
	}
	if !strings.Contains(errText, "Git OAuth 授权已失效") {
		t.Fatalf("want Chinese reauth hint, got %v", row)
	}
}

func TestMergeRequestStatusNotConnectedErrorIsChinese(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = func(string, string) (string, error) {
		return "", nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+gitlabHTMLURL()+`"]}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	results, _ := out["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results=%v", out)
	}
	row, _ := results[0].(map[string]any)
	errText := fmt.Sprint(row["error"])
	if strings.Contains(errText, "git oauth not connected") {
		t.Fatalf("must not leak English status error: %v", row)
	}
	if !strings.Contains(errText, "尚未绑定 Git 网站 OAuth") {
		t.Fatalf("want Chinese bind hint, got %v", row)
	}
}

func TestAccessTokenForMergeURLUsesConnectionLookupKeys(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = nil
	const userID = "42"
	cipher, err := app.Fernet.Encrypt("ghp_merge_lookup_token")
	if err != nil || cipher == "" {
		t.Fatalf("encrypt: %v", err)
	}
	_, err = app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, ?, 'remote-1', 'ljy', 'api', 'active', '', NOW(), NOW())`,
		"gitlab:tencent-sh-1", userID, cipher)
	if err != nil {
		t.Fatal(err)
	}
	token, err := app.accessTokenForMergeURL(userID, gitlabHTMLURL())
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if token != "ghp_merge_lookup_token" {
		t.Fatalf("token=%q", token)
	}
}

func insertGitlabMergeCredential(t *testing.T, app *App, userID, refreshPlain string) {
	t.Helper()
	cipher, err := app.Fernet.Encrypt(refreshPlain)
	if err != nil || cipher == "" {
		t.Fatalf("encrypt: %v", err)
	}
	_, err = app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, ?, 'remote-1', 'ljy', 'api', 'active', '', NOW(), NOW())`,
		"gitlab:tencent-sh-1", userID, cipher)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeRequestStatusUsesCachedAccessTokenWithoutRefresh(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = nil
	const userID = "u-shared-cache"
	insertGitlabMergeCredential(t, app, userID, "glrt_stale_would_400")
	app.Cache.Put(cacheKeyID("gitlab:tencent-sh-1", "remote-1"), userID, "glpat-cached-shared", 3600)

	refreshCalls := 0
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls++
		return nil, fmt.Errorf("gitlab refresh http 400")
	}
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer glpat-cached-shared" {
			t.Fatalf("authorization=%q", got)
		}
		return jsonHTTPResponse(http.StatusOK, `{"state":"opened","title":"hello"}`), nil
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+gitlabHTMLURL()+`"]}`, userID)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	results, _ := out["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results=%v", out)
	}
	row, _ := results[0].(map[string]any)
	if row["state"] != "open" {
		t.Fatalf("state=%v body=%s", row["state"], rr.Body.String())
	}
	if errText := strings.TrimSpace(fmt.Sprint(row["error"])); errText != "" && errText != "<nil>" {
		t.Fatalf("cached token must not surface refresh error: %v", row)
	}
	if refreshCalls != 0 {
		t.Fatalf("shared cache hit must not refresh, calls=%d", refreshCalls)
	}
}

func TestAccessTokenForMergeURLRefreshPersistsAndCaches(t *testing.T) {
	app := mergeTestApp(t)
	app.MergeAccessTokenFn = nil
	const userID = "u-persist-refresh"
	insertGitlabMergeCredential(t, app, userID, "glrt_old")

	refreshCalls := 0
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls++
		if refreshPlain != "glrt_old" {
			t.Fatalf("refreshPlain=%q", refreshPlain)
		}
		return map[string]any{
			"access_token":  "glpat-after-refresh",
			"refresh_token": "glrt_new",
			"expires_in":    7200,
		}, nil
	}

	token, err := app.accessTokenForMergeURL(userID, gitlabHTMLURL())
	if err != nil {
		t.Fatalf("first issue: %v", err)
	}
	if token != "glpat-after-refresh" {
		t.Fatalf("token=%q", token)
	}
	if refreshCalls != 1 {
		t.Fatalf("first issue refreshCalls=%d", refreshCalls)
	}

	var stored string
	if err := app.DB.QueryRow(
		`SELECT refresh_token_cipher FROM git_oauth_appusercredential WHERE task2app_user_id=?`,
		userID,
	).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	plain, _ := app.Fernet.Decrypt(stored)
	if plain != "glrt_new" {
		t.Fatalf("persisted refresh=%q", plain)
	}

	token, err = app.accessTokenForMergeURL(userID, gitlabHTMLURL())
	if err != nil {
		t.Fatalf("second issue: %v", err)
	}
	if token != "glpat-after-refresh" {
		t.Fatalf("cached token=%q", token)
	}
	if refreshCalls != 1 {
		t.Fatalf("second issue must hit cache, refreshCalls=%d", refreshCalls)
	}
}

func TestGitMergeRequestStatusURL_HTTPOrigin(t *testing.T) {
	ref, err := domain.ParseMergeRequestURL("http://115.29.110.74/example-user/somanyad/-/merge_requests/1")
	if err != nil {
		t.Fatal(err)
	}
	got := gitMergeRequestStatusURL(ref)
	want := "http://115.29.110.74/api/v4/projects/example-user%2Fsomanyad/merge_requests/1"
	if got != want {
		t.Fatalf("status url=%s want %s", got, want)
	}
}

func httpTenantGitLabProvider() infrastructure.ProviderConfig {
	return infrastructure.ProviderConfig{
		Provider:        "gitlab",
		ServiceProvider: "tenant-http",
		ProviderKey:     "gitlab:tenant-http",
		Website:         "http://115.29.110.74",
	}
}

func TestMergeRequestStatusHTTPHtmlDoesNotUseHTTPS443(t *testing.T) {
	app := mergeTestApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"gitlab": {httpTenantGitLabProvider()},
	}
	html := "http://115.29.110.74/example-user/somanyad/-/merge_requests/1"
	var gotURL string
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		if req.URL.Scheme != "http" {
			t.Fatalf("scheme=%s want http (https defaults to :443)", req.URL.Scheme)
		}
		if req.URL.Port() == "443" {
			t.Fatalf("port=443 url=%s", req.URL.String())
		}
		return jsonHTTPResponse(http.StatusOK, `{"state":"opened","title":"mr"}`), nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+html+`"]}`, "u1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	if !strings.HasPrefix(gotURL, "http://115.29.110.74/") {
		t.Fatalf("api url=%s", gotURL)
	}
}

func TestMergeRequestStatusHTTPSHtmlUsesProviderHTTPWebsite(t *testing.T) {
	app := mergeTestApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"gitlab": {httpTenantGitLabProvider()},
	}
	html := "https://115.29.110.74/example-user/somanyad/-/merge_requests/1"
	var gotURL string
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		if req.URL.Scheme != "http" {
			t.Fatalf("scheme=%s want provider website http", req.URL.Scheme)
		}
		return jsonHTTPResponse(http.StatusOK, `{"state":"opened"}`), nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+html+`"]}`, "u1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	if !strings.HasPrefix(gotURL, "http://115.29.110.74/") {
		t.Fatalf("api url=%s want http origin from website", gotURL)
	}
}
