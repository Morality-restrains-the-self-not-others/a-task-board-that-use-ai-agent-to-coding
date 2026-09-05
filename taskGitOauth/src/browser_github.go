package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func (a *App) githubStoredProviderKey() string {
	rows := a.Cfg.GetProviderConfigs("github")
	if len(rows) > 0 {
		if pk := strings.TrimSpace(strings.ToLower(rows[0].ProviderKey)); pk != "" {
			return pk
		}
		if sp := strings.TrimSpace(strings.ToLower(rows[0].ServiceProvider)); sp != "" {
			return domain.ProviderKey(githubProvider, sp)
		}
	}
	return domain.ProviderKey(githubProvider, "github-official")
}

// githubStoredProviderKeys returns every configured github storage key, mirroring the
// gitlab branch of userAppConnectionLookupKeys. When multiple github providers are
// configured (different service_provider), the status check must cover all of them,
// not just rows[0]. Deduplicated to keep the lookup key set minimal.
func (a *App) githubStoredProviderKeys() []string {
	var keys []string
	seen := map[string]bool{}
	add := func(k string) {
		k = strings.TrimSpace(strings.ToLower(k))
		if k != "" && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for _, row := range a.Cfg.GetProviderConfigs("github") {
		add(row.ProviderKey)
		add(domain.ProviderKey(githubProvider, row.ServiceProvider))
	}
	return keys
}

func (a *App) handleGithubStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	feb := ""
	defer func() {
		if rec := recover(); rec != nil {
			logWarn("GitHub OAuth start panic: %v", rec)
			a.frontendOAuthErrorRedirect(w, r, feb, "/profile/git-site-oauth/?github=bad_state")
		}
	}()
	raw := strings.TrimSpace(r.URL.Query().Get("token"))
	if raw == "" {
		a.frontendOAuthErrorRedirect(w, r, "", "/profile/git-site-oauth/?github=bad_state")
		return
	}
	claims, err := a.decodeStartJWT(raw, "github_oauth_start", a.Cfg.BridgeJWTAudienceGH)
	if err != nil {
		logWarn("GitHub OAuth start token: %v", err)
		a.frontendOAuthErrorRedirect(w, r, "", "/profile/git-site-oauth/?github=bad_state")
		return
	}
	// taskAuth auth_user.id 为 varchar(36)，确定性 ID（如 bootstrap-admin）非数字：
	// uid 全链路按字符串传递（2026-08-07 回归，OPT-20260807-041）。
	uid := strings.TrimSpace(fmt.Sprint(claims["sub"]))
	if uid == "" {
		a.frontendOAuthErrorRedirect(w, r, "", "/profile/git-site-oauth/?github=bad_state")
		return
	}
	nextSafe := sanitizeAppNext(fmt.Sprint(claims["next"]))
	rk := sanitizeReturnKey(fmt.Sprint(claims["rk"]))
	feb = strings.TrimSpace(fmt.Sprint(claims["feb"]))
	csrf := infrastructure.RandomTokenURLSafe(24)
	redirectURI := infrastructure.ResolveOAuthRedirectURI(
		a.Cfg.GithubRedirectURI, "github", feb, a.Cfg.PublicEntryOrigins,
	)
	cid := a.Cfg.GithubClientID
	if cid == "" || redirectURI == "" {
		logWarn("gitOauth 未配置 GITHUB_APP_CLIENT_ID / GITHUB_APP_REDIRECT_URI")
		a.redirectGithubContinue(w, r, feb, rk, "bad_state", "")
		return
	}
	state, err := a.Sess.EncodeOAuthState(infrastructure.OAuthBrowserState{
		CSRF: csrf, UID: uid, Next: nextSafe, RK: rk, Feb: feb, RedirectURI: redirectURI,
	})
	if err != nil {
		logWarn("GitHub OAuth encode state: %v", err)
		a.redirectGithubContinue(w, r, feb, rk, "bad_state", "")
		return
	}
	sess := map[string]any{
		githubSessionKey: map[string]any{
			"csrf": csrf, "uid": uid, "next": nextSafe, "rk": rk, "feb": feb, "redirect_uri": redirectURI,
		},
	}
	// merge with existing session keys
	existing := a.Sess.Get(r)
	for k, v := range existing {
		if k != githubSessionKey {
			sess[k] = v
		}
	}
	a.Sess.SetForRequest(w, r, sess)
	params := url.Values{}
	params.Set("client_id", cid)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", a.Cfg.GithubScope)
	params.Set("state", state)
	redirect(w, r, "https://github.com/login/oauth/authorize?"+params.Encode())
}

