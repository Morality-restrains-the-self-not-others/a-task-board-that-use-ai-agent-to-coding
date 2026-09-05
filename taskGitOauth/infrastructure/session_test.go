package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gitoauth 会话 cookie 的 Domain 行为：配置了共享父域时 Set/Clear 必须带 Domain=，
// 使会话在任意子域可用；未配置（localhost/IP 开发态）时保持 host-only。
func TestSessionStoreSetForRequestEmitsParentDomain(t *testing.T) {
	s := NewSessionStore("test-secret", "example.com")
	req := httptest.NewRequest(http.MethodGet, "https://gitoauth_api.daydaymoney.com/oauth/start", nil)
	rec := httptest.NewRecorder()
	s.SetForRequest(rec, req, map[string]any{"k": "v"})

	sc := rec.Header().Get("Set-Cookie")
	if !strings.Contains(sc, "Domain=example.com") {
		t.Fatalf("Set-Cookie %q missing Domain=example.com（跨子域会话必需）", sc)
	}
	if !strings.Contains(sc, "gitoauth_sessionid=") {
		t.Fatalf("Set-Cookie %q missing session name", sc)
	}
	// HTTPS 请求（含 X-Forwarded-Proto 反代形态）必须带 Secure
	if !strings.Contains(sc, "Secure") {
		t.Fatalf("Set-Cookie %q missing Secure for HTTPS request", sc)
	}
}

func TestSessionStoreClearMatchesDomain(t *testing.T) {
	s := NewSessionStore("test-secret", "example.com")
	rec := httptest.NewRecorder()
	s.Clear(rec)

	sc := rec.Header().Get("Set-Cookie")
	if !strings.Contains(sc, "Domain=example.com") {
		t.Fatalf("Clear Set-Cookie %q missing Domain=example.com（域 cookie 删除必须匹配 Domain）", sc)
	}
	if !strings.Contains(sc, "Max-Age=0") {
		t.Fatalf("Clear Set-Cookie %q missing Max-Age=0", sc)
	}
}

func TestSessionStoreHostOnlyWhenNoDomainConfigured(t *testing.T) {
	s := NewSessionStore("test-secret", "")
	rec := httptest.NewRecorder()
	s.SetForRequest(rec, nil, map[string]any{"k": "v"})

	if sc := rec.Header().Get("Set-Cookie"); strings.Contains(sc, "Domain=") {
		t.Fatalf("Set-Cookie %q must be host-only without configured domain", sc)
	}
}

func TestCookieDomainFromPublicBase(t *testing.T) {
	cases := []struct {
		raw string
		exp string
	}{
		{"https://gitoauth_api.daydaymoney.com", ".daydaymoney.com"},
		{"https://www.daydaymoney.com", ".daydaymoney.com"},
		{"http://example.com", ".example.com"},
		{"http://localhost:8002", ""},
		{"http://127.0.0.1:8002", ""},
		{"http://10.2.150.68:8002", ""},
		{"http://::1:8002", ""},
		{"not-a-url", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := CookieDomainFromPublicBase(c.raw); got != c.exp {
			t.Errorf("CookieDomainFromPublicBase(%q) = %q, want %q", c.raw, got, c.exp)
		}
	}
}
