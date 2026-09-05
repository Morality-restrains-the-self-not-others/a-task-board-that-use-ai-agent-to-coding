package infrastructure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const SessionCookieName = "gitoauth_sessionid"

type SessionStore struct {
	secret []byte
	// domain 为共享父域（如 .daydaymoney.com，不带前导点亦可）；空 = host-only。
	// 非空时 Set/Clear 均带 Domain=，使会话 cookie 在任意子域可用（OAuth 授权
	// 发起于 gitoauth_api.*，回调落地裸域/其他子域时凭据不丢失）。
	domain string
}

func NewSessionStore(secret, cookieDomain string) *SessionStore {
	sum := sha256.Sum256([]byte(secret))
	d := strings.TrimSpace(cookieDomain)
	if d != "" && !strings.HasPrefix(d, ".") {
		d = "." + d
	}
	return &SessionStore{secret: sum[:], domain: d}
}

// CookieDomainFromPublicBase 从服务公开 base URL 推导共享父域（eTLD+1 形态）：
// https://gitoauth_api.daydaymoney.com → .daydaymoney.com；localhost/IP/解析失败 → ""
// （host-only，浏览器不接受为 localhost/IP 设置 Domain cookie）。
func CookieDomainFromPublicBase(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "::1" {
		return ""
	}
	if isIPv4Host(host) {
		return ""
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ""
	}
	return "." + strings.Join(labels[len(labels)-2:], ".")
}

func isIPv4Host(host string) bool {
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func (s *SessionStore) Get(r *http.Request) map[string]any {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c == nil || strings.TrimSpace(c.Value) == "" {
		return map[string]any{}
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(c.Value)
		if err != nil {
			return map[string]any{}
		}
	}
	if len(raw) < 32 {
		return map[string]any{}
	}
	body, sig := raw[:len(raw)-32], raw[len(raw)-32:]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return map[string]any{}
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload
}

func requestLooksHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = strings.TrimSpace(r.Header.Get("X-Forwarded-Protocol"))
	}
	return strings.EqualFold(strings.Split(proto, ",")[0], "https")
}

func (s *SessionStore) Set(w http.ResponseWriter, data map[string]any) {
	s.SetForRequest(w, nil, data)
}

// SetForRequest writes the session cookie; Secure is set when the request is HTTPS
// (including when TLS is terminated at a reverse proxy via X-Forwarded-Proto).
func (s *SessionStore) SetForRequest(w http.ResponseWriter, r *http.Request, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	body, _ := json.Marshal(data)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(body)
	token := append(append([]byte{}, body...), mac.Sum(nil)...)
	val := base64.RawURLEncoding.EncodeToString(token)
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestLooksHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((12 * time.Hour).Seconds()),
	}
	if s.domain != "" {
		cookie.Domain = s.domain
	}
	http.SetCookie(w, cookie)
}

func (s *SessionStore) Clear(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	// 清除必须与 Set 的 Domain 一致，否则域 cookie 无法被删除
	if s.domain != "" {
		cookie.Domain = s.domain
	}
	http.SetCookie(w, cookie)
}

func RandomTokenURLSafe(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
