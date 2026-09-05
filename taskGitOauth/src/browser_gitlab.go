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

func (a *App) handleGitlabStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	feb := ""
	raw := strings.TrimSpace(r.URL.Query().Get("token"))
	if raw == "" {
		a.frontendOAuthErrorRedirect(w, r, "", "/profile/git-site-oauth/?gitlab=bad_state")
		return
	}
	claims, err := a.decodeStartJWT(raw, "gitlab_oauth_start", a.Cfg.BridgeJWTAudienceGL)
	if err != nil {
		logWarn("GitLab OAuth start token: %v", err)
		a.frontendOAuthErrorRedirect(w, r, "", "/profile/git-site-oauth/?gitlab=bad_state")
		return
	}
	uid := strings.TrimSpace(fmt.Sprint(claims["sub"]))
	if uid == "" {
		a.frontendOAuthErrorRedirect(w, r, "", "/profile/git-site-oauth/?gitlab=bad_state")
		return
	}
	nextSafe := sanitizeAppNext(fmt.Sprint(claims["next"]))
	rk := sanitizeReturnKey(fmt.Sprint(claims["rk"]))
	feb = strings.TrimSpace(fmt.Sprint(claims["feb"]))
	csrf := infrastructure.RandomTokenURLSafe(24)
	sp := strings.TrimSpace(strings.ToLower(fmt.Sprint(claims["service_provider"])))
	if sp == "" || sp == "<nil>" {
		sp = "default"
	}
	allowedHost := strings.TrimSpace(fmt.Sprint(claims["allowed_host"]))
	if allowedHost == "<nil>" {
		allowedHost = ""
	}
	routeCtx, reason := a.resolveGitLabAuthorizeContext(sp, allowedHost)
	if routeCtx == nil {
		logWarn("GitLab OAuth start route: %s", reason)
		a.redirectGitlabContinue(w, r, feb, rk, "bad_state", "")
		return
	}
	redirectURI := infrastructure.ResolveOAuthRedirectURI(
		routeCtx["redirect_uri"], routeCtx["service_provider"], feb, a.Cfg.PublicEntryOrigins,
	)
	state, err := a.Sess.EncodeOAuthState(infrastructure.OAuthBrowserState{
		CSRF: csrf, UID: uid, Next: nextSafe, RK: rk, Feb: feb,
		RedirectURI: redirectURI, ServiceProvider: routeCtx["service_provider"], AllowedHost: routeCtx["origin"],
	})
	if err != nil {
		logWarn("GitLab OAuth encode state: %v", err)
		a.redirectGitlabContinue(w, r, feb, rk, "bad_state", "")
		return
	}
	sess := a.Sess.Get(r)
	sess[gitlabSessionKey] = map[string]any{
		"csrf": csrf, "uid": uid, "next": nextSafe, "rk": rk, "feb": feb,
		"service_provider": routeCtx["service_provider"],
		"allowed_host":     routeCtx["origin"],
		"redirect_uri":     redirectURI,
	}
	a.Sess.SetForRequest(w, r, sess)
	params := url.Values{}
	params.Set("client_id", routeCtx["client_id"])
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", routeCtx["scope"])
	params.Set("state", state)
	params.Set("response_type", "code")
	redirect(w, r, routeCtx["origin"]+"/oauth/authorize?"+params.Encode())
}

