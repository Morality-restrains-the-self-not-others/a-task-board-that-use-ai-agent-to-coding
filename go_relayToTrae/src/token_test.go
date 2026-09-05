package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestExtractOriginValidHTTP(t *testing.T) {
	got := extractOrigin("http://example.com/path/to/resource")
	if got != "http://example.com" {
		t.Errorf("expected 'http://example.com', got %q", got)
	}
}

func TestExtractOriginValidHTTPS(t *testing.T) {
	got := extractOrigin("https://api.daydaymoney.com:8080/v1/endpoint")
	if got != "https://api.daydaymoney.com:8080" {
		t.Errorf("expected 'https://api.daydaymoney.com:8080', got %q", got)
	}
}

func TestExtractOriginTrailingSlash(t *testing.T) {
	got := extractOrigin("http://127.0.0.1:8001/")
	if got != "http://127.0.0.1:8001" {
		t.Errorf("expected 'http://127.0.0.1:8001', got %q", got)
	}
}

func TestExtractOriginEmptyString(t *testing.T) {
	got := extractOrigin("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractOriginWhitespaceOnly(t *testing.T) {
	got := extractOrigin("   ")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractOriginInvalidURL(t *testing.T) {
	got := extractOrigin("not-a-valid-url")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractOriginFTPIgnored(t *testing.T) {
	got := extractOrigin("ftp://files.example.com/data")
	if got != "" {
		t.Errorf("expected empty for non-http scheme, got %q", got)
	}
}

func TestExtractOriginNoPath(t *testing.T) {
	got := extractOrigin("http://localhost:8797")
	if got != "http://127.0.0.1:8797" {
		t.Errorf("expected 'http://127.0.0.1:8797', got %q", got)
	}
}

func TestNormalizeLoopbackOriginLocalhost(t *testing.T) {
	got := normalizeLoopbackOrigin("http://localhost:8001")
	if got != "http://127.0.0.1:8001" {
		t.Errorf("expected 'http://127.0.0.1:8001', got %q", got)
	}
}

func TestNormalizeLoopbackOriginPreservesNonLoopback(t *testing.T) {
	got := normalizeLoopbackOrigin("http://api.daydaymoney.com:8001")
	if got != "http://api.daydaymoney.com:8001" {
		t.Errorf("expected unchanged origin, got %q", got)
	}
}

func TestBusinessAPIEndpointForExchangeDirectEndpoint(t *testing.T) {
	env := map[string]string{
		"BUSINESS_API_ENDPOINT": "http://custom.example.com/api/v2",
	}
	got := businessAPIEndpointForExchange(env)
	if got != "http://custom.example.com/api/v2" {
		t.Errorf("expected direct endpoint, got %q", got)
	}
}

func TestBusinessAPIEndpointForExchangeAltKey(t *testing.T) {
	env := map[string]string{
		"BusinessApiEndPoint": "http://alt.example.com/api/",
	}
	got := businessAPIEndpointForExchange(env)
	if got != "http://alt.example.com/api" {
		t.Errorf("expected stripped trailing slash, got %q", got)
	}
}

func TestBusinessAPIEndpointForExchangeFromOrigin(t *testing.T) {
	env := map[string]string{
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://origin.example.com:8001",
	}
	got := businessAPIEndpointForExchange(env)
	if got != "http://origin.example.com:8001/api" {
		t.Errorf("expected origin-derived endpoint, got %q", got)
	}
}

func TestBusinessAPIEndpointForExchangeEmptyAll(t *testing.T) {
	env := map[string]string{}
	got := businessAPIEndpointForExchange(env)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExpandEnvForRuntimeBasic(t *testing.T) {
	envIn := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     "http://task.example.com",
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://biz.example.com",
		"COMMENT_ID":                   "cmt_1",
	}
	got := expandEnvForRuntime(envIn, "t1", "w1", "task1")
	if got["TASK_API_ENDPOINT"] != "http://task.example.com/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud" {
		t.Errorf("unexpected TASK_API_ENDPOINT: %s", got["TASK_API_ENDPOINT"])
	}
	if got["BUSINESS_API_ENDPOINT"] != "http://biz.example.com/api" {
		t.Errorf("unexpected BUSINESS_API_ENDPOINT: %s", got["BUSINESS_API_ENDPOINT"])
	}
}

func TestExpandEnvForRuntimeIncludesCommentInTaskApiEndPoint(t *testing.T) {
	envIn := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN": "http://task.example.com",
		"COMMENT_ID":               "cmt_1",
	}
	got := expandEnvForRuntime(envIn, "t1", "w1", "task1")
	want := "http://task.example.com/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud"
	if got["TASK_API_ENDPOINT"] != want {
		t.Errorf("unexpected TASK_API_ENDPOINT: %s", got["TASK_API_ENDPOINT"])
	}
	if got["TaskApiEndPoint"] != want {
		t.Errorf("unexpected TaskApiEndPoint: %s", got["TaskApiEndPoint"])
	}
}

func TestCloudTokenAPIPrefixWithCommentID(t *testing.T) {
	got, err := cloudTokenAPIPrefix("http://task.example.com", "t1", "w1", "task1", "cmt_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "http://task.example.com/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestCloudTokenAPIPrefixWithoutCommentID(t *testing.T) {
	got, err := cloudTokenAPIPrefix("http://task.example.com", "t1", "w1", "task1", "")
	if err == nil {
		t.Fatalf("expected error, got %q", got)
	}
	if got != "" {
		t.Errorf("expected empty prefix, got %q", got)
	}
}

func TestCloudTokenAPIPrefixIgnoresDashCommentID(t *testing.T) {
	got, err := cloudTokenAPIPrefix("http://task.example.com", "t1", "w1", "task1", "-")
	if err == nil {
		t.Fatalf("expected error, got %q", got)
	}
	if got != "" {
		t.Errorf("expected empty prefix, got %q", got)
	}
}

func TestExpandEnvForRuntimeWithoutCommentOmitsTaskApiEndpoint(t *testing.T) {
	envIn := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN": "http://task.example.com",
	}
	got := expandEnvForRuntime(envIn, "t1", "w1", "task1")
	if _, ok := got["TASK_API_ENDPOINT"]; ok {
		t.Errorf("must not emit legacy TASK_API_ENDPOINT, got %q", got["TASK_API_ENDPOINT"])
	}
	if _, ok := got["TaskApiEndPoint"]; ok {
		t.Errorf("must not emit legacy TaskApiEndPoint, got %q", got["TaskApiEndPoint"])
	}
}

func TestExpandEnvForRuntimeSetsTenantIDs(t *testing.T) {
	envIn := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN": "http://task.example.com",
	}
	got := expandEnvForRuntime(envIn, "tid1", "wid1", "task1")
	if got["tenantId"] != "tid1" {
		t.Errorf("expected tenantId=tid1, got %q", got["tenantId"])
	}
	if got["workspaceId"] != "wid1" {
		t.Errorf("expected workspaceId=wid1, got %q", got["workspaceId"])
	}
	if got["taskId"] != "task1" {
		t.Errorf("expected taskId=task1, got %q", got["taskId"])
	}
}

func TestExpandEnvForRuntimePreservesExistingKeys(t *testing.T) {
	envIn := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN": "http://task.example.com",
		"TASK_API_ENDPOINT":        "http://custom/do-not-overwrite",
	}
	got := expandEnvForRuntime(envIn, "t1", "w1", "task1")
	if got["TASK_API_ENDPOINT"] != "http://custom/do-not-overwrite" {
		t.Errorf("expected preserved value, got %q", got["TASK_API_ENDPOINT"])
	}
}

