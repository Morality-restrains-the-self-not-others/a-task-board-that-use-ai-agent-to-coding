package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"confload"
)

func TestProviderResolverMatchesResolvedGitLabHost(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("monorepo root: %v", err)
	}
	cfg.GitoauthBaseURL = "http://127.0.0.1:8002"
	resolver, err := loadProviderConfigs(repoRoot)
	if err != nil {
		t.Fatalf("loadProviderConfigs: %v", err)
	}

	subs := confload.ResolveBaseYaml(repoRoot)
	gitlabOrigin := confload.ResolveTemplate("${scheme}://${subdomains.gitlab}", subs)
	if strings.Contains(gitlabOrigin, "${") || gitlabOrigin == "" {
		t.Fatalf("expected resolved gitlab origin, got %q", gitlabOrigin)
	}

	match := resolver.matchProvider(gitlabOrigin + "/example-user/task2app.git")
	if !strings.HasPrefix(match.ProviderKey, "gitlab:") {
		t.Fatalf("provider key = %q, want gitlab:* for %s", match.ProviderKey, gitlabOrigin)
	}
	if match.GitoauthBase == "" {
		t.Fatal("expected gitoauth base from provider config")
	}
	if strings.Contains(match.GitoauthBase, "${") {
		t.Fatalf("gitoauth base still has unresolved template: %q", match.GitoauthBase)
	}
}

func TestProviderResolverMatchesTencentSh1GitLabHost(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("monorepo root: %v", err)
	}
	cfg.GitoauthBaseURL = "http://127.0.0.1:8002"
	resolver, err := loadProviderConfigs(repoRoot)
	if err != nil {
		t.Fatalf("loadProviderConfigs: %v", err)
	}

	subs := confload.ResolveBaseYaml(repoRoot)
	origin := confload.ResolveTemplate("${scheme}://${subdomains.gitlabTencentSh1}", subs)
	if strings.Contains(origin, "${") || origin == "" {
		t.Fatalf("expected resolved tencent-sh-1 origin, got %q", origin)
	}

	match := resolver.matchProvider(origin + "/example-user/ram-work")
	if match.ProviderKey != "gitlab:tencent-sh-1" {
		t.Fatalf("provider key = %q, want gitlab:tencent-sh-1 for %s", match.ProviderKey, origin)
	}
	if match.GitoauthBase == "" {
		t.Fatal("expected gitoauth base from provider config")
	}
}

func TestParseProviderYAMLResolvesDomainTemplates(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("monorepo root: %v", err)
	}
	path := filepath.Join(repoRoot, "conf", "auth", "task-credential", "git-oauth-providers", "http-localhost-8012.yaml")
	subs := confload.ResolveBaseYaml(repoRoot)
	record := parseProviderYAML(path, subs)
	if record.Provider != "gitlab" || record.ServiceProvider != "gitlab-local" {
		t.Fatalf("unexpected provider record: %+v", record)
	}
	if record.GitoauthBase != "http://127.0.0.1:8002" {
		t.Fatalf("allowedHost = %q", record.GitoauthBase)
	}
	// OPT-20260818-043 已把 gitlab-local provider 的 website 独立化：
	// localhost 提供方不应再携带主站 gitlab.daydaymoney.com（否则会再次造成
	// 默认实例 token 被借走）。此处断言模板已解析且不与主站实例混用。
	primaryGitLab := confload.ResolveTemplate("${scheme}://${subdomains.gitlab}", subs)
	for _, website := range record.Websites {
		if strings.Contains(website, "${") {
			t.Fatalf("website still unresolved: %q", website)
		}
		if website == primaryGitLab {
			t.Fatalf("localhost provider must not resolve to primary gitlab website %q", website)
		}
	}
	if len(record.Websites) == 0 {
		t.Fatalf("expected at least one resolved localhost website, got none")
	}
}

