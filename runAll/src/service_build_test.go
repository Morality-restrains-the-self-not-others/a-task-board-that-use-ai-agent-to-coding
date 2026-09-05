package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBuildCommand_ExplicitConfigWhenDeployModeUnset(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "")
	got := resolveBuildCommand(Service{
		BuildCommand: "echo built",
		Command:      "echo run",
	})
	if got != "echo built" {
		t.Fatalf("unset DEPLOY_MODE must keep build_command, got %q", got)
	}
}

func TestResolveBuildCommand_ExplicitConfig(t *testing.T) {
	got := resolveBuildCommand(Service{
		BuildCommand: "npm run build",
		Command:      "npm run dev",
	})
	if got != "npm run build" {
		t.Fatalf("resolveBuildCommand explicit = %q, want npm run build", got)
	}
}

func TestResolveBuildCommand_FromBuildScript(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	got := resolveBuildCommand(Service{
		Command:    "./bin/go_relayToTrae",
		WorkingDir: filepath.Join(repoRoot, "go_relayToTrae"),
	})
	if got != "./build.sh" {
		t.Fatalf("resolveBuildCommand go-relay = %q, want ./build.sh", got)
	}
}

func TestResolveBuildCommand_FromGoBuildInCommand(t *testing.T) {
	got := resolveBuildCommand(Service{
		Command:    "go build -o go_relayToTrae . && ./go_relayToTrae",
		WorkingDir: "/tmp/unused",
	})
	if got != "go build -o go_relayToTrae ." {
		t.Fatalf("resolveBuildCommand inline go build = %q", got)
	}
}

func TestResolveBuildCommand_FromRunScript(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	// taskAuth/taskBill now ship build.sh (preferred over run.sh go build inference).
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "task-auth", dir: "taskAuth", want: "./build.sh"},
		{name: "task-bill", dir: "taskBill", want: "./build.sh"},
		{name: "git-oauth", dir: "taskGitOauth", want: "./build.sh"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBuildCommand(Service{
				Command:    "bash run.sh",
				WorkingDir: filepath.Join(repoRoot, tc.dir),
			})
			if got != tc.want {
				t.Fatalf("resolveBuildCommand() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveBuildCommand_NotBuildable(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	// Archived Python gitOauth stub (README only) must not resolve a build command.
	got := resolveBuildCommand(Service{
		Command:    "./run.sh",
		WorkingDir: filepath.Join(repoRoot, "gitOauth"),
	})
	if got != "" {
		t.Fatalf("resolveBuildCommand archived gitOauth = %q, want empty", got)
	}
}

func TestResolveBuildCommand_SkipsShellVariableInRunScript(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	got := resolveBuildCommand(Service{
		Command:    "bash run.sh start accounts",
		WorkingDir: filepath.Join(repoRoot, "taskEvents"),
	})
	if got != "" {
		t.Fatalf("resolveBuildCommand taskEvents without build_command = %q, want empty", got)
	}
}

func TestResolveBuildCommand_DeployModeDisablesInference(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.sh"), []byte("#!/bin/sh\necho no\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := resolveBuildCommand(Service{
		BuildCommand: "go build -o app .",
		WorkingDir:   dir,
	})
	if got != "" {
		t.Fatalf("DEPLOY_MODE must suppress build_command, got %q", got)
	}
}

func TestServiceBuildable(t *testing.T) {
	if !serviceBuildable(&Service{BuildCommand: "npm run build"}) {
		t.Fatal("expected buildable service with explicit build_command")
	}
	if serviceBuildable(&Service{BuildCommand: "   ", Command: "bash run.sh"}) {
		t.Fatal("expected non-buildable service without resolvable build command")
	}
	if serviceBuildable(nil) {
		t.Fatal("expected nil service to be non-buildable")
	}
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	if !serviceBuildable(&Service{
		Command:    "bash run.sh",
		WorkingDir: filepath.Join(repoRoot, "taskAuth"),
	}) {
		t.Fatal("expected task-auth to be buildable via run.sh inference")
	}
}
