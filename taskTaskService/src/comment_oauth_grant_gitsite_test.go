package main

import (
	"strings"
	"testing"
)

func TestStampImplicitCommentOAuthGitsite_GitLabRamWork(t *testing.T) {
	url := "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git"
	got := stampImplicitCommentOAuthGitsite([]RepoIdentitySelection{{
		RepoURL:       url,
		GitIdentityID: "gid-1",
	}})
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].OauthGitsite != "gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("oauth_gitsite=%q", got[0].OauthGitsite)
	}
	if strings.TrimSpace(got[0].OauthGrantedAt) == "" {
		t.Fatal("oauth_granted_at required")
	}
	kept := stampImplicitCommentOAuthGitsite([]RepoIdentitySelection{{
		RepoURL:      url,
		OauthGitsite: "already.example",
	}})
	if kept[0].OauthGitsite != "already.example" {
		t.Fatalf("must not overwrite existing grant: %q", kept[0].OauthGitsite)
	}
	generic := stampImplicitCommentOAuthGitsite([]RepoIdentitySelection{{
		RepoURL: "https://git.example/acme/demo.git",
	}})
	if generic[0].OauthGitsite != "" {
		t.Fatalf("generic git must stay unstamped: %q", generic[0].OauthGitsite)
	}
}

func TestGitsiteFromRepoURLSSH(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://github.com/acme/demo.git", "github.com"},
		{"http://gitlab.daydaymoney.com:8080/g/r.git", "gitlab.daydaymoney.com:8080"},
		{"git@github.com:acme/demo.git", "github.com"},
		{"ssh://git@github.com/acme/demo.git", "github.com"},
		{"ssh://git@gitlab.daydaymoney.com:2222/g/r.git", "gitlab.daydaymoney.com"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := gitsiteFromRepoURL(tc.in); got != tc.want {
			t.Errorf("gitsiteFromRepoURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
