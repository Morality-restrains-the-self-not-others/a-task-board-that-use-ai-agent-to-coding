package application

import "testing"

func TestCloneProviderSitePathAIP(t *testing.T) {
	repo := "http://115.29.110.74/example-user/somanyad.git"
	if got := cloneProviderSite(repo); got != "115.29.110.74" {
		t.Fatalf("cloneProviderSite=%q", got)
	}
}

func TestCloneProviderSiteGithub(t *testing.T) {
	if got := cloneProviderSite("https://github.com/acme/demo.git"); got != "github.com" {
		t.Fatalf("github.com site=%q", got)
	}
	if got := cloneProviderSite(""); got != "" {
		t.Fatalf("empty URL=%q", got)
	}
}

func TestCloneCredentialProvider(t *testing.T) {
	// YAML-resolved key wins (GitHub Enterprise stays github).
	if got := cloneCredentialProvider("gitlab:tencent-sh-1", "https://gitlab.daydaymoney.com/a.git"); got != "gitlab" {
		t.Fatalf("provider=%q", got)
	}
	if got := cloneCredentialProvider("github:github-official", "https://github.com/a.git"); got != "github" {
		t.Fatalf("provider=%q", got)
	}
	if got := cloneCredentialProvider("github:ghe-enterprise", "https://ghe.example.com/a.git"); got != "github" {
		t.Fatalf("GHE provider=%q", got)
	}
	// YAML miss → derive from host.
	if got := cloneCredentialProvider("", "http://115.29.110.74/a.git"); got != "gitlab" {
		t.Fatalf("path-a provider=%q", got)
	}
	if got := cloneCredentialProvider("", "https://github.com/a.git"); got != "github" {
		t.Fatalf("github provider=%q", got)
	}
}

func TestCloneRepoHostSCP(t *testing.T) {
	if got := cloneRepoHost("git@115.29.110.74:example-user/somanyad.git"); got != "115.29.110.74" {
		t.Fatalf("host=%q", got)
	}
}
