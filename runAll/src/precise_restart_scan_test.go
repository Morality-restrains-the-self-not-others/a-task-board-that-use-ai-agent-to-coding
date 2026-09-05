package main

import (
	"path/filepath"
	"testing"
)

func TestPreciseRestartScanScript_MissingReturnsEmpty(t *testing.T) {
	t.Setenv("SOURCE_ROOT", t.TempDir())
	t.Setenv("DEPLOY_MODE", "1")
	got := preciseRestartScanScript(filepath.Join(t.TempDir(), "conf", "runAll.yaml"))
	if got != "" {
		t.Fatalf("missing script must return empty, got %q", got)
	}
}

func TestFillPreciseRestartFromScan_SkipsWhenAlreadyRegistered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"task-auth"}); err != nil {
		t.Fatal(err)
	}
	// Must not exec bash when registry already has names (would still be a no-op add).
	fillPreciseRestartFromScan("")
	got, err := readRegisteredServices(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "task-auth" {
		t.Fatalf("got %v", got)
	}
}
