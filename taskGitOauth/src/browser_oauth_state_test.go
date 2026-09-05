package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"taskGitOauth/infrastructure"
	"tracelog"
)

// signTestHS256JWT — 测试用 HS256 JWT 构造器（生产签发在 gateway，仓库内
// 只有 Parse 侧；此处按 ParseHS256JWT 的校验契约构造最小合法 token）。
func signTestHS256JWT(secret string, claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	signingInput := enc(header) + "." + enc(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + enc(mac.Sum(nil)), nil
}

func TestGithubCallbackAcceptsSignedStateWithoutCookie(t *testing.T) {
	app := testApp(t)
	app.Cfg.GithubClientID = "cid"
	app.Cfg.GithubClientSecret = "sec"
	app.Cfg.GithubRedirectURI = "https://gitoauth.api.daydaymoney.com/api/accounts/github/oauth/callback/"
	app.Cfg.FrontendBase = "https://www.daydaymoney.com"

	// No network: empty client id forces exchange_failed only after state validation.
	app.Cfg.GithubClientID = ""

	state, err := app.Sess.EncodeOAuthState(infrastructure.OAuthBrowserState{
		CSRF:        "csrf-1",
		UID:         "827923618451263488",
		Next:        "/profile/git-site-oauth/",
		Feb:         "https://www.daydaymoney.com",
		RedirectURI: app.Cfg.GithubRedirectURI,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/accounts/github/oauth/callback/?code=fake&state="+url.QueryEscape(state),
		nil,
	)
	req = req.WithContext(tracelog.ContextWithTraceID(req.Context(), "oauth-exchg-trace1"))
	rr := httptest.NewRecorder()
	app.handleGithubCallback(rr, req)

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatalf("expected redirect, status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(loc, "github=bad_state") {
		t.Fatalf("signed state without cookie must not bad_state: %s", loc)
	}
	if !strings.Contains(loc, "github=exchange_failed") {
		t.Fatalf("expected exchange_failed after state OK, got %s", loc)
	}
	if !strings.Contains(loc, "trace_id=oauth-exchg-trace1") {
		t.Fatalf("expected trace_id on OAuth failure redirect for data-traceId, got %s", loc)
	}
}

func TestGithubStartEmitsSignedState(t *testing.T) {
	app := testApp(t)
	app.Cfg.GithubClientID = "cid"
	app.Cfg.GithubClientSecret = "sec"
	app.Cfg.GithubRedirectURI = "https://gitoauth.api.daydaymoney.com/api/accounts/github/oauth/callback/"
	app.Cfg.FrontendBase = "https://www.daydaymoney.com"

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/github/oauth/start-from-gateway/?next=/profile/git-site-oauth/", nil)
	req.Header.Set("X-User-Id", "42")
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	app.handleGithubStartFromGateway(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	authURL, _ := body["authorize_url"].(string)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	state := u.Query().Get("state")
	if !strings.HasPrefix(state, "v1.") {
		t.Fatalf("expected signed state, got %q", state)
	}
	cookie := rr.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "Secure") {
		t.Fatalf("HTTPS start should set Secure cookie: %s", cookie)
	}
	decoded, signed, err := app.Sess.DecodeOAuthState(state)
	if err != nil || !signed || decoded == nil || decoded.UID != "42" {
		t.Fatalf("decode state: %+v signed=%v err=%v", decoded, signed, err)
	}
}

// TestGithubStartStateRedirectURIEqualsAuthorizeParam — v2 契约不变量：
// state 签名内 redirect_uri 必须与 authorize_url 的 redirect_uri 参数完全一致
// （同一请求同一变量，换票用 state.redirect_uri 与 GitHub 授权时登记的
// redirect_uri 匹配）。线上曾出现部署产物两者不一致（state claim 带 www、
// GitHub 参数为裸域）→ 换票 redirect_uri mismatch → 授权码即便有效也被
// GitHub 拒绝（exchange_rejected），授权无法完成。
func TestGithubStartStateRedirectURIEqualsAuthorizeParam(t *testing.T) {
	app := testApp(t)
	app.Cfg.GithubClientID = "cid"
	app.Cfg.GithubClientSecret = "sec"
	const want = "https://daydaymoney.com/redirect/gitsite/github.com/oauth/callback/"
	app.Cfg.GithubRedirectURI = want
	app.Cfg.FrontendBase = "https://www.daydaymoney.com"

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/github-app-start/?next=/profile/git-site-oauth/", nil)
	req.Header.Set("X-User-Id", "42")
	rr := httptest.NewRecorder()
	app.handleGithubStartFromGateway(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	authURL, _ := body["authorize_url"].(string)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	paramURI := u.Query().Get("redirect_uri")
	if paramURI != want {
		t.Fatalf("authorize redirect_uri=%q want %q", paramURI, want)
	}
	state := u.Query().Get("state")
	decoded, signed, err := app.Sess.DecodeOAuthState(state)
	if err != nil || !signed || decoded == nil {
		t.Fatalf("decode state: %+v signed=%v err=%v", decoded, signed, err)
	}
	if decoded.RedirectURI != paramURI {
		t.Fatalf("state claim redirect_uri=%q != authorize param %q（换票将 redirect_uri mismatch）", decoded.RedirectURI, paramURI)
	}
	if decoded.RedirectURI != want {
		t.Fatalf("state claim redirect_uri=%q want %q", decoded.RedirectURI, want)
	}
}

// TestGithubBrowserStartStateRedirectURIEqualsAuthorizeParam — 与上面同一不变量，
// 覆盖浏览器 token 启动路径（/api/git-oauth/github-start/?token=…，带签名
// bridge JWT）：state 与 authorize 参数同源一致，防止两条启动链路分叉。
func TestGithubBrowserStartStateRedirectURIEqualsAuthorizeParam(t *testing.T) {
	app := testApp(t)
	app.Cfg.GithubClientID = "cid"
	app.Cfg.GithubClientSecret = "sec"
	const want = "https://daydaymoney.com/redirect/gitsite/github.com/oauth/callback/"
	app.Cfg.GithubRedirectURI = want
	app.Cfg.FrontendBase = "https://www.daydaymoney.com"

	token, err := signTestHS256JWT(app.Cfg.BridgeJWTSecret, map[string]any{
		"iss":  app.Cfg.BridgeJWTIssuer,
		"aud":  app.Cfg.BridgeJWTAudienceGH,
		"typ":  "github_oauth_start",
		"sub":  "42",
		"next": "/profile/git-site-oauth/",
		"feb":  "https://www.daydaymoney.com",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Unix() + 600,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/github-start/?token="+url.QueryEscape(token), nil)
	rr := httptest.NewRecorder()
	app.handleGithubStart(rr, req)
	if rr.Code < 300 || rr.Code >= 400 {
		t.Fatalf("expected redirect to github, status=%d body=%s", rr.Code, rr.Body.String())
	}
	loc := rr.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	paramURI := u.Query().Get("redirect_uri")
	if paramURI != want {
		t.Fatalf("authorize redirect_uri=%q want %q", paramURI, want)
	}
	state := u.Query().Get("state")
	decoded, signed, err := app.Sess.DecodeOAuthState(state)
	if err != nil || !signed || decoded == nil {
		t.Fatalf("decode state: %+v signed=%v err=%v", decoded, signed, err)
	}
	if decoded.RedirectURI != paramURI {
		t.Fatalf("state claim redirect_uri=%q != authorize param %q（换票将 redirect_uri mismatch）", decoded.RedirectURI, paramURI)
	}
}

// TestAccountsCallbackRouteRegisteredContractPath — 回归 8c47b8a：浏览器回调契约路径
// /api/accounts/<service_provider>/oauth/callback/（SSOT redirect_uri 与 GitHub/GitLab
// App 白名单指向它）必须由 RegisterRoutes 注册。无会话访问应 302 跳前端（bad_state），
// 而不是 404 page not found。
func TestAccountsCallbackRouteRegisteredContractPath(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	for _, sp := range []string{"github", "gitlab"} {
		req := httptest.NewRequest(http.MethodGet,
			"/api/accounts/"+sp+"/oauth/callback/?code=fake&state=zzz", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("%s contract callback path must not 404: route not registered", sp)
		}
		if rec.Code < 300 || rec.Code >= 500 {
			t.Fatalf("expected 3xx handling, got %d body=%s", rec.Code, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc == "" {
			t.Fatalf("expected Location redirect for %s, code=%d", sp, rec.Code)
		}
	}
}

// TestGitsiteCallbackRouteRegisteredContractPath — v2 浏览器回调契约路径
// /redirect/gitsite/<gitsite>/oauth/callback/（SSOT redirect_uri 指向它）必须由
// RegisterRoutes 注册：已知 gitsite（github.com / localhost）无会话访问应 302 跳
// 前端（bad_state），未知 gitsite 应 404 JSON；均不得 404 page not found。
func TestGitsiteCallbackRouteRegisteredContractPath(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	for _, gitsite := range []string{"github.com", "localhost"} {
		req := httptest.NewRequest(http.MethodGet,
			"/redirect/gitsite/"+gitsite+"/oauth/callback/?code=fake&state=zzz", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("gitsite=%s contract callback path must not 404: route not registered", gitsite)
		}
		if rec.Code < 300 || rec.Code >= 500 {
			t.Fatalf("expected 3xx handling for gitsite=%s, got %d body=%s", gitsite, rec.Code, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc == "" {
			t.Fatalf("expected Location redirect for gitsite=%s, code=%d", gitsite, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet,
		"/redirect/gitsite/unknown.example.org/oauth/callback/?code=fake&state=zzz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown gitsite should 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestGithubCallbackExchangeErrorClassification — 回归 2026-08-07：换票失败分类。
// GitHub 明确拒绝（GitHubExchangeRejectedError）→ 302 带 github=exchange_rejected；
// 网络类失败 → github=exchange_failed。前端据此分级提示。
func TestGithubCallbackExchangeErrorClassification(t *testing.T) {
	state, err := testApp(t).Sess.EncodeOAuthState(infrastructure.OAuthBrowserState{
		CSRF: "csrf-x", UID: "873093522473906176", Next: "/profile/git-site-oauth/",
		Feb: "https://www.daydaymoney.com", RedirectURI: "https://gitoauth_api.daydaymoney.com/api/accounts/github/oauth/callback/",
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		err     error
		wantKey string
	}{
		{"rejected", &infrastructure.GitHubExchangeRejectedError{StatusCode: 422}, "exchange_rejected"},
		{"network", errors.New("dial tcp github.com:443: connection refused"), "exchange_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := providerTestApp(t)
			app.ExchangeGitHubCodeFn = func(*infrastructure.Config, string, string) (map[string]any, error) {
				return nil, tc.err
			}
			req := httptest.NewRequest(http.MethodGet,
				"/api/accounts/github/oauth/callback/?code=fake&state="+url.QueryEscape(state), nil)
			rec := httptest.NewRecorder()
			app.handleGithubCallback(rec, req)
			loc := rec.Header().Get("Location")
			if !strings.Contains(loc, "github="+tc.wantKey) {
				t.Fatalf("want github=%s in redirect, got %s", tc.wantKey, loc)
			}
		})
	}
}

// TestFlattenedCallbackRoutesStillRegistered — 扁平化约定路径
// /api/git-oauth/{provider}-callback/ 仍须注册（内部/网关调用方）。
func TestFlattenedCallbackRoutesStillRegistered(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	for _, path := range []string{"/api/git-oauth/github-callback/", "/api/git-oauth/gitlab-callback/"} {
		req := httptest.NewRequest(http.MethodGet, path+"?code=fake&state=zzz", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("%s must not 404", path)
		}
	}
}

// TestGithubStartFromGatewayAcceptsNonNumericUserID — 回归 2026-08-07：
// taskAuth auth_user.id 为 varchar(36)，确定性 ID（bootstrap-admin）非数字。
// start-from-gateway 此前 parsePositiveInt 拒绝 → 503 bad_x_user_id，
// 前端所有用户点击「OAuth 授权」必失败。非空字符串 user ID 必须放行。
func TestGithubStartFromGatewayAcceptsNonNumericUserID(t *testing.T) {
	app := providerTestApp(t)
	app.Cfg.GithubClientID = "Iv23liEA2c4007xdQGXJ"
	app.Cfg.GithubRedirectURI = "https://gitoauth_api.daydaymoney.com/api/accounts/github/oauth/callback/"
	app.Cfg.FrontendBase = "https://www.daydaymoney.com"

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/github-start-from-gateway/?next=/profile/git-site-oauth/", nil)
	req.Header.Set("X-User-Id", "bootstrap-admin")
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	app.handleGithubStartFromGateway(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("bootstrap-admin start: status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	authURL, _ := body["authorize_url"].(string)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	state := u.Query().Get("state")
	decoded, signed, err := app.Sess.DecodeOAuthState(state)
	if err != nil || !signed || decoded == nil || decoded.UID != "bootstrap-admin" {
		t.Fatalf("state uid must carry bootstrap-admin verbatim: %+v signed=%v err=%v", decoded, signed, err)
	}
}

func TestGithubStartFromGatewayRedirectsWhenBrowserAcceptsHTML(t *testing.T) {
	app := testApp(t)
	app.Cfg.GithubClientID = "cid"
	app.Cfg.GithubClientSecret = "sec"
	app.Cfg.GithubRedirectURI = "https://gitoauth.api.daydaymoney.com/api/accounts/github/oauth/callback/"
	app.Cfg.FrontendBase = "https://www.daydaymoney.com"

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/github-start-from-gateway/?repo_url="+url.QueryEscape("https://github.com/acme/demo.git")+"&next=/task-detail/",
		nil)
	req.Header.Set("X-User-Id", "42")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rr := httptest.NewRecorder()
	app.handleGithubStartFromGateway(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "https://github.com/login/oauth/authorize?") {
		t.Fatalf("Location=%q", loc)
	}
}

// TestGithubCallbackNonNumericUIDReachesExchange — bootstrap-admin 的 signed state
// 必须在 callback 中穿过 uid 校验到达换票阶段（错误分类为 exchange_rejected 而非
// bad_state），证明非数字 uid 在完整 OAuth 流程中可用。
func TestGithubCallbackNonNumericUIDReachesExchange(t *testing.T) {
	state, err := testApp(t).Sess.EncodeOAuthState(infrastructure.OAuthBrowserState{
		CSRF: "csrf-x", UID: "bootstrap-admin", Next: "/profile/git-site-oauth/",
		Feb: "https://www.daydaymoney.com", RedirectURI: "https://gitoauth_api.daydaymoney.com/api/accounts/github/oauth/callback/",
	})
	if err != nil {
		t.Fatal(err)
	}
	app := providerTestApp(t)
	app.ExchangeGitHubCodeFn = func(*infrastructure.Config, string, string) (map[string]any, error) {
		return nil, &infrastructure.GitHubExchangeRejectedError{StatusCode: 422}
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/accounts/github/oauth/callback/?code=fake&state="+url.QueryEscape(state), nil)
	rec := httptest.NewRecorder()
	app.handleGithubCallback(rec, req)
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "github=exchange_rejected") {
		t.Fatalf("bootstrap-admin callback must reach exchange stage (exchange_rejected), got %s", loc)
	}
	if strings.Contains(loc, "github=bad_state") {
		t.Fatalf("non-numeric uid must not be rejected as bad_state: %s", loc)
	}
}
