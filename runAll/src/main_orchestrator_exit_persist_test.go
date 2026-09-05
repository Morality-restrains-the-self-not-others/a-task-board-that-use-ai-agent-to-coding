package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// OPT-20260905-005: exit fingerprint must survive console-log truncate.

func TestPersistOrchestratorExitFingerprint_WritesJSON(t *testing.T) {
	root := t.TempDir()
	if err := persistOrchestratorExitFingerprint(root, "signal", "terminated", "healthy=2 failed=1"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	path := lastOrchestratorExitPath(root)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fingerprint: %v", err)
	}
	var fp orchestratorExitFingerprint
	if err := json.Unmarshal(raw, &fp); err != nil {
		t.Fatalf("json: %v\n%s", err, raw)
	}
	if fp.Source != "signal" || fp.Detail != "terminated" {
		t.Fatalf("source/detail = %q/%q", fp.Source, fp.Detail)
	}
	if fp.Lifecycle != "healthy=2 failed=1" {
		t.Fatalf("lifecycle = %q", fp.Lifecycle)
	}
	if fp.PID != os.Getpid() {
		t.Fatalf("pid = %d, want %d", fp.PID, os.Getpid())
	}
	if fp.ExitedAt == "" {
		t.Fatal("exited_at empty")
	}
	if _, err := time.Parse(time.RFC3339, fp.ExitedAt); err != nil {
		t.Fatalf("exited_at parse: %v", err)
	}
}

func TestPersistOrchestratorExitFingerprint_OverwritesPrevious(t *testing.T) {
	root := t.TempDir()
	if err := persistOrchestratorExitFingerprint(root, "signal", "interrupted", ""); err != nil {
		t.Fatal(err)
	}
	if err := persistOrchestratorExitFingerprint(root, "shutdown-self", "/api/shutdown-self", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(lastOrchestratorExitPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var fp orchestratorExitFingerprint
	if err := json.Unmarshal(raw, &fp); err != nil {
		t.Fatal(err)
	}
	if fp.Source != "shutdown-self" {
		t.Fatalf("want last write to win, got source=%q", fp.Source)
	}
}

func TestPersistOrchestratorExitFingerprint_EmptyRootRejected(t *testing.T) {
	if err := persistOrchestratorExitFingerprint("  ", "signal", "terminated", ""); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestPersistOrchestratorExitFingerprint_UnknownSourceWhenEmpty(t *testing.T) {
	root := t.TempDir()
	if err := persistOrchestratorExitFingerprint(root, "", "", ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(lastOrchestratorExitPath(root))
	var fp orchestratorExitFingerprint
	_ = json.Unmarshal(raw, &fp)
	if fp.Source != "unknown" {
		t.Fatalf("source = %q, want unknown", fp.Source)
	}
}

func TestLastOrchestratorExitPath(t *testing.T) {
	got := lastOrchestratorExitPath("/repo")
	want := filepath.Join("/repo", ".runall", lastOrchestratorExitFile)
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}
