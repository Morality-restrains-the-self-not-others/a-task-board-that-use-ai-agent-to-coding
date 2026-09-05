package aliyun

import "testing"

func TestPlatformExtraIngressCIDRsIncludesEnv(t *testing.T) {
	prev := detectPlatformEgressCIDRFn
	detectPlatformEgressCIDRFn = func() string { return "203.0.113.9/32" }
	t.Cleanup(func() { detectPlatformEgressCIDRFn = prev })
	t.Setenv(extraIngressCIDRsEnv, "198.51.100.0/24")
	got := map[string]bool{}
	for _, c := range platformExtraIngressCIDRs() {
		got[c] = true
	}
	if !got["198.51.100.0/24"] || !got["203.0.113.9/32"] {
		t.Fatalf("missing cidrs: %v", platformExtraIngressCIDRs())
	}
}
