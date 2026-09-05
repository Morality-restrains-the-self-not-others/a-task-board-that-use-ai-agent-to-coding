package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMonorepoMarker(t *testing.T, root string) {
	t.Helper()
	p := filepath.Join(root, "trae-agent", "onlineServiceJS", "run.sh")
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestFindMonorepoRootFrom_FindsRootFromGoRelayBinDir(t *testing.T) {
	root := t.TempDir()
	writeMonorepoMarker(t, root)

	binDir := filepath.Join(root, "go_relayToTrae", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := findMonorepoRootFrom(binDir)
	if err != nil {
		t.Fatalf("findMonorepoRootFrom(bin): %v", err)
	}
	if got != root {
		t.Fatalf("got root %q, want %q", got, root)
	}
}

func TestFindMonorepoRootFrom_FindsRootFromGoRelayWorkingDir(t *testing.T) {
	root := t.TempDir()
	writeMonorepoMarker(t, root)

	wd := filepath.Join(root, "go_relayToTrae")
	if err := os.MkdirAll(wd, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := findMonorepoRootFrom(wd)
	if err != nil {
		t.Fatalf("findMonorepoRootFrom(wd): %v", err)
	}
	if got != root {
		t.Fatalf("got root %q, want %q", got, root)
	}
}

func TestFindMonorepoRootFrom_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := findMonorepoRootFrom(dir)
	if err == nil {
		t.Fatal("want error when marker missing")
	}
}

func TestFindMonorepoRoot_PrefersDeployRootWhenCwdHasNoMarker(t *testing.T) {
	deploy := t.TempDir()
	writeMonorepoMarker(t, deploy)
	cwd := t.TempDir()
	t.Chdir(cwd)
	t.Setenv("DEPLOY_ROOT", deploy)
	t.Setenv("CONF_ROOT", "")

	got, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	if got != deploy {
		t.Fatalf("got root %q, want DEPLOY_ROOT %q", got, deploy)
	}
}
