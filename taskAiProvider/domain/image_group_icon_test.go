package domain

import "testing"

func TestValidateImageGroupIconMetaOK(t *testing.T) {
	ext, ct, err := ValidateImageGroupIconMeta("logo.PNG", "image/png", 128)
	if err != nil {
		t.Fatal(err)
	}
	if ext != ".png" || ct != "image/png" {
		t.Fatalf("ext=%s ct=%s", ext, ct)
	}
}

func TestValidateImageGroupIconMetaRejectsSVGAndOversize(t *testing.T) {
	if _, _, err := ValidateImageGroupIconMeta("x.svg", "image/svg+xml", 10); err == nil {
		t.Fatal("svg should fail")
	}
	if _, _, err := ValidateImageGroupIconMeta("x.png", "image/png", MaxImageGroupIconBytes+1); err == nil {
		t.Fatal("oversize should fail")
	}
	if _, _, err := ValidateImageGroupIconMeta("x.pdf", "application/pdf", 10); err == nil {
		t.Fatal("pdf should fail")
	}
}

func TestRequireImageGroupFields(t *testing.T) {
	if err := RequireImageGroupFields("n", "d", ""); err == nil || err.Error() != "镜像组图标必填" {
		t.Fatalf("icon: %v", err)
	}
	if err := RequireImageGroupFields("", "d", "k"); err == nil || err.Error() != "镜像组名称必填" {
		t.Fatalf("name: %v", err)
	}
	if err := RequireImageGroupFields("n", "  ", "k"); err == nil || err.Error() != "镜像组描述必填" {
		t.Fatalf("desc: %v", err)
	}
	if err := RequireImageGroupFields("n", "d", "k"); err != nil {
		t.Fatal(err)
	}
}

func TestRenderPathRuleImageGroupIcon(t *testing.T) {
	key, err := RenderPathRule("{userId}/{kind}_{id}{ext}", "", 7, VendorDocKindImageGroupIcon, ".png", 9)
	if err != nil {
		t.Fatal(err)
	}
	if key != "7/image_group_icon_9.png" {
		t.Fatalf("key=%s", key)
	}
}

func TestValidateVendorDocMetaRoutesIconKind(t *testing.T) {
	if _, _, err := ValidateVendorDocMeta(VendorDocKindImageGroupIcon, "a.pdf", "application/pdf", 10); err == nil {
		t.Fatal("icon kind must not accept pdf")
	}
}
