package main

import (
	"testing"
)

func TestNormalizeAuthorizationIDField(t *testing.T) {
	body := map[string]interface{}{"authorization_id": "862031128628060160"}
	if got := normalizeAuthorizationIDField(body, "authorization_id"); got != "862031128628060160" {
		t.Fatalf("string id=%q", got)
	}
	body = map[string]interface{}{"authorization_id": float64(862031128628060160)}
	// Go float64 may round-trip this snowflake; ensure we emit decimal string, not scientific notation.
	got := normalizeAuthorizationIDField(body, "authorization_id")
	if got != "862031128628060160" {
		t.Fatalf("float64 snowflake id=%q", got)
	}
	body = map[string]interface{}{"authorization_id": "8.620311286280602e+17"}
	if got := normalizeAuthorizationIDField(body, "authorization_id"); got != "" {
		t.Fatalf("scientific string should be rejected, got %q", got)
	}
}
