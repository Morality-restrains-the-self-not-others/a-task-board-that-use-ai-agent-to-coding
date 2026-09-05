package infrastructure

import (
	"testing"
)

func TestNormalizeProviderConfigsClientID(t *testing.T) {
	raw := map[string]any{
		"github:github-official-daydaymoney": map[string]any{
			"provider":         "github",
			"service_provider": "github-official-daydaymoney",
			"target": map[string]any{
				"website":       "http://github.com",
				"client_id":     "Iv23liEA2c4007xdQGXJ",
				"client_secret": "test-secret",
				"redirect_uri":  "http://gitoauth.example.com/callback/",
				"scope":         "repo read:user",
			},
			"service": map[string]any{
				"allowedHost": "http://gitoauth.example.com",
				"host":        "0.0.0.0",
				"port":        8002,
			},
		},
	}

	out := normalizeProviderConfigs(raw)
	ghList := out["github"]
	if len(ghList) != 1 {
		t.Fatalf("expected 1 github config, got %d", len(ghList))
	}
	pc := ghList[0]
	if pc.ClientID != "Iv23liEA2c4007xdQGXJ" {
		t.Fatalf("ClientID = %q, want %q", pc.ClientID, "Iv23liEA2c4007xdQGXJ")
	}
	if pc.Website != "http://github.com" {
		t.Fatalf("Website = %q, want %q", pc.Website, "http://github.com")
	}
	t.Logf("OK: provider_key=%s website=%q client_id=%q", pc.ProviderKey, pc.Website, pc.ClientID)
}

func TestResolveProviderByGitsite(t *testing.T) {
	cfg := &Config{Providers: map[string][]ProviderConfig{
		"github": {
			{
				Provider:        "github",
				ServiceProvider: "github-official-daydaymoney",
				ProviderKey:     "github:github-official-daydaymoney",
				Website:         "https://github.com",
			},
		},
		"gitlab": {
			{
				Provider:        "gitlab",
				ServiceProvider: "daydaymoney-gitlab",
				ProviderKey:     "gitlab:daydaymoney-gitlab",
				Website:         "https://gitlab.daydaymoney.com",
				MatchOrigins:    []string{"https://gitlab.daydaymoney.com"},
			},
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Website:         "http://localhost:8012",
			},
			{
				Provider:        "gitlab",
				ServiceProvider: "tencent-sh-1",
				ProviderKey:     "gitlab:tencent-sh-1",
				Website:         "https://gitlab-tencent-sh-1.daydaymoney.com",
			},
		},
	}}

	cases := []struct {
		gitsite string
		wantSP  string
		wantNil bool
	}{
		{"github.com", "github-official-daydaymoney", false},
		{"GitHub.com", "github-official-daydaymoney", false}, // case-insensitive
		{"gitlab.daydaymoney.com", "daydaymoney-gitlab", false},
		{"gitlab-tencent-sh-1.daydaymoney.com", "tencent-sh-1", false},
		{"localhost", "gitlab-local", false},
		{"unknown.example.org", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		pc := cfg.ResolveProviderByGitsite(tc.gitsite)
		if tc.wantNil {
			if pc != nil {
				t.Fatalf("gitsite=%q: expected nil, got %+v", tc.gitsite, pc)
			}
			continue
		}
		if pc == nil {
			t.Fatalf("gitsite=%q: expected provider, got nil", tc.gitsite)
		}
		if pc.ServiceProvider != tc.wantSP {
			t.Fatalf("gitsite=%q: SP=%q, want %q", tc.gitsite, pc.ServiceProvider, tc.wantSP)
		}
	}
}
