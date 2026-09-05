package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

// ----- helpers ---------------------------------------------------------------

func parseSpaceSep(raw string) []string {
	parts := strings.Fields(raw)
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func resolveTokenUserIDFromRequest(r *http.Request) (string, error) {
	return resolveUserIDFromAuthSources(r, userIDAuthOpts{withIP: true, allowUserIDCookie: true})
}

// resolveTokenUserIDFromRequestStrict is the strict variant WITHOUT userId cookie fallback.
// Used by gateway forward-auth, container gateway, KYC, and other non-OIDC endpoints
// where cross-port browser redirects never occur.
func resolveTokenUserIDFromRequestStrict(r *http.Request) (string, error) {
	return resolveUserIDFromAuthSources(r, userIDAuthOpts{withIP: true, allowUserIDCookie: false})
}

type userIDAuthOpts struct {
	withIP            bool
	allowUserIDCookie bool
}

// resolveUserIDFromAuthSources 身份优先级：显式 Authorization（Token / Bearer JWT）
// 优于网页 Cookie。插件 OAuth JWT 与 www 会话 Cookie 并存时，必须用 JWT，
// 否则会用网页账号去访问插件账号的租户 → 403「您不是该公司的成员」。
func resolveUserIDFromAuthSources(r *http.Request, opts userIDAuthOpts) (string, error) {
	clientIP := ""
	if opts.withIP {
		clientIP = resolveClientIP(r)
	}
	if uid, err := resolveTokenUserIDWithIP(tokenFromRequest(r), clientIP); err == nil && uid != "" {
		return uid, nil
	}
	if uid, err := resolveBearerJWTSubject(r); err == nil && uid != "" {
		return uid, nil
	}
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		if uid, err := resolveTokenUserIDWithIP(cookie.Value, clientIP); err == nil && uid != "" {
			return uid, nil
		}
	}
	if opts.allowUserIDCookie {
		if uid, err := resolveSignedUserIDCookie(r); err == nil && uid != "" {
			return uid, nil
		}
	}
	return "", fmt.Errorf("not authenticated")
}

func resolveBearerJWTSubject(r *http.Request) (string, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", fmt.Errorf("no bearer")
	}
	bearer := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	claims, err := verifyRS256Signature(bearer)
	if err != nil {
		return "", err
	}
	if exp, ok := claims["exp"].(float64); ok && exp > 0 {
		if time.Now().Unix() > int64(exp) {
			return "", fmt.Errorf("bearer token expired")
		}
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("bearer missing sub")
	}
	return sub, nil
}

func resolveSignedUserIDCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("userId")
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", fmt.Errorf("no userId cookie")
	}
	userID, perr := parseUserIDCookieValue(cookie.Value)
	if perr != nil {
		return "", perr
	}
	isActive, err := userIsActive(userID)
	if err != nil || !isActive {
		return "", fmt.Errorf("user inactive")
	}
	hasToken, err := userHasLiveToken(userID)
	if err != nil || !hasToken {
		return "", fmt.Errorf("no live token")
	}
	return userID, nil
}

func issuerURL() string {
	if cfg.OidcIssuer != "" {
		return cfg.OidcIssuer
	}
	if cfg.GatewayPublicBase != "" {
		if u, err := url.Parse(cfg.GatewayPublicBase); err == nil {
			return fmt.Sprintf("http://%s:%d", u.Hostname(), cfg.Port)
		}
		return cfg.GatewayPublicBase
	}
	host := cfg.Host
	if host == "0.0.0.0" || host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, cfg.Port)
}

// oidcLoginRedirectBase prefers the public SPA entry (www.*) from
// postLogoutRedirectOrigins (resolved by confload) so unauthenticated authorize
// redirects land on the same host that holds the user's session cookies.
func oidcLoginRedirectBase() string {
	for _, origin := range cfg.PostLogoutOrigins {
		o := strings.TrimRight(strings.TrimSpace(origin), "/")
		if o == "" || strings.Contains(o, "${") {
			continue
		}
		u, err := url.Parse(o)
		if err != nil || u == nil || u.Hostname() == "" {
			continue
		}
		host := strings.ToLower(u.Hostname())
		if strings.HasPrefix(host, "www.") {
			return o
		}
	}
	if gw := strings.TrimRight(cfg.GatewayPublicBase, "/"); gw != "" {
		return gw
	}
	return strings.TrimRight(issuerURL(), "/")
}

// oidcErr builds a structured error response map with trace_id for every OIDC error.
func oidcErr(r *http.Request, extra map[string]string) map[string]interface{} {
	m := map[string]interface{}{
		"trace_id": tracelog.TraceIDFromContext(r.Context()),
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

// ----- PKCE (RFC 7636) code_verifier validation --------------------------------

// base64urlEncodeNoPad encodes bytes to base64url without padding, per RFC 7636 Appendix A.
func base64urlEncodeNoPad(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// verifyPKCECodeChallenge verifies that the code_verifier matches the stored
// code_challenge using the specified method (currently only S256 supported).
func verifyPKCECodeChallenge(verifier, challenge, method string) bool {
	if method == "" {
		method = "S256"
	}
	if method != "S256" {
		return false
	}

	// RFC 7636 §4.6: code_verifier must be 43-128 chars
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}

	h := sha256.Sum256([]byte(verifier))
	computed := base64urlEncodeNoPad(h[:])
	return computed == challenge
}
