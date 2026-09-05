package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeGitOauthResolveErrorUnit(t *testing.T) {
	got := sanitizeGitOauthResolveError("400 Client Error: Bad Request for url: http://127.0.0.1:8012/oauth/token", 502)
	if got == "" || got == "400 Client Error: Bad Request for url: http://127.0.0.1:8012/oauth/token" {
		t.Fatalf("expected friendly message, got %q", got)
	}
	if sanitizeGitOauthResolveError("not_found", 404) != "" {
		t.Fatal("404 not_found should be suppressed")
	}
	if got := sanitizeGitOauthResolveError(`Post "https://github.com/login/oauth/access_token": http2: timeout awaiting response headers`, 502); got != "gitOauth internal timeout" {
		t.Fatalf("timeout sanitize = %q", got)
	}
	gotGLTimeout := sanitizeGitOauthResolveError(
		`Post "https://gitlab-tencent-sh-1.daydaymoney.com/oauth/token": net/http: timeout awaiting response headers`,
		502,
	)
	if gotGLTimeout != "gitOauth internal timeout" {
		t.Fatalf("gitlab token URL timeout must not be classified as invalid grant, got %q", gotGLTimeout)
	}
}

func TestNestedAuthFailureMessageClasses(t *testing.T) {
	if got := nestedAuthFailureMessage(""); !strings.Contains(got, "未检测到可用授权") {
		t.Fatalf("unbound = %q", got)
	}
	if got := nestedAuthFailureMessage("gitOauth internal timeout"); !strings.Contains(got, "授权刷新超时") {
		t.Fatalf("timeout = %q", got)
	}
	if got := nestedAuthFailureMessage("http 502"); !strings.Contains(got, "授权服务暂时不可用") {
		t.Fatalf("service = %q", got)
	}
	got := nestedAuthFailureMessage("gitlab refresh http 400")
	if strings.Contains(got, "gitlab refresh http 400") {
		t.Fatalf("must not leak raw gitlab refresh status, got %q", got)
	}
	if !strings.Contains(got, "重新绑定") {
		t.Fatalf("want rebind hint, got %q", got)
	}
	got = nestedAuthFailureMessage("gitlab refresh http 400: invalid_grant")
	if strings.Contains(got, "invalid_grant") || strings.Contains(got, "gitlab refresh") {
		t.Fatalf("must not leak oauth internals, got %q", got)
	}
}

func TestSanitizeGitOauthResolveErrorGitLabRefresh400(t *testing.T) {
	got := sanitizeGitOauthResolveError("gitlab refresh http 400", 502)
	if strings.Contains(got, "gitlab refresh http 400") {
		t.Fatalf("must not leak raw refresh status, got %q", got)
	}
	if !strings.Contains(got, "重新绑定") {
		t.Fatalf("want rebind hint, got %q", got)
	}
}

func TestEmptyBranchPayloadSerializesArray(t *testing.T) {
	payload := emptyBranchPayload()
	if payload.Branches == nil {
		t.Fatal("branches should be non-nil slice")
	}
}