func (a *App) handleGithubStartFromGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uid := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if uid == "" {
		a.gatewayStartError(w, "GitHub", "missing_x_user_id", "")
		return
	}
	nextSafe := sanitizeAppNext(r.URL.Query().Get("next"))
	rk := sanitizeReturnKey(r.URL.Query().Get("return_key"))
	feb := a.febFromRequest(r, a.Cfg.FrontendBase)
	csrf := infrastructure.RandomTokenURLSafe(24)
	redirectURI := infrastructure.ResolveOAuthRedirectURI(
		a.Cfg.GithubRedirectURI, "github", feb, a.Cfg.PublicEntryOrigins,
	)
	cid := a.Cfg.GithubClientID
	if cid == "" || redirectURI == "" {
		a.gatewayStartError(w, "GitHub", "bad_state", "missing_client_id_or_redirect_uri")
		return
	}
	st := infrastructure.OAuthBrowserState{
		CSRF: csrf, UID: uid, Next: nextSafe, RK: rk, Feb: feb, RedirectURI: redirectURI,
	}
	applyOAuthGrantQuery(&st, r)
	state, err := a.Sess.EncodeOAuthState(st)
	if err != nil {
		a.gatewayStartError(w, "GitHub", "bad_state", "encode_state_failed")
		return
	}
	sess := a.Sess.Get(r)
	sess[githubSessionKey] = map[string]any{
		"csrf": csrf, "uid": uid, "next": nextSafe, "rk": rk, "feb": feb, "redirect_uri": redirectURI,
	}
	a.Sess.SetForRequest(w, r, sess)
	params := url.Values{}
	params.Set("client_id", cid)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", a.Cfg.GithubScope)
	params.Set("state", state)
	writeOAuthStartResponse(w, r, "https://github.com/login/oauth/authorize?"+params.Encode())
}

func (a *App) handleGithubCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.handleGithubCallbackWithSP(w, r, "")
}

