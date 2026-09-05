package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"tracelog"
)

// ----- OIDC Discovery --------------------------------------------------------

func handleOidcDiscovery(w http.ResponseWriter, r *http.Request) {
	iss := issuerURL()
	doc := map[string]interface{}{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/api/oidc/authorize",
		"token_endpoint":                        iss + "/api/oidc/token",
		"userinfo_endpoint":                     iss + "/api/oidc/userinfo",
		"jwks_uri":                              iss + "/api/oidc/jwks",
		"end_session_endpoint":                  iss + "/api/oidc/endsession",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "email", "email_verified", "name", "preferred_username"},
	}
	writeJSON(w, http.StatusOK, doc)
}

// ----- OIDC Health Check ----------------------------------------------------

// handleOidcHealth provides a self-check for the OIDC provider configuration.
// It validates that the issuer URL is well-formed and reports registered clients,
// signing key status, and template resolution status.
func handleOidcHealth(w http.ResponseWriter, r *http.Request) {
	iss := issuerURL()
	healthy := true
	details := map[string]interface{}{
		"issuer":  iss,
		"service": "taskAuth-oidc",
	}

	// Validate issuer URL
	if iss == "" {
		healthy = false
		details["issuer_valid"] = false
		details["issuer_error"] = "empty issuer URL"
	} else if !strings.HasPrefix(iss, "http://") && !strings.HasPrefix(iss, "https://") {
		healthy = false
		details["issuer_valid"] = false
		details["issuer_error"] = "issuer must start with http:// or https://"
	} else {
		details["issuer_valid"] = true
	}

	// Check if issuer still contains unresolved templates
	if strings.Contains(iss, "${") {
		healthy = false
		details["template_resolved"] = false
		details["template_error"] = "issuer contains unresolved ${...} placeholder"
	} else {
		details["template_resolved"] = true
	}

	// Report bootstrap client count
	details["bootstrap_clients"] = len(cfg.OidcBootstrapClients)

	// Check signing key
	if signingKey != nil {
		details["signing_key"] = map[string]string{
			"kid": signingKey.KeyID,
			"alg": "RS256",
		}
	} else {
		healthy = false
		details["signing_key"] = nil
		details["signing_key_error"] = "signing key not initialized"
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !healthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, map[string]interface{}{
		"status":  status,
		"details": details,
	})
}

// ----- OIDC Authorize --------------------------------------------------------

func handleOidcAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	clientID := strings.TrimSpace(q.Get("client_id"))
	redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
	responseType := strings.TrimSpace(q.Get("response_type"))
	scope := strings.TrimSpace(q.Get("scope"))
	state := strings.TrimSpace(q.Get("state"))
	nonce := strings.TrimSpace(q.Get("nonce"))
	codeChallenge := strings.TrimSpace(q.Get("code_challenge"))
	codeChallengeMethod := strings.TrimSpace(q.Get("code_challenge_method"))

	// Validate params — basic presence checks
	if redirectURI == "" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request", "error_description": "redirect_uri required"}))
		return
	}
	if responseType != "code" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "unsupported_response_type", "error_description": "only code is supported"}))
		return
	}
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request", "error_description": "client_id required"}))
		return
	}
	scopes := parseSpaceSep(scope)
	hasOpenID := false
	for _, s := range scopes {
		if s == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_scope", "error_description": "openid scope required"}))
		return
	}

	// Validate client and redirect_uri BEFORE any redirect.
	// Log structured WARN (client_id + trace_id, never client_secret) so missing
	// OIDC clients are locatable in Loki without first querying the DB (OPT-20260818-039).
	client, err := loadOidcClient(clientID)
	if err != nil || client == nil {
		slog.WarnContext(r.Context(), "oidc authorize client not found",
			"client_id", clientID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "unauthorized_client", "error_description": "client not found"}))
		return
	}
	var allowedURIs []string
	if err := json.Unmarshal([]byte(client.RedirectURIs), &allowedURIs); err != nil {
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "invalid client config"}))
		return
	}
	uriOK := false
	for _, u := range allowedURIs {
		if u == redirectURI {
			uriOK = true
			break
		}
	}
	if !uriOK {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request", "error_description": "redirect_uri not allowed"}))
		return
	}
	if rejectOidcPathTenantMismatch(w, r, ownerCompanyIDOfClient(client), clientID) {
		return
	}

	// Only now — after redirect_uri is validated — define the error redirect helper
	sendError := func(code, desc string) {
		params := url.Values{}
		params.Set("error", code)
		if desc != "" {
			params.Set("error_description", desc)
		}
		if state != "" {
			params.Set("state", state)
		}
		// Include trace_id so GitLab error page can render data-traceId attribute
		if tid := tracelog.TraceIDFromContext(r.Context()); tid != "" {
			params.Set("trace_id", tid)
		}
		http.Redirect(w, r, redirectURI+"?"+params.Encode(), http.StatusFound)
	}

	// Authenticate user
	userID, authErr := resolveTokenUserIDFromRequest(r)
	if authErr != nil {
		// User not logged in — redirect to login with return URL.
		// resumeTarget 用相对路径（r.URL.RequestURI() = /api/oidc[/{tid}]/authorize?...），
		// 登录页 sanitizeOidcResumeNext 已接受该形态并拼回 gateway，不依赖
		// www vs apex vs api 精确 host 匹配。此前拼 oidcLoginRedirectBase()（www）
		// 绝对 URL，在 apex/反向代理 host 变化时仍会丢回跳（OPT-20260808-024）。
		resumeTarget := r.URL.RequestURI()
		loginBase := oidcLoginRedirectBase()
		loginURL := loginBase + "/auth/login/?next=" + url.QueryEscape(resumeTarget)
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	isActive, err := userIsActive(userID)
	if err != nil || !isActive {
		sendError("access_denied", "user inactive or not found")
		return
	}

	// 区域开发模式闸门（OPT-20260823-051）：GitLab 区域实例 OIDC 直连登录时，
	// development 区域仅测试角色账号可登录，其余拒绝签发授权码。
	if oidcRegionGateEnabled(clientID) {
		mode, known, gateErr := gitlabRegionAccessModeForRedirectURI(r.Context(), redirectURI)
		if gateErr != nil {
			// 区域模式查询失败 → fail-open 放行并告警（缺省 release）
			slog.WarnContext(r.Context(), "oidc_region_mode_lookup_failed",
				"client_id", clientID, "redirect_uri", redirectURI,
				"trace_id", tracelog.TraceIDFromContext(r.Context()), "error", gateErr.Error())
		} else if known && mode == oidcRegionAccessDevelopment {
			isTester, testerErr := loadUserIsTester(userID)
			if testerErr != nil || !isTester {
				// 已知开发模式但无法确认测试角色 → 拒绝（fail-closed）
				slog.WarnContext(r.Context(), "oidc_dev_region_blocked_non_tester",
					"client_id", clientID, "user_id", userID,
					"trace_id", tracelog.TraceIDFromContext(r.Context()),
					"tester_check_error", testerErr != nil)
				sendError("access_denied", oidcRegionGateDenyMsg)
				return
			}
		}
	}

	if !enforceTenantGitLabOidcAuthorize(w, r, sendError, client, userID) {
		return
	}

	// PKCE: If code_challenge is provided, validate it
	if codeChallenge != "" {
		if codeChallengeMethod == "" {
			codeChallengeMethod = "S256" // default
		}
		if codeChallengeMethod != "S256" {
			sendError("invalid_request", "only S256 code_challenge_method is supported")
			return
		}
		// RFC 7636 §4.1: code_challenge must be 43-128 chars (base64url-encoded SHA-256 = 43 chars)
		if len(codeChallenge) < 43 || len(codeChallenge) > 128 {
			sendError("invalid_request", "code_challenge must be 43-128 characters")
			return
		}
	}

	// Generate authorization code
	code := mustRandHex(32)
	if err := storeAuthorizationCode(code, clientID, userID, redirectURI, scope, nonce, codeChallenge, codeChallengeMethod); err != nil {
		slog.ErrorContext(r.Context(), "oidc authorize store", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		sendError("server_error", "failed to generate authorization code")
		return
	}

	// Redirect back
	params := url.Values{}
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	http.Redirect(w, r, redirectURI+"?"+params.Encode(), http.StatusFound)
}

// ----- OIDC Token ------------------------------------------------------------

func handleOidcToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, oidcErr(r, map[string]string{"error": "method_not_allowed"}))
		return
	}

	// Support both form-encoded body and JSON
	var clientID, clientSecret, grantType, code, redirectURI, codeVerifier, refreshToken string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		body, err := readJSONBody(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request"}))
			return
		}
		clientID = strField(body, "client_id")
		clientSecret = strField(body, "client_secret")
		grantType = strField(body, "grant_type")
		code = strField(body, "code")
		redirectURI = strField(body, "redirect_uri")
		codeVerifier = strField(body, "code_verifier")
		refreshToken = strField(body, "refresh_token")
	} else {
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request"}))
			return
		}
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
		grantType = r.FormValue("grant_type")
		code = r.FormValue("code")
		redirectURI = r.FormValue("redirect_uri")
		codeVerifier = r.FormValue("code_verifier")
		refreshToken = r.FormValue("refresh_token")
	}

	// Also support HTTP Basic auth for client credentials
	if clientID == "" {
		if id, sec, ok := r.BasicAuth(); ok {
			clientID = id
			clientSecret = sec
		}
	}

	if grantType != "authorization_code" && grantType != "refresh_token" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "unsupported_grant_type"}))
		return
	}
	if clientID == "" || clientSecret == "" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request", "error_description": "missing required parameters"}))
		return
	}

	// Verify client (both grants require client credentials)
	valid, client, err := verifyClientSecret(clientID, clientSecret)
	if err != nil || !valid || client == nil {
		writeJSON(w, http.StatusUnauthorized, oidcErr(r, map[string]string{"error": "invalid_client"}))
		return
	}
	if rejectOidcPathTenantMismatch(w, r, ownerCompanyIDOfClient(client), clientID) {
		return
	}

	// ---- grant_type=refresh_token（offline_access 轮换制）----
	if grantType == "refresh_token" {
		handleOidcRefreshTokenGrant(w, r, clientID, refreshToken)
		return
	}

	if code == "" || redirectURI == "" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request", "error_description": "missing required parameters"}))
		return
	}

	// Consume authorization code
	auth, err := consumeAuthorizationCode(code, clientID, redirectURI)
	if err != nil || auth == nil {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "invalid or expired authorization code"}))
		return
	}

	// PKCE: If code_challenge was stored during authorize, verify code_verifier
	if auth.CodeChallenge.Valid && auth.CodeChallenge.String != "" {
		if codeVerifier == "" {
			writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "code_verifier required (PKCE)"}))
			return
		}
		if !verifyPKCECodeChallenge(codeVerifier, auth.CodeChallenge.String, auth.CodeChallengeMethod.String) {
			writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "invalid code_verifier"}))
			return
		}
	}

	// Look up user info for ID token claims
	email, username, isActive, err := lookupUserByID(auth.UserID)
	if err != nil || !isActive {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "user not found or inactive"}))
		return
	}

	scopes := parseSpaceSep(auth.Scope)
	nonce := ""
	if auth.Nonce.Valid {
		nonce = auth.Nonce.String
	}

	// Issue tokens (only include claims matching requested scopes)
	iss := issuerURL()
	idToken, err := makeIDToken(auth.UserID, email, username, iss, clientID, nonce, scopes)
	if err != nil {
		slog.ErrorContext(r.Context(), "oidc id_token", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "id_token generation failed"}))
		return
	}
	accessToken, err := makeAccessToken(auth.UserID, iss, clientID, scopes)
	if err != nil {
		slog.ErrorContext(r.Context(), "oidc access_token", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "access_token generation failed"}))
		return
	}

	resp := map[string]interface{}{
		"access_token": accessToken,
		"id_token":     idToken,
		"token_type":   "Bearer",
		"expires_in":   cfg.OidcAccessTokenTTL,
	}
	// offline_access scope → 追加签发 refresh token（哈希落库，轮换制）
	if scopeHasOfflineAccess(scopes) {
		refreshTokenPlain := mustRandHex(32)
		if err := storeRefreshToken(clientID, auth.UserID, auth.Scope, sha256Hash(refreshTokenPlain), cfg.OidcRefreshTokenTTL); err != nil {
			slog.ErrorContext(r.Context(), "oidc refresh_token issue", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
			writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "refresh_token issuance failed"}))
			return
		}
		resp["refresh_token"] = refreshTokenPlain
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleOidcRefreshTokenGrant 处理 grant_type=refresh_token（offline_access 轮换制）。
// 原子消费旧 token（撤销）→ 校验用户激活 → 签发新 access_token/id_token + 轮换 refresh_token。
func handleOidcRefreshTokenGrant(w http.ResponseWriter, r *http.Request, clientID, refreshToken string) {
	if refreshToken == "" {
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_request", "error_description": "missing refresh_token"}))
		return
	}
	row, err := consumeRefreshToken(refreshToken, clientID)
	if err != nil {
		if err.Error() == "refresh token expired" {
			slog.WarnContext(r.Context(), "oidc refresh token expired",
				"client_id", clientID,
				"trace_id", tracelog.TraceIDFromContext(r.Context()))
			writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "refresh token expired"}))
			return
		}
		slog.ErrorContext(r.Context(), "oidc refresh token consume", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "refresh token lookup failed"}))
		return
	}
	if row == nil {
		// 已撤销 / 重放 / 不存在 — 统一 invalid_grant，不暴露内部状态
		slog.WarnContext(r.Context(), "oidc refresh token invalid or replayed",
			"client_id", clientID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "invalid or expired refresh token"}))
		return
	}

	scopes := parseSpaceSep(row.Scope)
	email, username, isActive, err := lookupUserByID(row.UserID)
	if err != nil || !isActive {
		slog.WarnContext(r.Context(), "oidc refresh user inactive",
			"client_id", clientID, "user_id", row.UserID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{"error": "invalid_grant", "error_description": "user not found or inactive"}))
		return
	}

	iss := issuerURL()
	accessToken, err := makeAccessToken(row.UserID, iss, clientID, scopes)
	if err != nil {
		slog.ErrorContext(r.Context(), "oidc refresh access_token", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "access_token generation failed"}))
		return
	}

	// 轮换：旧 token 已撤销，签发新 refresh token 替代
	newRefreshToken := mustRandHex(32)
	if err := storeRefreshToken(clientID, row.UserID, row.Scope, sha256Hash(newRefreshToken), cfg.OidcRefreshTokenTTL); err != nil {
		slog.ErrorContext(r.Context(), "oidc refresh token rotate", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "refresh token rotation failed"}))
		return
	}

	resp := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    cfg.OidcAccessTokenTTL,
		"refresh_token": newRefreshToken,
	}
	// 同步签发 id_token（scope 含 openid 时）——供依赖 id_token 的客户端刷新后使用
	if scopeHasOpenID(scopes) {
		idToken, err := makeIDToken(row.UserID, email, username, iss, clientID, "", scopes)
		if err != nil {
			slog.ErrorContext(r.Context(), "oidc refresh id_token", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
			writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "id_token generation failed"}))
			return
		}
		resp["id_token"] = idToken
	}
	writeJSON(w, http.StatusOK, resp)
}
