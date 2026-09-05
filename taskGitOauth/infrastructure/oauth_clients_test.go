package infrastructure

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseGitHubTokenBodySurfacesOAuthError(t *testing.T) {
	_, err := parseGitHubTokenBody(
		[]byte(`{"error":"bad_verification_code","error_description":"The code passed is incorrect or expired."}`),
		"application/json",
	)
	if err == nil {
		t.Fatal("expected error for GitHub OAuth error body")
	}
	if !strings.Contains(err.Error(), "bad_verification_code") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseGitHubTokenBodyOK(t *testing.T) {
	out, err := parseGitHubTokenBody(
		[]byte(`{"access_token":"gho_x","refresh_token":"ghr_y","scope":"repo"}`),
		"application/json",
	)
	if err != nil {
		t.Fatal(err)
	}
	if out["access_token"] != "gho_x" || out["refresh_token"] != "ghr_y" {
		t.Fatalf("out=%v", out)
	}
}

// TestGitHubExchangeRejectedErrorClassification — 回归 2026-08-07：换票失败须区分
// 「GitHub 明确拒绝」（HTTP 4xx / 200+OAuth error body → exchange_rejected）与
// 「网络类失败」（直连/代理不可达 → exchange_failed），前端据此分级提示。
func TestGitHubExchangeRejectedErrorClassification(t *testing.T) {
	rej := &GitHubExchangeRejectedError{StatusCode: 422, Body: `{"error":"bad_verification_code"}`}
	if rej.StatusCode != 422 {
		t.Fatalf("status=%d", rej.StatusCode)
	}
	if err := (&GitHubExchangeRejectedError{StatusCode: 400}).Error(); err == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TestExchangeGitHubCodeRejectsHTTP400 — 用假 Transport 注入 4xx 响应，验证返回
// GitHubExchangeRejectedError（handler 据此走 exchange_rejected 而非 exchange_failed）。
func TestExchangeGitHubCodeRejectsHTTP400(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 400,
			Status:     "400 Bad Request",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"bad_verification_code"}`)),
			Request:    r,
		}, nil
	})}
	cfg := &Config{GithubClientID: "cid", GithubClientSecret: "sec", GithubRedirectURI: "https://x/callback/"}
	_, err := ExchangeGitHubCode(cfg, "code", "https://x/callback/")
	var rej *GitHubExchangeRejectedError
	if !errors.As(err, &rej) {
		t.Fatalf("want GitHubExchangeRejectedError, got %v", err)
	}
	if rej.StatusCode != 400 {
		t.Fatalf("status=%d", rej.StatusCode)
	}
}

// TestExchangeGitHubCodeNetworkErrorFirstErrKept — 直连错误优先于回退代理错误：
// 即使后续 client 也失败，返回首个错误（可观测性：日志不再只见 socks 错误）。
// 注意：配置了 outbound_proxy 时直连走 httpClientFastFail（非 httpClient），且
// 回退代理客户端经 proxyClientCache 缓存 —— 两者都必须注入 mock，否则测试会
// 真实出网（曾致 2026-08-07 环境依赖失败：直连可达时 GitHub 对无效 cid 返回
// HTTP 404，覆盖了 first-error 断言）。
func TestExchangeGitHubCodeNetworkErrorFirstErrKept(t *testing.T) {
	oldClient, oldFast := httpClient, httpClientFastFail
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("http2: timeout awaiting response headers")
	})}
	httpClientFastFail = httpClient // 有 proxy 时直连用 fastFail
	const proxyURL = "socks5://127.0.0.1:1080"
	proxyClientMu.Lock()
	proxyClientCache[proxyURL] = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("socks5 dial: connection refused") // 死代理
	})}
	proxyClientMu.Unlock()
	defer func() {
		httpClient, httpClientFastFail = oldClient, oldFast
		proxyClientMu.Lock()
		delete(proxyClientCache, proxyURL)
		proxyClientMu.Unlock()
	}()

	cfg := &Config{
		GithubClientID: "cid", GithubClientSecret: "sec", GithubRedirectURI: "https://x/callback/",
		GithubOutboundProxy: proxyURL,
	}
	_, err := ExchangeGitHubCode(cfg, "code", "https://x/callback/")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http2: timeout") {
		t.Fatalf("want first (direct) error preserved, got %v", err)
	}
}

// TestRefreshGitHubTokenRetriesAfterHeaderTimeout — 回归 2026-08-17：
// access-for-user 走 Refresh（非 Exchange）。边缘 IP「TCP 通但 HTTP 挂起」时
// 旧实现只试一次 → nested-git / auto_run 直接软跳过。须在轮转 dial 后重试。
func TestRefreshGitHubTokenRetriesAfterHeaderTimeout(t *testing.T) {
	old := httpClient
	t.Cleanup(func() { httpClient = old })
	calls := 0
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("net/http: timeout awaiting response headers")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"access_token":"gho_ok","refresh_token":"ghr_ok","expires_in":28800}`)),
			Request:    r,
		}, nil
	})}
	cfg := &Config{GithubClientID: "cid", GithubClientSecret: "sec"}
	out, err := RefreshGitHubToken(cfg, "ghr_old")
	if err != nil {
		t.Fatalf("want success after retry, got %v", err)
	}
	if out["access_token"] != "gho_ok" {
		t.Fatalf("out=%v", out)
	}
	if calls < 2 {
		t.Fatalf("want ≥2 attempts (timeout then success), calls=%d", calls)
	}
}