func TestSanitizeGitOauthResolveError(t *testing.T) {
	got := sanitizeGitOauthResolveError("400 Client Error: Bad Request for url: http://127.0.0.1:8012/oauth/token", 502)
	if got == "" || got == "400 Client Error: Bad Request for url: http://127.0.0.1:8012/oauth/token" {
		t.Fatalf("expected friendly message, got %q", got)
	}
}

func TestIsGitLabRepoDoesNotMisclassifyGitHubDotGit(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("monorepo root: %v", err)
	}
	resolver, err := loadProviderConfigs(repoRoot)
	if err != nil {
		t.Fatalf("loadProviderConfigs: %v", err)
	}
	url := "https://github.com/ruandao/somanyad.git"
	if resolver.IsGitLabRepo(url) {
		t.Fatalf("IsGitLabRepo(%q)=true; GitHub .git URLs must not be classified as GitLab", url)
	}
	if !resolver.IsGitHubRepo(url) {
		t.Fatalf("IsGitHubRepo(%q)=false; want true", url)
	}
}

// TestProviderHeuristicMatrix covers IsGitLabRepo / matchProvider / IsGitHubRepo
// for HTTPS, SCP, self-hosted *.git, and gitlab.com — OPT-20260720-030.
func TestProviderHeuristicMatrix(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("monorepo root: %v", err)
	}
	cfg.GitoauthBaseURL = "http://127.0.0.1:8002"
	resolver, err := loadProviderConfigs(repoRoot)
	if err != nil {
		t.Fatalf("loadProviderConfigs: %v", err)
	}

	type want struct {
		isGitHub bool
		isGitLab bool
		keyPref  string // provider key prefix; empty = do not assert
		notKey   string // exact key that must not be used (cross-instance token)
	}
	cases := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "https github with .git",
			url:  "https://github.com/ruandao/somanyad.git",
			want: want{isGitHub: true, isGitLab: false, keyPref: "github:"},
		},
		{
			name: "https github without .git",
			url:  "https://github.com/ruandao/somanyad",
			want: want{isGitHub: true, isGitLab: false, keyPref: "github:"},
		},
		{
			name: "scp github with .git",
			url:  "git@github.com:ruandao/somanyad.git",
			want: want{isGitHub: true, isGitLab: false, keyPref: "github:"},
		},
		{
			name: "scp github without .git",
			url:  "git@github.com:org/repo",
			want: want{isGitHub: true, isGitLab: false, keyPref: "github:"},
		},
		{
			name: "unknown self-hosted https *.git is GitLab but must not borrow default",
			url:  "https://git.example.com/a/b.git",
			want: want{isGitHub: false, isGitLab: true, notKey: "gitlab:default"},
		},
		{
			name: "unknown self-hosted scp *.git is GitLab but must not borrow default",
			url:  "git@git.example.com:team/app.git",
			want: want{isGitHub: false, isGitLab: true, notKey: "gitlab:default"},
		},
		{
			name: "gitlab.com https",
			url:  "https://gitlab.com/group/project.git",
			want: want{isGitHub: false, isGitLab: true, keyPref: "gitlab:"},
		},
		{
			name: "host containing gitlab substring",
			url:  "https://code.gitlab.example.org/g/p.git",
			want: want{isGitHub: false, isGitLab: true, notKey: "gitlab:default"},
		},
		{
			name: "unknown host without .git is neither",
			url:  "https://git.example.com/a/b",
			want: want{isGitHub: false, isGitLab: false, keyPref: ""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotGH := resolver.IsGitHubRepo(tc.url)
			gotGL := resolver.IsGitLabRepo(tc.url)
			key := resolver.ResolveProvider(tc.url)
			if gotGH != tc.want.isGitHub {
				t.Errorf("IsGitHubRepo(%q)=%v want %v (key=%q)", tc.url, gotGH, tc.want.isGitHub, key)
			}
			if gotGL != tc.want.isGitLab {
				t.Errorf("IsGitLabRepo(%q)=%v want %v (key=%q)", tc.url, gotGL, tc.want.isGitLab, key)
			}
			if tc.want.keyPref != "" && !strings.HasPrefix(key, tc.want.keyPref) {
				t.Errorf("ResolveProvider(%q)=%q want prefix %q", tc.url, key, tc.want.keyPref)
			}
			if tc.want.notKey != "" && key == tc.want.notKey {
				t.Errorf("ResolveProvider(%q)=%q must not borrow %q", tc.url, key, tc.want.notKey)
			}
			// Dual-flag conflict must never happen for GitHub URLs.
			if gotGH && gotGL {
				t.Errorf("both IsGitHub and IsGitLab true for %q (key=%q)", tc.url, key)
			}
		})
	}
}

