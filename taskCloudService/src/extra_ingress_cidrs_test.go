package main

import "testing"

func TestResolveExtraIngressCIDRs(t *testing.T) {
	prev := detectPlatformEgressCIDRFn
	detectPlatformEgressCIDRFn = func() string { return "203.0.113.77/32" }
	t.Cleanup(func() { detectPlatformEgressCIDRFn = prev })
	t.Setenv("TASK2APP_SG_EXTRA_INGRESS_CIDRS", "198.51.100.0/24,0.0.0.0/0")
	got := resolveExtraIngressCIDRs(map[string]interface{}{
		"extra_ingress_cidrs": []interface{}{"100.104.0.0/16", "198.51.100.0/24"},
	})
	seen := map[string]bool{}
	for _, c := range got {
		seen[c] = true
	}
	if !seen["198.51.100.0/24"] || !seen["100.104.0.0/16"] || !seen["203.0.113.77/32"] {
		t.Fatalf("got %v", got)
	}
	// 0.0.0.0/0 must remain rejected even from env.
	if seen["0.0.0.0/0"] {
		t.Fatalf("full-open CIDR must be rejected, got %v", got)
	}
}
