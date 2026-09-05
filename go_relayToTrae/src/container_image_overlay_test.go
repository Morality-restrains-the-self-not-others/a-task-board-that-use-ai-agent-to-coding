package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendOnlineServiceSrcOverlayMounts_MountsWholeSrcDir(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "trae-agent", "onlineServiceJS", "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate new module that whitelist historically omitted.
	if err := os.WriteFile(filepath.Join(srcDir, "layerFileContent.mjs"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "server.mjs"), []byte("import './layerFileContent.mjs'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RELAY_OVERLAY_ONLINE_SERVICE_SRC", "")
	got := appendOnlineServiceSrcOverlayMounts(nil, root)
	joined := strings.Join(got, " ")
	want := srcDir + ":/app/onlineServiceJS/src:ro"
	if !strings.Contains(joined, want) {
		t.Fatalf("expected whole-src mount %q in %v", want, got)
	}
	// Must NOT be limited to a single-file whitelist that omits layerFileContent.mjs
	if strings.Contains(joined, "server.mjs:/app/onlineServiceJS/src/server.mjs") {
		t.Fatalf("expected directory overlay, not per-file whitelist; got %v", got)
	}
}

func TestAppendOnlineServiceSrcOverlayMounts_DisabledByEnv(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "trae-agent", "onlineServiceJS", "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RELAY_OVERLAY_ONLINE_SERVICE_SRC", "0")
	got := appendOnlineServiceSrcOverlayMounts([]string{"run"}, root)
	if len(got) != 1 || got[0] != "run" {
		t.Fatalf("expected no overlay when disabled, got %v", got)
	}
}

func TestAppendOnlineServiceSrcOverlayMounts_MissingSrcSkipped(t *testing.T) {
	t.Setenv("RELAY_OVERLAY_ONLINE_SERVICE_SRC", "")
	got := appendOnlineServiceSrcOverlayMounts([]string{"run"}, t.TempDir())
	if len(got) != 1 || got[0] != "run" {
		t.Fatalf("expected no overlay when src missing, got %v", got)
	}
}
