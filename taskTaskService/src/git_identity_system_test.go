package main

import "testing"

func TestBuildSystemGitEmail_FormatAndStable(t *testing.T) {
	a := BuildSystemGitEmail("m1", "t1")
	b := BuildSystemGitEmail("m1", "t1")
	if a != b {
		t.Fatalf("expected stable email, got %q vs %q", a, b)
	}
	if !stringsHasSuffix(a, "@daydaymoney.com") {
		t.Fatalf("suffix: %q", a)
	}
	// 16 + 1 + 16 + len(@daydaymoney.com)
	if len(a) != 16+1+16+len("@daydaymoney.com") {
		t.Fatalf("unexpected length %d for %q", len(a), a)
	}
	other := BuildSystemGitEmail("m2", "t1")
	if other == a {
		t.Fatal("different member_id must change email")
	}
}

func TestResolveSystemGitUserName(t *testing.T) {
	if got := ResolveSystemGitUserName(" Alice ", "u1"); got != "Alice" {
		t.Fatalf("got %q", got)
	}
	if got := ResolveSystemGitUserName("  ", "u1"); got != "u1" {
		t.Fatalf("fallback user_id got %q", got)
	}
	if got := ResolveSystemGitUserName("", ""); got != "user" {
		t.Fatalf("fallback default got %q", got)
	}
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
