package main

import (
	"strings"
	"testing"
)

func TestDefaultGitlabDiskUnitPriceCentsIsFourYuan(t *testing.T) {
	if DefaultGitlabDiskUnitPriceCents != 400 {
		t.Fatalf("DefaultGitlabDiskUnitPriceCents=%d want 400 (4.00 元/GB/月)", DefaultGitlabDiskUnitPriceCents)
	}
	if got := centsToYuanStr(DefaultGitlabDiskUnitPriceCents); got != "4.00" {
		t.Fatalf("centsToYuanStr(%d)=%q want 4.00", DefaultGitlabDiskUnitPriceCents, got)
	}
}

func TestGitlabDiskMinPurchaseGBIsTen(t *testing.T) {
	if GitlabDiskMinPurchaseGB != 10 {
		t.Fatalf("GitlabDiskMinPurchaseGB=%d want 10", GitlabDiskMinPurchaseGB)
	}
}

func TestValidateGitlabDiskPurchaseQuantity(t *testing.T) {
	if err := validateGitlabDiskPurchaseQuantity(9); err == nil {
		t.Fatal("want error for 9 GB")
	} else if !strings.Contains(err.Error(), "起购") || !strings.Contains(err.Error(), "10") {
		t.Fatalf("err=%v want 起购 10", err)
	}
	if err := validateGitlabDiskPurchaseQuantity(10); err != nil {
		t.Fatalf("10 GB should pass: %v", err)
	}
	if err := validateGitlabDiskPurchaseQuantity(11); err != nil {
		t.Fatalf("11 GB should pass: %v", err)
	}
}

func TestResourcePricingToJSONGitlabDiskMinPurchase(t *testing.T) {
	rp := &ResourcePricing{GitlabDiskUnitPriceCents: DefaultGitlabDiskUnitPriceCents}
	j := resourcePricingToJSON(rp)
	disk, ok := j["gitlab_disk"].(map[string]interface{})
	if !ok {
		t.Fatalf("gitlab_disk type %T", j["gitlab_disk"])
	}
	minQ, ok := disk["min_quantity"].(int64)
	if !ok {
		t.Fatalf("min_quantity type %T want int64", disk["min_quantity"])
	}
	if minQ != GitlabDiskMinPurchaseGB {
		t.Fatalf("min_quantity=%d want %d", minQ, GitlabDiskMinPurchaseGB)
	}
	if _, hasMax := disk["max_quantity"]; hasMax {
		t.Fatal("max_quantity must be removed; 1 GB cap is superseded by min 10 GB")
	}
	desc, _ := disk["description"].(string)
	if !strings.Contains(desc, "起购") || !strings.Contains(desc, "10") {
		t.Fatalf("description=%q want 起购 10 GB", desc)
	}
	if got := disk["price_yuan"]; got != "4.00" {
		t.Fatalf("price_yuan=%v want 4.00", got)
	}
}
