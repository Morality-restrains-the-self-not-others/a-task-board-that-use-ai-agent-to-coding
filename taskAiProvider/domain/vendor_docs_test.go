package domain

import "testing"

func TestRenderPathRuleOK(t *testing.T) {
	key, err := RenderPathRule("{keyPrefix}/{userId}/{kind}_{id}{ext}", "vendor-docs", 42, VendorDocKindIDCard, ".png", 99)
	if err != nil {
		t.Fatal(err)
	}
	if key != "vendor-docs/42/id_card_99.png" {
		t.Fatalf("key=%s", key)
	}
}

func TestRenderPathRuleRejectsTraversalAndUnknown(t *testing.T) {
	if _, err := RenderPathRule("{keyPrefix}/../etc/{userId}", "vendor-docs", 1, VendorDocKindIDCard, ".jpg", 1); err == nil {
		t.Fatal("expected .. reject")
	}
	if _, err := RenderPathRule("{keyPrefix}/{evil}", "vendor-docs", 1, VendorDocKindIDCard, ".jpg", 1); err == nil {
		t.Fatal("expected unknown placeholder")
	}
	if err := ValidatePathRule(""); err == nil {
		t.Fatal("expected empty reject")
	}
}

func TestOwnsFileKey(t *testing.T) {
	if !OwnsFileKey(42, "vendor-docs/42/id_card_1.png") {
		t.Fatal("should own")
	}
	if OwnsFileKey(42, "vendor-docs/99/id_card_1.png") {
		t.Fatal("should not own")
	}
	if OwnsFileKey(42, "../42/x.png") {
		t.Fatal("traversal")
	}
}