func (a *App) handleGitlabStartFromGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uid := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if uid == "" {
		a.gatewayStartError(w, "GitLab", "missing_x_user_id", "")
		return
	}
	nextSafe := sanitizeAppNext(r.URL.Query().Get("next"))
	rk := sanitizeReturnKey(r.URL.Query().Get("return_key"))
	repoURL := strings.TrimSpace(r.URL.Query().Get("repo_url"))
	feb := a.febFromRequest(r, a.Cfg.FrontendBase)
	csrf := infrastructure.RandomTokenURLSafe(24)
	sp := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("service_provider")))
	if sp == "" {
		sp = "default"
	}
	routeCtx, reason := a.resolveGitLabAuthorizeContext(sp, repoURL)
	if routeCtx == nil {
		a.gatewayStartError(w, "GitLab", "bad_state", reason)
		return
	}
	redirectURI := infrastructure.ResolveOAuthRedirectURI(
		routeCtx["redirect_uri"], routeCtx["service_provider"], feb, a.Cfg.PublicEntryOrigins,
	)
	st := infrastructure.OAuthBrowserState{
		CSRF: csrf, UID: uid, Next: nextSafe, RK: rk, Feb: feb,
		RedirectURI: redirectURI, ServiceProvider: routeCtx["service_provider"], AllowedHost: routeCtx["origin"],
	}
	applyOAuthGrantQuery(&st, r)
	if st.RepoURL == "" {
		st.RepoURL = repoURL
	}
	state, err := a.Sess.EncodeOAuthState(st)
	if err != nil {
		a.gatewayStartError(w, "GitLab", "bad_state", "encode_state_failed")
		return
	}
	sess := a.Sess.Get(r)
	sess[gitlabSessionKey] = map[string]any{
		"csrf": csrf, "uid": uid, "next": nextSafe, "rk": rk, "feb": feb,
		"service_provider": routeCtx["service_provider"],
		"allowed_host":     routeCtx["origin"],
		"redirect_uri":     redirectURI,
	}
	a.Sess.SetForRequest(w, r, sess)
	params := url.Values{}
	params.Set("client_id", routeCtx["client_id"])
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", routeCtx["scope"])
	params.Set("state", state)
	params.Set("response_type", "code")
	writeOAuthStartResponse(w, r, routeCtx["origin"]+"/oauth/authorize?"+params.Encode())
}

func (a *App) handleGitlabCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.handleGitlabCallbackWithSP(w, r, "")
}

