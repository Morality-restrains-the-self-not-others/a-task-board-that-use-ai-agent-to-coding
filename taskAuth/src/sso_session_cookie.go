package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ssoCookieDomain mirrors Django frontend_app.views.auth_views._sso_cookie_domain:
// shared parent domain for www/api, or empty for host-only (localhost/IP).
func ssoCookieDomain() string {
	configured := strings.TrimSpace(os.Getenv("SSO_COOKIE_DOMAIN"))
	if configured != "" {
		if !strings.HasPrefix(configured, ".") {
			return "." + configured
		}
		return configured
	}
	base := strings.TrimSpace(cfg.GatewayPublicBase)
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || isIPv4Host(host) {
		return ""
	}
	labels := strings.Split(host, ".")
	if len(labels) >= 2 {
		return "." + strings.Join(labels[len(labels)-2:], ".")
	}
	return ""
}

func isIPv4Host(host string) bool {
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" || !isAllDigits(p) {
			return false
		}
	}
	return true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// clearAuthCookies 清除 userId/token 会话 cookie 的两种变体（每个名字两个 Set-Cookie）：
//   - 父域变体（Domain=.daydaymoney.com，OPT-20260807-071 后主流形态）：只清 host-only
//     时域 cookie 残留 → 在 www 登出后，裸域/其他子域仍携带有效会话（「登出未登出」）。
//   - host-only 变体（071 修复前旧登录遗留的影子 cookie）：只清域变体时影子残留，
//     且 host-only 在 www 上优先于域 cookie 被发送（更具体的 domain 先匹配）→
//     影子 token 失效后 www 永久 401，域 cookie 形同虚设。
// 浏览器按 name+domain+path 匹配删除，HttpOnly/SameSite 属性不影响清除。
// ssoCookieDomain() 返回空（localhost/IP 开发态）时两个 Set-Cookie 属性相同，
// 第二个为幂等替换，无害。
func clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{"userId", "token"} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Domain:   ssoCookieDomain(),
			MaxAge:   -1,
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func ssoCookieSecure(r *http.Request) bool {
	if r != nil && r.TLS != nil {
		return true
	}
	if r != nil {
		proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
		if strings.EqualFold(proto, "https") {
			return true
		}
	}
	return strings.HasPrefix(strings.TrimSpace(cfg.GatewayPublicBase), "https://")
}

// extractSessionKey, setBrowserSessionIDCookie, applyEnrichSessionCookie removed.
// Django session cookies are no longer needed — token-based auth via localStorage
// is the canonical approach. The ssoCookieDomain/ssoCookieSecure helpers remain
// for potential future use by OIDC SLO flows.