func TestExpandEnvForRuntimeEmptyInputs(t *testing.T) {
	got := expandEnvForRuntime(map[string]string{}, "", "", "")
	if len(got) != 0 {
		t.Errorf("expected empty env, got %d keys", len(got))
	}
}

func TestExtractAccessTokenFromUIURLValid(t *testing.T) {
	got := extractAccessTokenFromUIURL("http://127.0.0.1:8765/ui/my-secret-token")
	if got != "my-secret-token" {
		t.Errorf("expected 'my-secret-token', got %q", got)
	}
}

func TestExtractAccessTokenFromUIURLScoped(t *testing.T) {
	got := extractAccessTokenFromUIURL(
		"http://127.0.0.1:8765/ui/tenant/t1/workspace/w1/task/task1/tok_scoped",
	)
	if got != "tok_scoped" {
		t.Errorf("expected 'tok_scoped', got %q", got)
	}
}

func TestExtractAccessTokenFromUIURLEmpty(t *testing.T) {
	got := extractAccessTokenFromUIURL("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractAccessTokenFromUIURLNoProtocol(t *testing.T) {
	got := extractAccessTokenFromUIURL("not-a-url")
	if got != "" {
		t.Errorf("expected empty for non-url, got %q", got)
	}
}

func TestExtractAccessTokenFromUIURLNoUIPath(t *testing.T) {
	got := extractAccessTokenFromUIURL("http://127.0.0.1:8765/other/token")
	if got != "" {
		t.Errorf("expected empty for non-ui path, got %q", got)
	}
}

func TestExtractAccessTokenFromUIURLOnlyHost(t *testing.T) {
	got := extractAccessTokenFromUIURL("http://127.0.0.1:8765")
	if got != "" {
		t.Errorf("expected empty for no path, got %q", got)
	}
}

func TestDefaultHTTPClientDisablesProxy(t *testing.T) {
	transportProxyIsNil(t, defaultHTTPClient)
}

func TestTokenExchangeIgnoresProxyEnv(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:9")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:9")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ok-token"}`))
	}))
	defer server.Close()

	resp, err := doJSONPost(context.Background(), defaultHTTPClient, server.URL, map[string]string{"refresh_token": "x"}, 300*time.Millisecond)
	if err != nil {
		t.Fatalf("expected no-proxy request to succeed, got error: %v", err)
	}
	got, _ := resp["access_token"].(string)
	if got != "ok-token" {
		t.Fatalf("expected access_token=ok-token, got %q", got)
	}
}

