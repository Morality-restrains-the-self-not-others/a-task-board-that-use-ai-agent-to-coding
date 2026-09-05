package main

import "testing"

func TestProviderFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/api/internal/git-oauth/github-token-use-report/", "github"},
		{"/api/internal/git-oauth/gitlab-token-use-report/", "gitlab"},
		{"/api/internal/git-oauth/gitlab-access-for-user/", "gitlab"},
		{"/api/internal/gitlab/oauth/access-for-user/", "gitlab"},
		{"/api/internal/github/oauth/access-for-user/", "github"},
		{"/api/internal/git-oauth/github-refresh/", "github"},
		{"/api/internal/git-oauth/gitlab-refresh/", "gitlab"},
	}
	for _, tc := range cases {
		if got := providerFromPath(tc.path); got != tc.want {
			t.Fatalf("providerFromPath(%q)=%q want %q", tc.path, got, tc.want)
		}
	}
}
