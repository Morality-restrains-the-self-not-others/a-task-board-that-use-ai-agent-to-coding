package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleWorkDir_MissingDirFallsBackToEmpty(t *testing.T) {
	got := lifecycleWorkDir(&Service{
		Name:       "go-relay",
		WorkingDir: filepath.Join(t.TempDir(), "go_relayToTrae"),
	})
	if got != "" {
		t.Fatalf("missing working_dir must not be used as cmd.Dir, got %q", got)
	}
}

func TestLifecycleWorkDir_ExistingDirKept(t *testing.T) {
	dir := t.TempDir()
	got := lifecycleWorkDir(&Service{Name: "docker-mysql", WorkingDir: dir})
	if got != dir {
		t.Fatalf("got %q want %q", got, dir)
	}
}

func TestLifecycleWorkDir_NilService(t *testing.T) {
	if lifecycleWorkDir(nil) != "" {
		t.Fatal("nil service")
	}
}

func TestStartCommandDir_FlatBinMissingDirFallsBackToEmpty(t *testing.T) {
	got, err := startCommandDir(&Service{
		Name:         "task-auth",
		StartCommand: "./bin/taskAuth",
		WorkingDir:   filepath.Join(t.TempDir(), "taskAuth"),
	})
	if err != nil {
		t.Fatalf("flat bin must not fail start when source dir is missing: %v", err)
	}
	if got != "" {
		t.Fatalf("missing working_dir for ./bin start must inherit cwd, got %q", got)
	}
}

func TestStartCommandDir_NonFlatMissingDirErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-recipe")
	_, err := startCommandDir(&Service{
		Name:       "sleep-svc",
		Command:    "sleep 30",
		WorkingDir: missing,
	})
	if err == nil {
		t.Fatal("expected working_dir error for non-flat start")
	}
	if !strings.Contains(err.Error(), "working_dir") {
		t.Fatalf("error=%q want working_dir mentioned", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("error=%q want missing path %q", err, missing)
	}
}

func TestLifecycleWorkDir_Empty(t *testing.T) {
	if err := os.MkdirAll(t.TempDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if lifecycleWorkDir(&Service{WorkingDir: "  "}) != "" {
		t.Fatal("blank working_dir")
	}
}
