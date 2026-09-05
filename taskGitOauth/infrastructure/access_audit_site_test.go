package infrastructure

import "testing"

func TestAuditSiteHostPort(t *testing.T) {
	cases := []struct {
		website string
		want    string
	}{
		{"https://github.com", "github.com"},
		{"https://github.com/", "github.com"},
		{"https://github.com:443", "github.com:443"},
		{"http://localhost:8012", "localhost:8012"},
		{"http://127.0.0.1:8012/oauth", "127.0.0.1:8012"},
		{"https://gitlab-tencent-sh-1.daydaymoney.com", "gitlab-tencent-sh-1.daydaymoney.com"},
		{"localhost:8012", "localhost:8012"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		got := AuditSiteHostPort(tc.website)
		if got != tc.want {
			t.Fatalf("AuditSiteHostPort(%q)=%q want %q", tc.website, got, tc.want)
		}
	}
}

func TestAuditSiteForProviderKey(t *testing.T) {
	cfg := &Config{
		Providers: map[string][]ProviderConfig{
			"github": {{
				Provider:        "github",
				ServiceProvider: "github-official",
				ProviderKey:     "github:github-official",
				Website:         "https://github.com",
			}},
			"gitlab": {{
				Provider:        "gitlab",
				ServiceProvider: "tencent-sh-1",
				ProviderKey:     "gitlab:tencent-sh-1",
				Website:         "https://gitlab-tencent-sh-1.daydaymoney.com",
			}, {
				Provider:        "gitlab",
				ServiceProvider: "default",
				ProviderKey:     "gitlab:default",
				Website:         "http://localhost:8012",
			}},
		},
	}
	if got := AuditSiteForProviderKey(cfg, "github:github-official"); got != "github.com" {
		t.Fatalf("github official site=%q", got)
	}
	if got := AuditSiteForProviderKey(cfg, "github"); got != "github.com" {
		t.Fatalf("github shorthand site=%q", got)
	}
	if got := AuditSiteForProviderKey(cfg, "gitlab:tencent-sh-1"); got != "gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("gitlab tencent site=%q", got)
	}
	if got := AuditSiteForProviderKey(cfg, "gitlab:default"); got != "localhost:8012" {
		t.Fatalf("gitlab default site=%q", got)
	}
	if got := AuditSiteForProviderKey(cfg, "gitlab:missing"); got != "" {
		t.Fatalf("missing key must not fallback to provider_key, got %q", got)
	}
	if got := AuditSiteForProviderKey(nil, "github"); got != "" {
		t.Fatalf("nil cfg site=%q", got)
	}
}