func (a *App) handleGitlabCallbackWithSP(w http.ResponseWriter, r *http.Request, serviceProvider string) {
	sessAll := a.Sess.Get(r)
	ctx, _ := sessAll[gitlabSessionKey].(map[string]any)
	if ctx == nil {
		ctx = map[string]any{}
	}
	feb := sessionGetString(ctx, "feb")
	rk := sessionGetString(ctx, "rk")
	nextSafe := sanitizeAppNext(sessionGetString(ctx, "next"))
	urlSP := strings.TrimSpace(strings.ToLower(serviceProvider))
	ctxSP := strings.TrimSpace(strings.ToLower(sessionGetString(ctx, "service_provider")))
	if ctxSP == "" {
		ctxSP = "default"
	}
	clearAnd := func(code string) {
		delete(sessAll, gitlabSessionKey)
		a.Sess.SetForRequest(w, r, sessAll)
		a.redirectGitlabContinue(w, r, feb, rk, code, "")
	}
	// Shared redirect callback SP "tenant-gitlab" defers to session service_provider.
	if urlSP == "tenant-gitlab" {
		urlSP = ""
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		clearAnd("bad_state")
		return
	}
	signed, isSigned, err := a.Sess.DecodeOAuthState(state)
	if err != nil {
		logWarn("GitLab OAuth state: %v", err)
		clearAnd("bad_state")
		return
	}
	if isSigned && signed != nil {
		ctx = map[string]any{
			"csrf": signed.CSRF, "uid": signed.UID, "next": signed.Next, "rk": signed.RK,
			"feb": signed.Feb, "redirect_uri": signed.RedirectURI,
			"service_provider": signed.ServiceProvider, "allowed_host": signed.AllowedHost,
		}
		feb = signed.Feb
		rk = signed.RK
		nextSafe = sanitizeAppNext(signed.Next)
		ctxSP = strings.TrimSpace(strings.ToLower(signed.ServiceProvider))
		if ctxSP == "" {
			ctxSP = "default"
		}
	} else if len(ctx) == 0 || state != sessionGetString(ctx, "csrf") {
		clearAnd("bad_state")
		return
	}
	if urlSP != "" && ctxSP != "" && urlSP != ctxSP {
		clearAnd("bad_state")
		return
	}
	sp := urlSP
	if sp == "" {
		sp = ctxSP
	}
	if sp == "" {
		sp = "default"
	}
	allowedHost := sessionGetString(ctx, "allowed_host")
	providerKey := domain.ProviderKey(gitlabProvider, sp)

	uid := strings.TrimSpace(sessionGetString(ctx, "uid"))
	if uid == "" {
		clearAnd("bad_state")
		return
	}

	pc := a.Cfg.ResolveProviderConfig("gitlab", allowedHost)
	if pc == nil {
		pc, _ = a.resolveProviderByServiceProvider(sp)
	}
	data, err := infrastructure.ExchangeGitLabCode(pc, code, sessionGetString(ctx, "redirect_uri"))
	if err != nil {
		logWarn("GitLab OAuth authorization_code 换票失败: %v", err)
		// 分类提示：GitLab 明确拒绝（4xx / 200+error body，如 code 过期 invalid_grant）→ exchange_rejected；
		// 网络类失败（直连/代理不可达）→ exchange_failed。与 GitHub 侧分类对齐。
		var rej *infrastructure.GitLabExchangeRejectedError
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
	profile, err := infrastructure.FetchGitLabProfile(pc, access)
	if err != nil {
		clearAnd("profile_failed")
		return
	}
	glUID := infrastructure.FormatRemoteUserID(profile["id"])
	if glUID == "" {
		clearAnd("no_gitlab_id")
		return
	}
	glLogin := truncate(fmt.Sprint(profile["username"]), 255)
	if glLogin == "" || glLogin == "<nil>" {
		glLogin = truncate(fmt.Sprint(profile["name"]), 255)
	}
	cipher, err := a.Fernet.Encrypt(refresh)
	if err != nil || cipher == "" {
		clearAnd("exchange_failed")
		return
	}
	_, err = a.DB.UpsertCredential(&infrastructure.CredentialRow{
		Provider:           providerKey,
		Task2appUserID:     uid,
		RefreshTokenCipher: cipher,
		RemoteUserID:       glUID,
		RemoteLogin:        glLogin,
		Scope:              truncate(fmt.Sprint(data["scope"]), 512),
		BindStatus:         domain.BindActive,
		BindError:          "",
	})
	if err != nil {
		clearAnd("exchange_failed")
		return
	}
	a.insertAccessAuditForProvider(providerKey, uid, "oauth_callback_token_issued",
		accessTokenFingerprint(access), map[string]any{
			"gitlab_user_id": glUID,
			"gitlab_login":   truncate(glLogin, 128),
		})

	nextSafe, grantTicket := a.finishResourceGrant(uid, glUID, signed, nextSafe)

	delete(sessAll, gitlabSessionKey)
	a.Sess.SetForRequest(w, r, sessAll)
	rkS := sanitizeReturnKey(rk)
	if rkS != "" && nextSafe != "" {
		redirect(w, r, a.frontendRedirect(feb, appendQueryParam(pathWithParamOK(nextSafe, "gitlab"), "grant_ticket", grantTicket)))
		return
	}
	if rkS != "" {
		a.redirectGitlabContinue(w, r, feb, rkS, "ok", grantTicket)
		return
	}
	if nextSafe != "" {
		redirect(w, r, a.frontendRedirect(feb, appendQueryParam(pathWithParamOK(nextSafe, "gitlab"), "grant_ticket", grantTicket)))
		return
	}
	redirect(w, r, a.frontendRedirect(feb, appendQueryParam("/profile/git-site-oauth/?gitlab=ok", "grant_ticket", grantTicket)))
}

func (a *App) redirectGitlabContinue(w http.ResponseWriter, r *http.Request, feb, returnKey, code, grantTicket string) {
	rk := sanitizeReturnKey(returnKey)
	var path string
	if rk != "" {
		q := url.Values{"returnKey": {rk}, "provider": {"gitlab"}, "gitlab": {code}}
		if strings.TrimSpace(grantTicket) != "" {
			q.Set("grant_ticket", grantTicket)
		}
		path = "/oauth/github-app/callback/?" + q.Encode()
	} else {
		path = "/profile/git-site-oauth/?gitlab=" + url.QueryEscape(code)
		path = appendQueryParam(path, "grant_ticket", grantTicket)
	}
	if code != "ok" {
		a.frontendOAuthErrorRedirect(w, r, feb, path)
		return
	}
	redirect(w, r, a.frontendRedirect(feb, path))
}