// TestFetchGitHubProfileRetriesAfterHeaderTimeout — 回归 OPT-20260817-037：
// FetchGitHubProfile 与 RefreshGitHubToken 对齐，无 proxy 直连时对挂起边缘 IP 多轮重试，
// 首轮 timeout 后次轮成功（旧实现单 client 只试一次即失败）。
func TestFetchGitHubProfileRetriesAfterHeaderTimeout(t *testing.T) {
	old := httpClient
	t.Cleanup(func() { httpClient = old })
	calls := 0
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("net/http: timeout awaiting response headers")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"login":"octocat","id":1}`)),
			Request:    r,
		}, nil
	})}
	cfg := &Config{}
	out, err := FetchGitHubProfile(cfg, "gho_old")
	if err != nil {
		t.Fatalf("want success after retry, got %v", err)
	}
	if out["login"] != "octocat" {
		t.Fatalf("out=%v", out)
	}
	if calls < 2 {
		t.Fatalf("want ≥2 attempts (timeout then success), calls=%d", calls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestExchangeGitLabCodeRejectClassification — 回归 2026-08-08（OPT-20260807-029）：
// GitLab 换票「明确拒绝」（4xx 或 200+error body）须返回 GitLabExchangeRejectedError，
// 网络类失败返回普通 error —— handler 据此分类 exchange_rejected / exchange_failed。
func TestExchangeGitLabCodeRejectClassification(t *testing.T) {
	pc := &ProviderConfig{
		Provider:        "gitlab",
		ServiceProvider: "gitlab-local",
		ProviderKey:     "gitlab:gitlab-local",
		Website:         "https://gitlab.example",
		ClientID:        "cid",
		ClientSecret:    "sec",
		RedirectURI:     "https://gitoauth.example/api/accounts/gitlab/oauth/callback/",
	}
	withTransport := func(t *testing.T, transport http.RoundTripper) {
		t.Helper()
		old := gitlabHTTPClient
		gitlabHTTPClient = &http.Client{Transport: transport}
		t.Cleanup(func() { gitlabHTTPClient = old })
	}

	t.Run("http_400_rejected", func(t *testing.T) {
		withTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "400 Bad Request",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`)),
				Request:    r,
			}, nil
		}))
		_, err := ExchangeGitLabCode(pc, "code", pc.RedirectURI)
		var rej *GitLabExchangeRejectedError
		if !errors.As(err, &rej) {
			t.Fatalf("want GitLabExchangeRejectedError, got %v", err)
		}
		if rej.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d", rej.StatusCode)
		}
	})

	t.Run("http_200_with_error_body_rejected", func(t *testing.T) {
		withTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`)),
				Request:    r,
			}, nil
		}))
		_, err := ExchangeGitLabCode(pc, "code", pc.RedirectURI)
		var rej *GitLabExchangeRejectedError
		if !errors.As(err, &rej) {
			t.Fatalf("want GitLabExchangeRejectedError for 200+error body, got %v", err)
		}
	})

	t.Run("http_200_ok", func(t *testing.T) {
		withTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"glpat_x","refresh_token":"glrt_y"}`)),
				Request:    r,
			}, nil
		}))
		out, err := ExchangeGitLabCode(pc, "code", pc.RedirectURI)
		if err != nil {
			t.Fatal(err)
		}
		if out["access_token"] != "glpat_x" || out["refresh_token"] != "glrt_y" {
			t.Fatalf("out=%v", out)
		}
	})

	t.Run("network_error_not_rejected", func(t *testing.T) {
		withTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp gitlab.example:443: connection refused")
		}))
		_, err := ExchangeGitLabCode(pc, "code", pc.RedirectURI)
		if err == nil {
			t.Fatal("expected error")
		}
		var rej *GitLabExchangeRejectedError
		if errors.As(err, &rej) {
			t.Fatalf("network error must NOT be rejected, got %v", err)
		}
	})
}

