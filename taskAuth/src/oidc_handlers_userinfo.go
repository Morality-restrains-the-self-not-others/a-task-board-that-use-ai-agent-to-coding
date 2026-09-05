package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskAuth/domain"
	"tracelog"
)

// ----- OIDC UserInfo ---------------------------------------------------------

func handleOidcUserInfo(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, oidcErr(r, map[string]string{"error": "invalid_token"}))
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	claims, err := verifyRS256Signature(token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, oidcErr(r, map[string]string{"error": "invalid_token"}))
		return
	}

	// Validate expiry
	exp, _ := claims["exp"].(float64)
	if exp > 0 && time.Now().Unix() > int64(exp) {
		writeJSON(w, http.StatusUnauthorized, oidcErr(r, map[string]string{"error": "invalid_token", "error_description": "token expired"}))
		return
	}

	userID, _ := claims["sub"].(string)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, oidcErr(r, map[string]string{"error": "invalid_token"}))
		return
	}
	if pathTid := oidcPathTenantID(r); pathTid != "" {
		aud := audienceFromClaims(claims)
		if err := domain.CheckOidcPathTenant(pathTid, ownerCompanyIDFromAudience(aud)); err != nil {
			slog.WarnContext(r.Context(), "oidc_path_tenant_mismatch",
				"path_tenant_id", pathTid, "aud", aud,
				"trace_id", tracelog.TraceIDFromContext(r.Context()))
			writeJSON(w, http.StatusUnauthorized, oidcErr(r, map[string]string{"error": "invalid_token"}))
			return
		}
	}

	email, username, _, err := lookupUserByID(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "user lookup failed"}))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sub":                userID,
		"email":              email,
		"email_verified":     true,
		"name":               username,
		"preferred_username": username,
	})
}

// ----- OIDC JWKS -------------------------------------------------------------

func handleOidcJWKS(w http.ResponseWriter, r *http.Request) {
	keys, err := jwksJSON()
	if err != nil {
		slog.ErrorContext(r.Context(), "oidc jwks", "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err)
		writeJSON(w, http.StatusInternalServerError, oidcErr(r, map[string]string{"error": "server_error", "error_description": "jwks generation failed"}))
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

// ----- OIDC EndSession (RP-Initiated Logout) ---------------------------------

// isValidPostLogoutRedirectURI validates the post_logout_redirect_uri against
// the allowlist of trusted origins (GatewayPublicBase + GitServicePublicBase).
// Only http/https schemes are permitted; javascript:/data: URIs are rejected.
// Cross-origin redirects are blocked by host matching.
func isValidPostLogoutRedirectURI(uri string) bool {
	if uri == "" {
		return true // empty is valid (no redirect specified)
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	// Build allowlist from config
	allowed := []string{}
	if cfg.GatewayPublicBase != "" {
		allowed = append(allowed, cfg.GatewayPublicBase)
	}
	if cfg.GitServicePublicBase != "" {
		allowed = append(allowed, cfg.GitServicePublicBase)
	}
	allowed = append(allowed, cfg.PostLogoutOrigins...)
	if len(allowed) == 0 {
		return false // no allowlist configured — reject all external URIs
	}
	for _, a := range allowed {
		au, err := url.Parse(a)
		if err != nil {
			continue
		}
		// Compare hostname (ignore port) — all services (gateway :18081,
		// GitLab :8012, main app :4000) share the same host. Port-scoped
		// matching would reject valid same-host redirects.
		if parsed.Scheme == au.Scheme && parsed.Hostname() == au.Hostname() {
			return true
		}
	}
	return false
}

func handleOidcEndSession(w http.ResponseWriter, r *http.Request) {
	postLogoutRedirectURI := r.URL.Query().Get("post_logout_redirect_uri")

	// Validate redirect URI against allowlist (security: prevent open redirect)
	if !isValidPostLogoutRedirectURI(postLogoutRedirectURI) {
		postLogoutRedirectURI = "" // reject unsafe URI, fall through to default
	}

	// Destroy taskAuth token if present
	key := tokenFromRequest(r)
	if key == "" {
		if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
			key = cookie.Value
		}
	}
	if key != "" {
		_ = deleteTokenByKey(key)
	}

	// Clear userId/token cookies (cross-port SSO bridge) — 双变体（域 + host-only
	// 旧影子）。只发 host-only 清空时，071 后的父域 cookie 会残留 → 在其他子域/
	// 裸域仍然登录（「登出未登出」）；域变体清空保证一次登出全子域生效。
	clearAuthCookies(w)

	// Clear GitLab session cookie directly.
	// GitLab's GET /sign_out returns 500 when a valid session exists,
	// so we cannot rely on the redirect chain. Instead, we clear the
	// _gitlab_session cookie here — cookies are domain-scoped (not
	// port-scoped), so a Set-Cookie from :18081 clears the cookie for
	// :8012 as well.
	// NOTE: This is a client-side-only clearing mechanism. GitLab's
	// server-side session is not invalidated. In production deployments
	// where GitLab is on a different domain, this approach will not work.
	http.SetCookie(w, &http.Cookie{
		Name:     "_gitlab_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect directly to post_logout_redirect_uri (skip GitLab sign_out
	// because we already cleared the session cookie above).
	var redirectURL string
	if postLogoutRedirectURI != "" {
		redirectURL = postLogoutRedirectURI
	} else if cfg.GatewayPublicBase != "" {
		redirectURL = cfg.GatewayPublicBase + "/auth/login/"
	} else {
		redirectURL = "/auth/login/"
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}
