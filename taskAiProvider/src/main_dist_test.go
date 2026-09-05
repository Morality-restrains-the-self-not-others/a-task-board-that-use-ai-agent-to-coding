package main

import (
	"os"
	"path/filepath"
	"testing"
)

// OPT-20260831-019: 缺 SPA dist 时 ai-provider 必须在进入 listen 前失败（Fatal）。
// ensureFrontendDist 把该守卫抽成可测函数；缺失 index.html 必须返回错误。
func TestEnsureFrontendDistFailsWhenIndexHTMLMissing(t *testing.T) {
	err := ensureFrontendDist(t.TempDir())
	if err == nil {
		t.Fatal("expected error when frontend/dist/index.html is absent")
	}
}

func TestEnsureFrontendDistOKWhenIndexHTMLPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureFrontendDist(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