func gitlabRefreshTestProvider() *ProviderConfig {
	return &ProviderConfig{
		Provider:        "gitlab",
		ServiceProvider: "tencent-sh-1",
		ProviderKey:     "gitlab:tencent-sh-1",
		Website:         "https://gitlab-tencent-sh-1.example",
		ClientID:        "cid-gl",
		ClientSecret:    "sec-gl",
		RedirectURI:     "https://app.example/redirect/gitsite/gitlab-tencent-sh-1.example/oauth/callback/",
	}
}

// TestRefreshGitLabTokenSendsOfficialForm 回归：GitLab 官方 refresh 须带
// client_id + client_secret + grant_type + refresh_token + redirect_uri。
// 缺 redirect_uri / client_secret 时 GitLab Doorkeeper 返回 400 invalid_grant，
// 前端表现为「无法获取子 Git 仓库列表：gitlab refresh http 400」。
func TestRefreshGitLabTokenSendsOfficialForm(t *testing.T) {
	pc := gitlabRefreshTestProvider()
	old := gitlabHTTPClient
	t.Cleanup(func() { gitlabHTTPClient = old })
	var gotBody, gotContentType string
	gitlabHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		gotContentType = r.Header.Get("Content-Type")
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"access_token":"glpat_new","refresh_token":"glrt_new"}`)),
			Request:    r,
		}, nil
	})}
	out, err := RefreshGitLabToken(pc, "glrt_old")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if out["access_token"] != "glpat_new" {
		t.Fatalf("out=%v", out)
	}
	if !strings.Contains(gotContentType, "application/x-www-form-urlencoded") {
		t.Fatalf("content-type=%q", gotContentType)
	}
	vals, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatalf("parse form: %v body=%q", err, gotBody)
	}
	for key, want := range map[string]string{
		"client_id":     pc.ClientID,
		"client_secret": pc.ClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": "glrt_old",
		"redirect_uri":  pc.RedirectURI,
	} {
		if got := vals.Get(key); got != want {
			t.Errorf("form %s=%q want %q", key, got, want)
		}
	}
}

// TestRefreshGitLabTokenRequiresRedirectURI 回归：缺 redirect_uri 时 GitLab Doorkeeper
// 返回 400 invalid_grant（前端「gitlab refresh http 400」）。禁止静默省略该字段。
func TestRefreshGitLabTokenRequiresRedirectURI(t *testing.T) {
	pc := gitlabRefreshTestProvider()
	pc.RedirectURI = "  "
	old := gitlabHTTPClient
	t.Cleanup(func() { gitlabHTTPClient = old })
	gitlabHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Fatal("must not POST /oauth/token when redirect_uri is empty")
		return nil, errors.New("unreachable")
	})}
	_, err := RefreshGitLabToken(pc, "glrt_old")
	if err == nil {
		t.Fatal("expected error for empty redirect_uri")
	}
	if !strings.Contains(err.Error(), "redirect_uri") {
		t.Fatalf("want redirect_uri in error, got %q", err.Error())
	}
}

func TestRefreshGitLabTokenHTTP400IncludesOAuthErrorCode(t *testing.T) {
	pc := gitlabRefreshTestProvider()
	old := gitlabHTTPClient
	t.Cleanup(func() { gitlabHTTPClient = old })
	gitlabHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"error":"invalid_grant","error_description":"token glrt_secret_do_not_leak is expired"}`,
			)),
			Request: r,
		}, nil
	})}
	_, err := RefreshGitLabToken(pc, "glrt_secret_do_not_leak")
	if err == nil {
		t.Fatal("expected refresh error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "gitlab refresh http 400") {
		t.Fatalf("want status in error, got %q", msg)
	}
	if !strings.Contains(msg, "invalid_grant") {
		t.Fatalf("want oauth error code, got %q", msg)
	}
	if strings.Contains(msg, "glrt_secret_do_not_leak") {
		t.Fatalf("must not leak refresh token, got %q", msg)
	}
}

// TestRefreshGitLabTokenCompletesAfterSlowHeaders 回归：GitLab /oauth/token 响应头
// 超过 GitHub 用的 4s ResponseHeaderTimeout 时，旧共用 client 超时，但 Doorkeeper
// 已旋转 refresh；须用独立 25s header timeout 等到 JSON。
func TestRefreshGitLabTokenCompletesAfterSlowHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}
		time.Sleep(5 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"glpat_slow","refresh_token":"glrt_slow","expires_in":7200}`))
	}))
	t.Cleanup(srv.Close)
	pc := gitlabRefreshTestProvider()
	pc.Website = srv.URL
	out, err := RefreshGitLabToken(pc, "glrt_old")
	if err != nil {
		t.Fatalf("slow gitlab refresh: %v", err)
	}
	if out["access_token"] != "glpat_slow" || out["refresh_token"] != "glrt_slow" {
		t.Fatalf("out=%v", out)
	}
}
