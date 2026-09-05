package domain

import "testing"

func TestNormalizeLoginEntry(t *testing.T) {
	if got := NormalizeLoginEntry(" customer "); got != LoginEntryCustomer {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeLoginEntry("unknown"); got != "" {
		t.Fatalf("invalid must be empty, got %q", got)
	}
}

func TestLoginEntryLabel(t *testing.T) {
	if got := LoginEntryLabel(LoginEntryAdmin); got != "管理员入口" {
		t.Fatalf("got %q", got)
	}
	if got := LoginEntryLabel("custom_x"); got != "custom_x" {
		t.Fatalf("unknown passthrough got %q", got)
	}
}

func TestLoginMethodLabel(t *testing.T) {
	if got := LoginMethodLabel("email"); got != "邮箱" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateLoginUserAgent(t *testing.T) {
	if got := TruncateLoginUserAgent("  Mozilla  "); got != "Mozilla" {
		t.Fatalf("got %q", got)
	}
	long := stringsRepeat("a", MaxLoginHistoryUserAgentRunes+10)
	got := TruncateLoginUserAgent(long)
	if len([]rune(got)) != MaxLoginHistoryUserAgentRunes {
		t.Fatalf("len=%d", len([]rune(got)))
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