func (a *App) handleGithubCallbackWithSP(w http.ResponseWriter, r *http.Request, serviceProvider string) {
	sessAll := a.Sess.Get(r)
	ctx, _ := sessAll[githubSessionKey].(map[string]any)
	if ctx == nil {
		ctx = map[string]any{}
	}
	feb := sessionGetString(ctx, "feb")
	rk := sessionGetString(ctx, "rk")
	nextSafe := sanitizeAppNext(sessionGetString(ctx, "next"))
	urlSP := strings.TrimSpace(strings.ToLower(serviceProvider))
	ctxSP := strings.TrimSpace(strings.ToLower(sessionGetString(ctx, "service_provider")))
	clearAnd := func(code string) {
		delete(sessAll, githubSessionKey)
		a.Sess.SetForRequest(w, r, sessAll)
		a.redirectGithubContinue(w, r, feb, rk, code, "")
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		clearAnd("bad_state")
		return
	}
	signed, isSigned, err := a.Sess.DecodeOAuthState(state)
	if err != nil {
		logWarn("GitHub OAuth state: %v", err)
		clearAnd("bad_state")
		return
	}
	if isSigned && signed != nil {
		ctx = map[string]any{
			"csrf": signed.CSRF, "uid": signed.UID, "next": signed.Next, "rk": signed.RK,
			"feb": signed.Feb, "redirect_uri": signed.RedirectURI,
			"service_provider": signed.ServiceProvider,
		}
		feb = signed.Feb
		rk = signed.RK
		nextSafe = sanitizeAppNext(signed.Next)
		ctxSP = strings.TrimSpace(strings.ToLower(signed.ServiceProvider))
	} else {
		// Legacy: plain CSRF in state requires matching session cookie.
		if len(ctx) == 0 || state != sessionGetString(ctx, "csrf") {
			clearAnd("bad_state")
			return
		}
	}
	if urlSP != "" && ctxSP != "" && urlSP != ctxSP {
		clearAnd("bad_state")
		return
	}
	uid := strings.TrimSpace(sessionGetString(ctx, "uid"))
	if uid == "" {
		clearAnd("bad_state")
		return
	}
	exchangeFn := a.ExchangeGitHubCodeFn
	if exchangeFn == nil {
		exchangeFn = infrastructure.ExchangeGitHubCode
	}
	data, err := exchangeFn(a.Cfg, code, sessionGetString(ctx, "redirect_uri"))
	if err != nil {
		logWarn("GitHub 回调 authorization_code 换票失败: %v", err)
		// 分类提示：GitHub 明确拒绝（code 过期/redirect_uri 不匹配）→ exchange_rejected；
		// 网络类失败（直连/代理不可达）→ exchange_failed。
		var rej *infrastructure.GitHubExchangeRejectedError
		if errors.As(err, &rej) {
			clearAnd("exchange_rejected")
		} else {
			clearAnd("exchange_failed")
		}
		return
	}
	refresh := strings.TrimSpace(fmt.Sprint(data["refresh_token"]))
	access := strings.TrimSpace(fmt.Sprint(data["access_token"]))
	if refresh == "" {
		clearAnd("no_refresh")
		return
	}
	if access == "" {
		clearAnd("no_access")
		return
	}
	profile, err := infrastructure.FetchGitHubProfile(a.Cfg, access)
	if err != nil {
		clearAnd("profile_failed")
		return
	}
	ghUID := infrastructure.FormatRemoteUserID(profile["id"])
	if ghUID == "" {
		clearAnd("no_gh_id")
		return
	}
	ghLogin := truncate(fmt.Sprint(profile["login"]), 255)
	cipher, err := a.Fernet.Encrypt(refresh)
	if err != nil || cipher == "" {
		clearAnd("exchange_failed")
		return
	}
	storedPK := a.githubStoredProviderKey()
	_, err = a.DB.UpsertCredential(&infrastructure.CredentialRow{
		Provider:           storedPK,
		Task2appUserID:     uid,
		RefreshTokenCipher: cipher,
		RemoteUserID:       ghUID,
		RemoteLogin:        ghLogin,
		Scope:              truncate(fmt.Sprint(data["scope"]), 512),
		BindStatus:         domain.BindActive,
		BindError:          "",
	})
	if err != nil {
		logWarn("upsert credential: %v", err)
		clearAnd("exchange_failed")
		return
	}
	a.insertAccessAuditForProvider(storedPK, uid, "oauth_callback_token_issued",
		accessTokenFingerprint(access), map[string]any{
			"remote_user_id": ghUID,
			"remote_login":   truncate(ghLogin, 128),
		})

	nextSafe, grantTicket := a.finishResourceGrant(uid, ghUID, signed, nextSafe)

	delete(sessAll, githubSessionKey)
	a.Sess.SetForRequest(w, r, sessAll)
	rkS := sanitizeReturnKey(rk)
	if rkS != "" && nextSafe != "" {
		redirect(w, r, a.frontendRedirect(feb, appendQueryParam(pathWithParamOK(nextSafe, "github"), "grant_ticket", grantTicket)))
		return
	}
	if rkS != "" {
		a.redirectGithubContinue(w, r, feb, rkS, "ok", grantTicket)
		return
	}
	if nextSafe != "" {
		redirect(w, r, a.frontendRedirect(feb, appendQueryParam(pathWithParamOK(nextSafe, "github"), "grant_ticket", grantTicket)))
		return
	}
	redirect(w, r, a.frontendRedirect(feb, appendQueryParam("/profile/git-site-oauth/?github=ok", "grant_ticket", grantTicket)))
}

func (a *App) redirectGithubContinue(w http.ResponseWriter, r *http.Request, feb, returnKey, code, grantTicket string) {
	rk := sanitizeReturnKey(returnKey)
	var path string
	if rk != "" {
		q := url.Values{"returnKey": {rk}, "github": {code}}
		if strings.TrimSpace(grantTicket) != "" {
			q.Set("grant_ticket", grantTicket)
		}
		path = "/oauth/github-app/callback/?" + q.Encode()
	} else {
		path = "/profile/git-site-oauth/?github=" + url.QueryEscape(code)
		path = appendQueryParam(path, "grant_ticket", grantTicket)
	}
	if code != "ok" {
		a.frontendOAuthErrorRedirect(w, r, feb, path)
		return
	}
	redirect(w, r, a.frontendRedirect(feb, path))
}
