package main

import (
	"testing"

	"confload"
)

func TestResolveProviderKeyFromRepoURL_TencentSh1InterpolatedWebsite(t *testing.T) {
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	origin := confload.ResolveTemplate("${scheme}://${subdomains.gitlabTencentSh1}", confload.ResolveBaseYaml(root))
	if origin == "" || origin == "${scheme}://${subdomains.gitlabTencentSh1}" {
		t.Fatalf("expected interpolated gitlabTencentSh1 origin, got %q", origin)
	}
	repoURL := origin + "/example-user/somanyad.git"
	got := resolveProviderKeyFromRepoURL(repoURL)
	if got != "gitlab:tencent-sh-1" {
		t.Fatalf("provider_key=%q want gitlab:tencent-sh-1 repo=%s (unexpanded ${subdomains.*} collapses to gitlab:default and borrows other CE tokens)", got, repoURL)
	}
}

func TestGitlabProviderKeysToTry_TencentSh1DoesNotBorrowOtherCE(t *testing.T) {
	keys := gitlabProviderKeysToTry("https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git")
	if len(keys) != 1 || keys[0] != "gitlab:tencent-sh-1" {
		t.Fatalf("keys=%v want only gitlab:tencent-sh-1 (daydaymoney-gitlab token must not be tried first)", keys)
	}
}

func TestGitlabProviderKeysToTry_UnmatchedHostStillTriesConfiguredProviders(t *testing.T) {
	keys := gitlabProviderKeysToTry("https://gitlab.daydaymoney.com/ljy/somanyad.git")
	if len(keys) == 0 {
		t.Fatal("unmatched gitlab host should still try configured provider keys")
	}
	found := false
	for _, k := range keys {
		if k == "gitlab:daydaymoney-gitlab" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("keys=%v want gitlab:daydaymoney-gitlab among unmatched-host fallbacks", keys)
	}
	if keys[0] == "gitlab:default" {
		t.Fatalf("keys=%v must not put invented gitlab:default first", keys)
	}
}
