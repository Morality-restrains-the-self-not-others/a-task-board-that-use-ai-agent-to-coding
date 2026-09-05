package main

import "testing"

func TestIsDirectGitHubAccessToken(t *testing.T) {
	cases := map[string]bool{
		"ghu_abc":        true,
		"gho_abc":        true,
		"ghp_abc":        true,
		"github_pat_abc": true,
		"ghr_refresh":    false,
		"":               false,
		"not-a-token":    false,
	}
	for in, want := range cases {
		if got := isDirectGitHubAccessToken(in); got != want {
			t.Errorf("%q got %v want %v", in, got, want)
		}
	}
}