// TestFetchGitAccessTokenForwardsTraceHeaders asserts the outbound gitoauth
// request joins the caller's trace (OPT-20260810-051): the full W3C header set
// must be forwarded, not X-Trace-Id alone (taskGitOauth 拒绝仅 X-Trace-Id 的请求).
func TestFetchGitAccessTokenForwardsTraceHeaders(t *testing.T) {
	var gotTraceID, gotTraceParent, gotParentSpan string
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-Id")
		gotTraceParent = r.Header.Get("traceparent")
		gotParentSpan = r.Header.Get("X-Parent-Span-Id")
		_, _ = w.Write([]byte(`{"access_token":"tok-abc"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	trace := map[string]string{
		"X-Trace-Id":       "trace-123456",
		"traceparent":      "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbbbb-01",
		"X-Parent-Span-Id": "span-abc",
	}
	token, tokenErr := fetchGitAccessToken(42, "https://gitlab.daydaymoney.com/group/repo.git", "", trace)
	if tokenErr != "" {
		t.Fatalf("tokenErr = %q", tokenErr)
	}
	if token != "tok-abc" {
		t.Fatalf("token = %q", token)
	}
	if gotTraceID != "trace-123456" {
		t.Errorf("X-Trace-Id = %q", gotTraceID)
	}
	if gotTraceParent != "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbbbb-01" {
		t.Errorf("traceparent = %q", gotTraceParent)
	}
	if gotParentSpan != "span-abc" {
		t.Errorf("X-Parent-Span-Id = %q", gotParentSpan)
	}
}

// TestFetchGitAccessTokenNoTraceOmitsHeaders guards the reverse case: when the
// caller has no request context (internal handler), no trace headers are sent.
func TestFetchGitAccessTokenNoTraceOmitsHeaders(t *testing.T) {
	var gotTraceID string
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-Id")
		_, _ = w.Write([]byte(`{"access_token":"tok-xyz"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	token, tokenErr := fetchGitAccessToken(42, "git@gitlab.daydaymoney.com:group/repo.git", "")
	if tokenErr != "" {
		t.Fatalf("tokenErr = %q", tokenErr)
	}
	if token != "tok-xyz" {
		t.Fatalf("token = %q", token)
	}
	if gotTraceID != "" {
		t.Errorf("X-Trace-Id should be empty when no trace passed, got %q", gotTraceID)
	}
}

// TestFetchGitAccessTokenUsesGitsitePath asserts the token exchange always goes
// through /api/internal/gitsite/{host}/oauth/access-for-user/ keyed by repo host
// (OPT-20260822-036 / OPT-20260827-036), and the body omits provider_key.
func TestFetchGitAccessTokenUsesGitsitePath(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"access_token":"ghp-tok"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	token, tokenErr := fetchGitAccessToken(42, "https://github.com/acme/demo.git", "")
	if tokenErr != "" {
		t.Fatalf("tokenErr = %q", tokenErr)
	}
	if token != "ghp-tok" {
		t.Fatalf("token = %q", token)
	}
	if gotPath != "/api/internal/gitsite/github.com/oauth/access-for-user/" {
		t.Fatalf("path = %q", gotPath)
	}
	if _, has := gotBody["provider_key"]; has {
		t.Fatalf("gitsite body must omit provider_key, got %v", gotBody)
	}
	if gotBody["user_id"] != float64(42) {
		t.Fatalf("user_id = %v want 42", gotBody["user_id"])
	}
}

func TestFetchGitAccessTokenUsesGitsitePathForGitLabHost(t *testing.T) {
	var gotPath string
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"access_token":"glpat-sh"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	token, tokenErr := fetchGitAccessToken(7, "git@gitlab-tencent-sh-1.daydaymoney.com:group/repo.git", "")
	if tokenErr != "" {
		t.Fatalf("tokenErr = %q", tokenErr)
	}
	if token != "glpat-sh" {
		t.Fatalf("token = %q", token)
	}
	if gotPath != "/api/internal/gitsite/gitlab-tencent-sh-1.daydaymoney.com/oauth/access-for-user/" {
		t.Fatalf("path = %q", gotPath)
	}
}

// TestOutboundTracePicksFirstNonEmpty asserts outboundTrace returns the first
// non-empty map in a chain, mirroring trace threading through list*/resolve*.
func TestOutboundTracePicksFirstNonEmpty(t *testing.T) {
	if got := outboundTrace(nil, map[string]string{"X-Trace-Id": "a"}, map[string]string{"X-Trace-Id": "b"}); got["X-Trace-Id"] != "a" {
		t.Fatalf("first non-empty = %v", got)
	}
	if got := outboundTrace(map[string]string{}, nil, map[string]string{"traceparent": "tp"}); got["traceparent"] != "tp" {
		t.Fatalf("skip empties = %v", got)
	}
	if got := outboundTrace(nil); got != nil {
		t.Fatalf("all empty should be nil, got %v", got)
	}
}
