package infrastructure

import (
	"strings"
	"testing"

	"taskGitOauth/domain"
)

func testGitLabProviders() *Config {
	return &Config{
		Providers: map[string][]ProviderConfig{
			"gitlab": {{
				Provider:        "gitlab",
				ServiceProvider: "daydaymoney-gitlab",
				ProviderKey:     "gitlab:daydaymoney-gitlab",
				Website:         "https://gitlab.daydaymoney.com",
				AuthorizeOrigin: "https://gitlab.daydaymoney.com",
				ClientID:        "cid",
				RedirectURI:     "https://daydaymoney.com/redirect/gitsite/gitlab.daydaymoney.com/oauth/callback/",
				Scope:           domain.DefaultTenantGitLabScope,
			}},
		},
	}
}

func TestResolveGitLabAuthorizeContextDoesNotUseDefaultInstanceForOtherHost(t *testing.T) {
	cfg := testGitLabProviders()
	ctx, reason := ResolveGitLabAuthorizeContext(cfg, "default",
		"https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work")
	if ctx != nil {
		t.Fatalf("must not start OAuth on default GitLab for another host, got %v reason=%s", ctx, reason)
	}
	if reason == "" {
		t.Fatal("expected a failure reason")
	}
}

func TestResolveGitLabAuthorizeContextMatchesRepoHost(t *testing.T) {
	cfg := testGitLabProviders()
	ctx, reason := ResolveGitLabAuthorizeContext(cfg, "default",
		"https://gitlab.daydaymoney.com/group/repo.git")
	if ctx == nil {
		t.Fatalf("expected default GitLab route, reason=%s", reason)
	}
	if ctx["origin"] != "https://gitlab.daydaymoney.com" {
		t.Fatalf("origin=%q", ctx["origin"])
	}
	if ctx["service_provider"] != "daydaymoney-gitlab" {
		t.Fatalf("service_provider=%q", ctx["service_provider"])
	}
}

func TestResolveGitLabAuthorizeContextEmptyScopeIncludesWriteRepository(t *testing.T) {
	cfg := testGitLabProviders()
	cfg.Providers["gitlab"][0].Scope = ""
	ctx, reason := ResolveGitLabAuthorizeContext(cfg, "default",
		"https://gitlab.daydaymoney.com/group/repo.git")
	if ctx == nil {
		t.Fatalf("expected route, reason=%s", reason)
	}
	if !strings.Contains(ctx["scope"], "write_repository") {
		t.Fatalf("empty YAML scope must default to git-over-HTTP write; got %q", ctx["scope"])
	}
}

func TestAllowGitCredentialPrefixFallbackSameHostOnly(t *testing.T) {
	cfg := testGitLabProviders()
	cfg.Providers["gitlab"] = append(cfg.Providers["gitlab"], ProviderConfig{
		Provider: "gitlab", ServiceProvider: "tencent-sh-1",
		ProviderKey: "gitlab:tencent-sh-1",
		Website:     "https://gitlab-tencent-sh-1.daydaymoney.com",
	})
	if AllowGitCredentialPrefixFallback(cfg, "gitlab:tencent-sh-1", "gitlab:daydaymoney-gitlab") {
		t.Fatal("cross-instance GitLab prefix fallback must be denied")
	}
	if !AllowGitCredentialPrefixFallback(cfg, "github:official", "github:legacy") {
		t.Fatal("GitHub prefix fallback stays allowed")
	}
	if AllowGitCredentialPrefixFallback(cfg, "gitlab:missing", "gitlab:daydaymoney-gitlab") {
		t.Fatal("unknown requested GitLab key must not borrow another instance")
	}
}

func TestOriginFromURLPreservesHTTPAndPort(t *testing.T) {
	if got := OriginFromURL("http://115.29.110.74/foo"); got != "http://115.29.110.74" {
		t.Fatalf("got %q", got)
	}
	if got := OriginFromURL("https://gitlab.example:8443/group"); got != "https://gitlab.example:8443" {
		t.Fatalf("got %q", got)
	}
}
