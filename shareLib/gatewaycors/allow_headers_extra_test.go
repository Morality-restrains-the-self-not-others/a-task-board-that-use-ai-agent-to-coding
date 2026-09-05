package gatewaycors

import (
	"strings"
	"testing"
)

func TestAllowHeadersWith_NoExtra(t *testing.T) {
	if got := AllowHeadersWith(); got != AllowHeaders {
		t.Fatalf("AllowHeadersWith() = %q want %q", got, AllowHeaders)
	}
}

func TestAllowHeadersWith_AppendsUniqueExtras(t *testing.T) {
	got := AllowHeadersWith("X-Request-Id", " X-Request-Id ", "X-Relay-To-Trae-Secret")
	for _, want := range []string{
		"Authorization",
		"X-Parent-Span-Id",
		"traceparent",
		"X-Request-Id",
		"X-Relay-To-Trae-Secret",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	parts := strings.Split(got, ", ")
	seen := make(map[string]int, len(parts))
	for _, p := range parts {
		seen[p]++
		if seen[p] > 1 {
			t.Fatalf("duplicate header %q in %q", p, got)
		}
	}
}

func TestAllowHeadersWith_SkipsEmptyAndWhitespace(t *testing.T) {
	got := AllowHeadersWith("  ", "\t", "X-Request-Id")
	if got == AllowHeaders {
		t.Fatal("expected extra header to be appended")
	}
	if !strings.HasSuffix(got, "X-Request-Id") {
		t.Fatalf("suffix = %q", got)
	}
}

func TestAllowHeadersWith_DoesNotDuplicateBaseHeaders(t *testing.T) {
	got := AllowHeadersWith("Authorization", "X-Trace-Id")
	parts := strings.Split(got, ", ")
	seen := make(map[string]int, len(parts))
	for _, p := range parts {
		seen[p]++
		if seen[p] > 1 {
			t.Fatalf("duplicate base header %q in %q", p, got)
		}
	}
	if len(parts) != len(CORSAllowHeaderNames) {
		t.Fatalf("len(parts)=%d want %d (%q)", len(parts), len(CORSAllowHeaderNames), got)
	}
}