func TestDoJSONPostUsesSystemProxyWhenConfigured(t *testing.T) {
	t.Setenv("RELAY_TO_TRAE_BACKEND_PROXY_MODE", backendProxyModeSystem)
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"proxy-token"}`))
	}))
	defer backendServer.Close()

	var proxyHits int32
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&proxyHits, 1)
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		forwardReq, err := http.NewRequest(http.MethodPost, backendServer.URL, io.NopCloser(bytes.NewReader(bodyBytes)))
		if err != nil {
			t.Fatalf("build forward request: %v", err)
		}
		forwardReq.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(forwardReq)
		if err != nil {
			t.Fatalf("forward request failed: %v", err)
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	t.Setenv("HTTP_PROXY", proxyServer.URL)
	t.Setenv("HTTPS_PROXY", proxyServer.URL)
	t.Setenv("ALL_PROXY", proxyServer.URL)

	resp, err := doJSONPost(
		context.Background(),
		newBackendHTTPClient(500*time.Millisecond),
		"http://example.invalid/internal/token",
		map[string]string{"refresh_token": "x"},
		500*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("expected request to succeed via system proxy, got: %v", err)
	}
	got, _ := resp["access_token"].(string)
	if got != "proxy-token" {
		t.Fatalf("expected access_token=proxy-token, got %q", got)
	}
	if atomic.LoadInt32(&proxyHits) < 1 {
		t.Fatal("expected proxy to be used at least once")
	}
}

func TestParseAccessTokenExpiresAtValidUTC(t *testing.T) {
	got := parseAccessTokenExpiresAt("2026-07-09 12:00:00")
	want := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseAccessTokenExpiresAtEmptyOrInvalid(t *testing.T) {
	if !parseAccessTokenExpiresAt("").IsZero() {
		t.Fatal("empty should be zero")
	}
	if !parseAccessTokenExpiresAt("not-a-time").IsZero() {
		t.Fatal("invalid should be zero")
	}
}

func TestAccessTokenNeedsRefresh(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	skew := 5 * time.Minute

	if accessTokenNeedsRefresh(time.Time{}, now, skew) {
		t.Fatal("zero expiry must not need refresh")
	}
	if accessTokenNeedsRefresh(now.Add(30*time.Minute), now, skew) {
		t.Fatal("30min remaining should not need refresh")
	}
	if !accessTokenNeedsRefresh(now.Add(2*time.Minute), now, skew) {
		t.Fatal("2min remaining should need refresh")
	}
	if !accessTokenNeedsRefresh(now.Add(-time.Minute), now, skew) {
		t.Fatal("already expired should need refresh")
	}
	if !accessTokenNeedsRefresh(now.Add(5*time.Minute), now, skew) {
		t.Fatal("exactly at skew boundary should need refresh")
	}
}

func TestPerformTokenExchangePersistsExpiresAt(t *testing.T) {
	var exchangeHits, refreshHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/server-container-token/exchange-refresh/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&exchangeHits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rt-abc"}`))
	})
	mux.HandleFunc("/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/server-container-token/refresh-access/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshHits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-new","expires_at":"2026-07-09 15:00:00"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	got, err := performTokenExchange("at-old", server.URL, server.URL+"/api", "t1", "w1", "task1", "cmt_1")
	if err != nil {
		t.Fatalf("performTokenExchange: %v", err)
	}
	if got.AccessToken != "at-new" {
		t.Fatalf("AccessToken=%q", got.AccessToken)
	}
	if got.RefreshToken != "rt-abc" {
		t.Fatalf("RefreshToken=%q", got.RefreshToken)
	}
	wantExp := time.Date(2026, 7, 9, 15, 0, 0, 0, time.UTC)
	if !got.ExpiresAt.Equal(wantExp) {
		t.Fatalf("ExpiresAt=%v want %v", got.ExpiresAt, wantExp)
	}
	if atomic.LoadInt32(&exchangeHits) != 1 || atomic.LoadInt32(&refreshHits) != 1 {
		t.Fatalf("hits exchange=%d refresh=%d", exchangeHits, refreshHits)
	}
}

