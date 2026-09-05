package domain

import (
	"strings"
	"testing"
)

func TestValidateSaasInboundSkillVersionRejectsEmptyAndUnknown(t *testing.T) {
	cat := []SkillVersionEntry{
		{Version: "1", Status: SkillVersionStatusCurrent},
		{Version: "0", Status: SkillVersionStatusSunset},
	}
	if err := ValidateSaasInboundSkillVersion("", cat); err == nil {
		t.Fatal("empty must fail")
	}
	if err := ValidateSaasInboundSkillVersion("9", cat); err == nil {
		t.Fatal("unknown must fail")
	}
	if err := ValidateSaasInboundSkillVersion("0", cat); err == nil {
		t.Fatal("sunset must fail")
	}
}

func TestValidateSaasInboundSkillVersionAcceptsCurrentAndDeprecated(t *testing.T) {
	cat := []SkillVersionEntry{
		{Version: "1", Status: SkillVersionStatusCurrent},
		{Version: "2", Status: SkillVersionStatusDeprecated},
	}
	if err := ValidateSaasInboundSkillVersion("1", cat); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSaasInboundSkillVersion("v1", cat); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSaasInboundSkillVersion("2", cat); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeSkillVersion(t *testing.T) {
	if got := NormalizeSkillVersion(" v1 "); got != "1" {
		t.Fatalf("got %q", got)
	}
	if strings.TrimSpace(NormalizeSkillVersion("1")) != "1" {
		t.Fatal("plain 1")
	}
}

func TestSupportedSkillVersionSetExcludesSunset(t *testing.T) {
	cat := []SkillVersionEntry{
		{Version: "1", Status: SkillVersionStatusCurrent},
		{Version: "2", Status: SkillVersionStatusDeprecated},
		{Version: "0", Status: SkillVersionStatusSunset},
	}
	set := SupportedSkillVersionSet(cat)
	if !set["1"] || !set["2"] {
		t.Fatalf("current/deprecated must be supported: %v", set)
	}
	if set["0"] {
		t.Fatalf("sunset must be excluded: %v", set)
	}
	if len(set) != 2 {
		t.Fatalf("expected 2 supported versions, got %v", set)
	}
}
