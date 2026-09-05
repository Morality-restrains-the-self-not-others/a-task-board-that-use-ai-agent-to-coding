package main

import (
	"os"
	"testing"
)

func TestLoadProductionValueStream(t *testing.T) {
	path, err := productionValueStreamConfigPath()
	if err != nil {
		t.Fatalf("productionValueStreamConfigPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("production config not found at %s: %v", path, err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}
	if len(cfg.ValueStreams) == 0 {
		t.Fatal("expected at least one value stream in production config")
	}
}
