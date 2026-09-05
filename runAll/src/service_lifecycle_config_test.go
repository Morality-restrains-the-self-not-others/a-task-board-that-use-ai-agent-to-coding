package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_StrictRequiresStopCommand(t *testing.T) {
	t.Setenv("RUNALL_LIFECYCLE_STRICT", "1")
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(path, []byte(`
version: "1"
groups:
  - name: g
    services:
      - name: svc
        start_command: "echo start"
        health_check:
          url: "http://127.0.0.1:1/"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected strict validation error for missing stop_command")
	}
}

func TestLoadConfig_StrictAcceptsLifecycleCommands(t *testing.T) {
	t.Setenv("RUNALL_LIFECYCLE_STRICT", "1")
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(path, []byte(`
version: "1"
groups:
  - name: g
    services:
      - name: svc
        start_command: "echo start"
        stop_command: "echo stop"
        health_check:
          url: "http://127.0.0.1:1/"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
}
