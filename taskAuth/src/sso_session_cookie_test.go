package main

import (
	"os"
	"testing"
)

func TestSsoCookieDomainFromEnv(t *testing.T) {
	t.Setenv("SSO_COOKIE_DOMAIN", "example.com")
	if got := ssoCookieDomain(); got != ".example.com" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("SSO_COOKIE_DOMAIN", ".example.com")
	if got := ssoCookieDomain(); got != ".example.com" {
		t.Fatalf("got %q", got)
	}
	_ = os.Unsetenv("SSO_COOKIE_DOMAIN")
}

func TestSsoCookieDomainFromGateway(t *testing.T) {
	_ = os.Unsetenv("SSO_COOKIE_DOMAIN")
	prev := cfg.GatewayPublicBase
	t.Cleanup(func() { cfg.GatewayPublicBase = prev })
	cfg.GatewayPublicBase = "https://api.daydaymoney.com"
	if got := ssoCookieDomain(); got != ".example.com" {
		t.Fatalf("got %q", got)
	}
	cfg.GatewayPublicBase = "http://127.0.0.1:18081"
	if got := ssoCookieDomain(); got != "" {
		t.Fatalf("expected host-only for IP, got %q", got)
	}
}

// TestApplyEnrichSessionCookieStripsKey removed — applyEnrichSessionCookie is no longer used.
// Django session cookies have been replaced by token-based auth.
