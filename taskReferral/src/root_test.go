package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindMonorepoRootBaseAnchor verifies the root anchor is conf/base.yaml
// (OPT-20260806-057: conf/core/django/config.yaml 退役，旧锚点下根查找失败).
func TestFindMonorepoRootBaseAnchor(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "base.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	if root != dir {
		t.Fatalf("root = %q, want %q", root, dir)
	}
}

// TestFindMonorepoRootLegacyAnchorGone ensures the retired django anchor
// alone does NOT satisfy root detection.
func TestFindMonorepoRootLegacyAnchorGone(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "conf", "core", "django"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "core", "django", "config.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if _, err := findMonorepoRoot(); err == nil {
		t.Fatal("legacy django anchor must not satisfy root detection")
	}
}
