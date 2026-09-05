package aliyun

import (
	"fmt"
	"strings"
	"testing"
)

func TestNetworkIdempotentConstants(t *testing.T) {
	if defaultVPCName == "" || defaultVSwitchName == "" || defaultSGName == "" {
		t.Fatal("default resource names must be set")
	}
	if defaultVPCCIDR == "" || defaultVSwitchCIDR == "" {
		t.Fatal("default CIDR blocks must be set")
	}
	if !strings.Contains(defaultSGName, "入网白名单") {
		t.Fatalf("defaultSGName %q must include whitelist suffix", defaultSGName)
	}
	if legacySGNameFullOpen != "task2app-sg-端口全开有风险" {
		t.Fatalf("legacySGNameFullOpen = %q", legacySGNameFullOpen)
	}
	if legacySGName != "task2app-sg" {
		t.Fatalf("legacySGName = %q, want task2app-sg", legacySGName)
	}
}

func TestWhitelistAutoSGIngressRulesNoFullOpen(t *testing.T) {
	rules := whitelistAutoSGIngressRules("203.0.113.10", "198.51.100.20", "0.0.0.0/0", "bad")
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %+v", rules)
	}
	for _, r := range rules {
		if r.cidr == "0.0.0.0/0" || r.cidr == "::/0" {
			t.Fatalf("full-open must not appear: %+v", rules)
		}
		if r.proto != "all" || r.port != "-1/-1" {
			t.Fatalf("unexpected rule shape: %+v", r)
		}
	}
	got := map[string]bool{}
	for _, r := range rules {
		got[r.cidr] = true
	}
	if !got["203.0.113.10/32"] || !got["198.51.100.20/32"] {
		t.Fatalf("missing expected /32 cidrs: %+v", rules)
	}
}

func TestWhitelistAcceptsExtraRangeCIDR(t *testing.T) {
	rules := whitelistAutoSGIngressRules("203.0.113.10", "100.104.0.0/16")
	got := map[string]bool{}
	for _, r := range rules {
		got[r.cidr] = true
	}
	if !got["203.0.113.10/32"] || !got["100.104.0.0/16"] {
		t.Fatalf("got %+v", rules)
	}
}

