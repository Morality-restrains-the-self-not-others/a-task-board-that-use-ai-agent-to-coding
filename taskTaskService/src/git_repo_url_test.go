package main

import "testing"

func TestCanonicalGitRepoURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  https://github.com/Ruandao/Helloworld.git/  ", "https://github.com/ruandao/helloworld"},
		{"https://github.com/ruandao/helloworld", "https://github.com/ruandao/helloworld"},
		{"https://github.com/test-ruandao/helloworld.git", "https://github.com/test-ruandao/helloworld"},
		{"git@github.com:acme/demo.git", "git@github.com:acme/demo"},
	}
	for _, tc := range cases {
		if got := canonicalGitRepoURL(tc.in); got != tc.want {
			t.Fatalf("canonicalGitRepoURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestGitRepoURLsEqual(t *testing.T) {
	t.Parallel()
	if !gitRepoURLsEqual("https://github.com/acme/demo", "https://github.com/acme/demo.git") {
		t.Fatal("same repo with/without .git must be equal")
	}
	if gitRepoURLsEqual("https://github.com/ruandao/helloworld", "https://github.com/test-ruandao/helloworld.git") {
		t.Fatal("different GitHub owners must not be equal")
	}
	if gitRepoURLsEqual("", "https://github.com/acme/demo.git") {
		t.Fatal("empty stored address must not equal a current URL")
	}
}

func TestIndexOfGitRepoURL(t *testing.T) {
	t.Parallel()
	urls := []string{
		"https://github.com/acme/alpha.git",
		"https://github.com/acme/beta",
	}
	if got := indexOfGitRepoURL(urls, "https://github.com/acme/beta.git"); got != 1 {
		t.Fatalf("indexOfGitRepoURL beta.git = %d want 1", got)
	}
	if got := indexOfGitRepoURL(urls, "https://github.com/acme/missing"); got != -1 {
		t.Fatalf("indexOfGitRepoURL missing = %d want -1", got)
	}
}