func TestPerformTokenExchangeUsesCommentPath(t *testing.T) {
	var exchangePath, refreshPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tenant/t1/workspace/w1/task/task1/comment/cmt_9/cloud/server-container-token/exchange-refresh/", func(w http.ResponseWriter, r *http.Request) {
		exchangePath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rt-cmt"}`))
	})
	mux.HandleFunc("/api/tenant/t1/workspace/w1/task/task1/comment/cmt_9/cloud/server-container-token/refresh-access/", func(w http.ResponseWriter, r *http.Request) {
		refreshPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-cmt","expires_at":"2026-07-09 15:00:00"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	got, err := performTokenExchange("at-old", server.URL, server.URL+"/api", "t1", "w1", "task1", "cmt_9")
	if err != nil {
		t.Fatalf("performTokenExchange: %v", err)
	}
	if got.AccessToken != "at-cmt" {
		t.Fatalf("AccessToken=%q", got.AccessToken)
	}
	if exchangePath == "" || refreshPath == "" {
		t.Fatal("expected both exchange-refresh and refresh-access to hit comment path")
	}
}

func TestEnsureFreshAccessTokenRefreshesWhenNearExpiry(t *testing.T) {
	var refreshHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tenant/t1/workspace/w1/task/task-near/comment/cmt_near/cloud/server-container-token/refresh-access/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshHits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-refreshed","expires_at":"2026-07-09 18:00:00"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	stateMu.Lock()
	state.AccessToken = "at-old"
	state.RefreshToken = "rt-keep"
	state.AccessTokenExpiresAt = time.Now().UTC().Add(2 * time.Minute)
	state.ActiveTaskID = "task-near"
	registeredTasks = map[string]*RegisteredTask{
		"task-near": {
			TenantID:      "t1",
			WorkspaceID:   "w1",
			TaskID:        "task-near",
			CommentID:     "cmt_near",
			TaskAPIOrigin: server.URL,
			AccessToken:   "at-old",
		},
	}
	stateMu.Unlock()
	defer func() {
		stateMu.Lock()
		state.AccessToken = ""
		state.RefreshToken = ""
		state.AccessTokenExpiresAt = time.Time{}
		state.ActiveTaskID = ""
		registeredTasks = make(map[string]*RegisteredTask)
		stateMu.Unlock()
	}()

	if err := ensureFreshAccessToken(time.Now().UTC()); err != nil {
		t.Fatalf("ensureFreshAccessToken: %v", err)
	}
	if atomic.LoadInt32(&refreshHits) != 1 {
		t.Fatalf("expected 1 refresh, got %d", refreshHits)
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if state.AccessToken != "at-refreshed" {
		t.Fatalf("AccessToken=%q", state.AccessToken)
	}
	wantExp := time.Date(2026, 7, 9, 18, 0, 0, 0, time.UTC)
	if !state.AccessTokenExpiresAt.Equal(wantExp) {
		t.Fatalf("ExpiresAt=%v", state.AccessTokenExpiresAt)
	}
	if registeredTasks["task-near"].AccessToken != "at-refreshed" {
		t.Fatalf("registered token not synced")
	}
}

func TestEnsureFreshAccessTokenSkipsWhenFarFromExpiry(t *testing.T) {
	var refreshHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	stateMu.Lock()
	state.AccessToken = "at-ok"
	state.RefreshToken = "rt-ok"
	state.AccessTokenExpiresAt = time.Now().UTC().Add(30 * time.Minute)
	state.ActiveTaskID = "task-far"
	registeredTasks = map[string]*RegisteredTask{
		"task-far": {
			TenantID: "t1", WorkspaceID: "w1", TaskID: "task-far",
			TaskAPIOrigin: server.URL, AccessToken: "at-ok",
		},
	}
	stateMu.Unlock()
	defer func() {
		stateMu.Lock()
		state.AccessToken = ""
		state.RefreshToken = ""
		state.AccessTokenExpiresAt = time.Time{}
		state.ActiveTaskID = ""
		registeredTasks = make(map[string]*RegisteredTask)
		stateMu.Unlock()
	}()

	if err := ensureFreshAccessToken(time.Now().UTC()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&refreshHits) != 0 {
		t.Fatalf("expected no refresh HTTP, got %d", refreshHits)
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if state.AccessToken != "at-ok" {
		t.Fatalf("token should be unchanged, got %q", state.AccessToken)
	}
}

func TestEnsureFreshAccessTokenKeepsTokenOnRefreshFailure(t *testing.T) {
	var refreshHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"detail":"boom"}`))
	}))
	defer server.Close()

	stateMu.Lock()
	state.AccessToken = "at-keep"
	state.RefreshToken = "rt-bad"
	state.AccessTokenExpiresAt = time.Now().UTC().Add(time.Minute)
	state.ActiveTaskID = "task-fail"
	registeredTasks = map[string]*RegisteredTask{
		"task-fail": {
			TenantID: "t1", WorkspaceID: "w1", TaskID: "task-fail", CommentID: "cmt_fail",
			TaskAPIOrigin: server.URL, AccessToken: "at-keep",
		},
	}
	stateMu.Unlock()
	defer func() {
		stateMu.Lock()
		state.AccessToken = ""
		state.RefreshToken = ""
		state.AccessTokenExpiresAt = time.Time{}
		state.ActiveTaskID = ""
		registeredTasks = make(map[string]*RegisteredTask)
		stateMu.Unlock()
	}()

	err := ensureFreshAccessToken(time.Now().UTC())
	if err == nil {
		t.Fatal("expected refresh failure")
	}
	if atomic.LoadInt32(&refreshHits) < 1 {
		t.Fatal("expected refresh attempt")
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if state.AccessToken != "at-keep" {
		t.Fatalf("AccessToken must stay at-keep, got %q", state.AccessToken)
	}
}
