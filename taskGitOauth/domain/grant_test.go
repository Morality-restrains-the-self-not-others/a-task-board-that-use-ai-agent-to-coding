package domain

import "testing"

func TestGitsiteHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"https://github.com/acme/demo.git", "github.com"},
		{"https://GitHub.com/acme/demo", "github.com"},
		{"http://gitlab.daydaymoney.com:8080/g/r.git", "gitlab.daydaymoney.com:8080"},
		{"git@github.com:acme/demo.git", "github.com"},
		{"ssh://git@github.com/acme/demo.git", "github.com"},
		{"ssh://git@gitlab.daydaymoney.com:2222/g/r.git", "gitlab.daydaymoney.com"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := GitsiteHost(tc.in); got != tc.want {
			t.Errorf("GitsiteHost(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestApplyResourceGrantGate(t *testing.T) {
	t.Parallel()
	if got := ApplyResourceGrantGate("token_available", false); got != "not_bound" {
		t.Fatalf("L1 without L2: %s", got)
	}
	if got := ApplyResourceGrantGate("token_error", false); got != "not_bound" {
		t.Fatalf("L1 error without L2 should look unbound: %s", got)
	}
	if got := ApplyResourceGrantGate("token_available", true); got != "token_available" {
		t.Fatalf("L2+L1: %s", got)
	}
	if got := ApplyResourceGrantGate("token_error", true); got != "token_error" {
		t.Fatalf("L2 but probe/exchange fail: %s", got)
	}
	if got := ApplyResourceGrantGate("not_bound", false); got != "not_bound" {
		t.Fatalf("already unbound: %s", got)
	}
}

func TestGrantIdempotencyKey(t *testing.T) {
	t.Parallel()
	got := GrantIdempotencyKey("project", "p1", "u1", "GitHub.com")
	want := "grant:project:p1:u1:github.com"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