func TestResolveProviderFallbackGitHubSCPAndUnknownDotGit(t *testing.T) {
	if got := resolveProviderFallback("git@github.com:org/repo.git"); got != "github:github-official" {
		t.Fatalf("fallback github scp = %q", got)
	}
	if got := resolveProviderFallback("https://git.example.com/a/b.git"); got != "" {
		t.Fatalf("fallback unknown .git = %q want empty (must not borrow gitlab:default)", got)
	}
	if got := resolveProviderFallback("https://git.example.com/a/b"); got != "" {
		t.Fatalf("fallback unknown no .git = %q want empty", got)
	}
	if got := resolveProviderFallback("https://gitlab-tencent-sh-1.daydaymoney.com/g/r"); got != "" {
		t.Fatalf("fallback regional gitlab host = %q want empty (must not borrow gitlab:default)", got)
	}
	if got := resolveProviderFallback("https://gitlab.com/g/r.git"); got != "gitlab:default" {
		t.Fatalf("fallback gitlab.com = %q", got)
	}
}

func TestEmptyBranchPayloadNonNil(t *testing.T) {
	payload := emptyBranchPayload()
	if payload.Branches == nil {
		t.Fatal("branches should be non-nil slice")
	}
}

func initProviderTestConfig(t *testing.T) {
	t.Helper()
	if cfg.GitoauthBaseURL == "" {
		cfg.GitoauthBaseURL = "http://127.0.0.1:8002"
	}
	if cfg.GitoauthBridgeSecret == "" {
		cfg.GitoauthBridgeSecret = os.Getenv("GITOAUTH_BRIDGE_JWT_SECRET")
	}
}

// OPT-20260807-071 回归：service 段显式 gitoauth_base 优先于 allowedHost。
// 修复前 parseProviderYAML 误把 allowedHost（公网域名）当 GitoauthBase → access-for-user
// 走公网被 deny-internal 拦截 → token_error。临时 yaml 直接构造两种形态断言。
func TestParseProviderYAMLGitoauthBaseExplicitPreferred(t *testing.T) {
	dir := t.TempDir()
	subs := map[string]string{}

	withExplicit := filepath.Join(dir, "with-explicit.yaml")
	yamlContent := "provider: github\nservice:\n  allowedHost: https://www.daydaymoney.com\n  gitoauth_base: http://127.0.0.1:8002\n  host: 0.0.0.0\n  port: 8002\n"
	if err := os.WriteFile(withExplicit, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := parseProviderYAML(withExplicit, subs)
	if rec.GitoauthBase != "http://127.0.0.1:8002" {
		t.Fatalf("gitoauth_base = %q, want http://127.0.0.1:8002（显式字段优先）", rec.GitoauthBase)
	}

	legacyOnly := filepath.Join(dir, "legacy-only.yaml")
	legacyContent := "provider: github\nservice:\n  allowedHost: https://www.daydaymoney.com\n  host: 0.0.0.0\n  port: 8002\n"
	if err := os.WriteFile(legacyOnly, []byte(legacyContent), 0o644); err != nil {
		t.Fatal(err)
	}
	rec2 := parseProviderYAML(legacyOnly, subs)
	if rec2.GitoauthBase != "https://www.daydaymoney.com" {
		t.Fatalf("legacy fallback gitoauth_base = %q, want allowedHost 兜底 https://www.daydaymoney.com", rec2.GitoauthBase)
	}
}
