package main

import "testing"

func TestRepoMatchKeyFromURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://github.com/acme/demo.git", "github.com/acme/demo"},
		{"http://localhost:8012/ljy/somanyad.git", "localhost:8012/ljy/somanyad"},
		{"https://gitlab.daydaymoney.com/example-user/somanyad-emailD.git", "gitlab.daydaymoney.com/example-user/somanyad-emaild"},
		{"git@github.com:Acme/Demo.git", "github.com/acme/demo"},
		{"ssh://git@github.com/acme/demo.git", "github.com/acme/demo"},
		{"ssh://git@gitlab.daydaymoney.com:2222/g/p.git", "gitlab.daydaymoney.com/g/p"},
		{"", ""},
	}
	for _, tc := range cases {
		got := repoMatchKeyFromURL(tc.in)
		if got != tc.want {
			t.Errorf("repoMatchKeyFromURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