func TestParseExtraIngressCIDRs(t *testing.T) {
	got := parseExtraIngressCIDRs("203.0.113.0/24, 0.0.0.0/0;100.104.0.0/16")
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestDefaultAutoSGIngressRulesIncludesEnvExtras(t *testing.T) {
	prev := detectPlatformEgressCIDRFn
	detectPlatformEgressCIDRFn = func() string { return "" }
	t.Cleanup(func() { detectPlatformEgressCIDRFn = prev })
	t.Setenv(extraIngressCIDRsEnv, "198.51.100.0/24")
	rules := defaultAutoSGIngressRules("203.0.113.10")
	got := map[string]bool{}
	for _, r := range rules {
		got[r.cidr] = true
	}
	if !got["203.0.113.10/32"] || !got["198.51.100.0/24"] {
		t.Fatalf("got %+v", rules)
	}
}

func TestDefaultAutoSGIngressRulesClientOnly(t *testing.T) {
	prev := detectPlatformEgressCIDRFn
	detectPlatformEgressCIDRFn = func() string { return "" }
	t.Cleanup(func() { detectPlatformEgressCIDRFn = prev })
	t.Setenv(extraIngressCIDRsEnv, "")
	rules := defaultAutoSGIngressRules("203.0.113.10")
	if len(rules) != 1 || rules[0].cidr != "203.0.113.10/32" {
		t.Fatalf("got %+v", rules)
	}
	empty := defaultAutoSGIngressRules("")
	if len(empty) != 0 {
		t.Fatalf("empty client ip should yield no rules, got %+v", empty)
	}
}

func TestDefaultAutoSGIngressRulesIncludesPlatformEgress(t *testing.T) {
	prev := detectPlatformEgressCIDRFn
	detectPlatformEgressCIDRFn = func() string { return "203.0.113.50/32" }
	t.Cleanup(func() { detectPlatformEgressCIDRFn = prev })
	t.Setenv(extraIngressCIDRsEnv, "")
	rules := defaultAutoSGIngressRules("203.0.113.10")
	got := map[string]bool{}
	for _, r := range rules {
		got[r.cidr] = true
	}
	if !got["203.0.113.10/32"] || !got["203.0.113.50/32"] {
		t.Fatalf("got %+v", rules)
	}
}

func TestNormalizePublicIPCIDR(t *testing.T) {
	if got := normalizePublicIPCIDR("1.2.3.4"); got != "1.2.3.4/32" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePublicIPCIDR("1.2.3.4/32"); got != "1.2.3.4/32" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePublicIPCIDR("1.2.3.0/24"); got != "" {
		t.Fatalf("non-host cidr must be rejected, got %q", got)
	}
	if got := normalizePublicIPCIDR("not-an-ip"); got != "" {
		t.Fatalf("invalid must be empty, got %q", got)
	}
}

func TestIsIPv6CIDR(t *testing.T) {
	if !isIPv6CIDR("240e:37a:24d0:8100:48de:c783:d660:34b1/128") {
		t.Fatal("expected IPv6 /128")
	}
	if !isIPv6CIDR("2001:db8::1") {
		t.Fatal("expected bare IPv6")
	}
	if isIPv6CIDR("203.0.113.10/32") || isIPv6CIDR("203.0.113.10") || isIPv6CIDR("") {
		t.Fatal("IPv4/empty must not be IPv6")
	}
}

func TestIngressSourceFieldsSplitsIPFamily(t *testing.T) {
	v4, v6 := ingressSourceFields("203.0.113.10/32")
	if v4 == nil || *v4 != "203.0.113.10/32" || v6 != nil {
		t.Fatalf("IPv4 fields: v4=%v v6=%v", v4, v6)
	}
	v4, v6 = ingressSourceFields("240e:37a:24d0:8100:48de:c783:d660:34b1/128")
	if v4 != nil || v6 == nil || *v6 != "240e:37a:24d0:8100:48de:c783:d660:34b1/128" {
		t.Fatalf("IPv6 fields: v4=%v v6=%v", v4, v6)
	}
}

func TestIsInvalidSourceCidrError(t *testing.T) {
	if !isInvalidSourceCidrError(fmt.Errorf("SDKError: Code: InvalidParam.SourceCidrIp Message: The specified parameter SourceCidrIp is not valid.")) {
		t.Fatal("expected InvalidParam.SourceCidrIp detection")
	}
	if isInvalidSourceCidrError(fmt.Errorf("other error")) {
		t.Fatal("unexpected match")
	}
}

func TestWhitelistAcceptsIPv6Host(t *testing.T) {
	rules := whitelistAutoSGIngressRules("203.0.113.10", "240e:37a:24d0:8100:48de:c783:d660:34b1")
	got := map[string]bool{}
	for _, r := range rules {
		got[r.cidr] = true
	}
	if !got["203.0.113.10/32"] || !got["240e:37a:24d0:8100:48de:c783:d660:34b1/128"] {
		t.Fatalf("got %+v", rules)
	}
}

func TestIsDuplicatePermissionError(t *testing.T) {
	if !isDuplicatePermissionError(fmt.Errorf("SDKError: Code: InvalidPermission.Duplicate")) {
		t.Fatal("expected duplicate detection")
	}
	if isDuplicatePermissionError(fmt.Errorf("other error")) {
		t.Fatal("unexpected duplicate detection")
	}
}

func TestIsMissingPermissionError(t *testing.T) {
	if !isMissingPermissionError(fmt.Errorf("SDKError: Code: InvalidPermission.NotFound")) {
		t.Fatal("expected missing detection")
	}
	if isMissingPermissionError(fmt.Errorf("other error")) {
		t.Fatal("unexpected missing detection")
	}
}

func TestIsCidrOverlappedError(t *testing.T) {
	if !isCidrOverlappedError(fmt.Errorf("SDKError: Code: InvalidCidrBlock.Overlapped")) {
		t.Fatal("expected overlap detection")
	}
	if isCidrOverlappedError(fmt.Errorf("other error")) {
		t.Fatal("unexpected overlap detection")
	}
}

func TestNextAvailableVSwitchCIDR(t *testing.T) {
	cidr, err := nextAvailableVSwitchCIDR([]string{"192.168.1.0/24"}, defaultVSwitchCIDR)
	if err != nil {
		t.Fatal(err)
	}
	if cidr != "192.168.2.0/24" {
		t.Fatalf("got %s want 192.168.2.0/24", cidr)
	}
}

func TestAutoNetworkInputPreservesExistingIDs(t *testing.T) {
	in := AutoNetworkInput{
		RegionID:          "cn-hangzhou",
		ExistingVPCID:     "vpc-existing",
		ExistingVSwitchID: "vsw-existing",
		ExistingSGID:      "sg-existing",
		ClientPublicIP:    "203.0.113.9",
	}
	out := AutoNetworkResult{
		VPCID: in.ExistingVPCID, VSwitchID: in.ExistingVSwitchID, SecurityGroupID: in.ExistingSGID,
	}
	if out.VPCID != "vpc-existing" || out.VSwitchID != "vsw-existing" || out.SecurityGroupID != "sg-existing" {
		t.Fatalf("existing ids not preserved: %+v", out)
	}
	if in.ClientPublicIP != "203.0.113.9" {
		t.Fatalf("client ip lost")
	}
}
