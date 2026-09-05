package process

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindBinaryLookPath(t *testing.T) {
	// This test verifies the detection logic is functional, even if claude isn't installed.
	// On systems without claude, it should return an error.
	_, err := FindBinary()
	if err != nil {
		// Expected on systems without claude installed - verify error message
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "Claude Code") {
			t.Errorf("unexpected error message: %v", err)
		}
	}
}

func TestIsAvailable(t *testing.T) {
	available := IsAvailable()
	// If claude is installed, it should return true; otherwise false.
	// Either is valid for unit tests.
	_, err := FindBinary()
	if available && err != nil {
		t.Error("IsAvailable() = true but FindBinary() returned error")
	}
}

func TestRunOptions(t *testing.T) {
	opts := &RunOptions{
		Model:    "claude-sonnet-4-20250514",
		MaxSteps: 50,
	}
	if opts.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q", opts.Model)
	}
	if opts.MaxSteps != 50 {
		t.Errorf("max_steps = %d", opts.MaxSteps)
	}
}

func TestFindBinaryFindsSelf(t *testing.T) {
	// Verify exec.LookPath works for a known binary (go itself)
	p, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in PATH")
	}
	if p == "" {
		t.Error("LookPath returned empty for 'go'")
	}
}

func TestManagerWorkingDirDefaults(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get wd")
	}

	// If claude is not installed, this will fail at FindBinary, which is fine
	m, err := NewManager("", nil)
	if err != nil {
		// Expected if claude not installed
		return
	}
	if m.WorkingDir != wd {
		t.Errorf("working dir = %q, want %q", m.WorkingDir, wd)
	}
}

func TestRunTaskInjectsConfigAPIKey(t *testing.T) {
	// 规则 41 回归：config 的 api_key 必须注入 claude -p 子进程 env（此前仅解析未使用）
	// 父环境即使存在不同的 ANTHROPIC_API_KEY，子进程也须使用 opts.APIKey（Go exec 去重取末值）
	t.Setenv("ANTHROPIC_API_KEY", "sk-parent-env-key")
	stub := filepath.Join(t.TempDir(), "stub-claude.sh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s' \"$ANTHROPIC_API_KEY\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Binary: stub, WorkingDir: t.TempDir()}
	res, err := m.RunTask(context.Background(), "demo task", &RunOptions{APIKey: "sk-config-key-123"})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if got := strings.TrimSpace(res.Stdout); got != "sk-config-key-123" {
		t.Errorf("child env ANTHROPIC_API_KEY = %q, want config key (injected, not parent env)", got)
	}
}

func TestRunTaskNoAPIKeyInheritsEnv(t *testing.T) {
	// 未配置 APIKey 时子进程沿用父环境 key（不注入）
	t.Setenv("ANTHROPIC_API_KEY", "sk-parent-env-key")
	stub := filepath.Join(t.TempDir(), "stub-claude.sh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s' \"$ANTHROPIC_API_KEY\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Binary: stub, WorkingDir: t.TempDir()}
	res, err := m.RunTask(context.Background(), "demo task", &RunOptions{})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if got := strings.TrimSpace(res.Stdout); got != "sk-parent-env-key" {
		t.Errorf("child env = %q, want inherited parent env key", got)
	}
}

func TestRunTaskInjectsBaseURL(t *testing.T) {
	// 规则 41 回归：config 的 base_url 必须注入 claude -p 子进程 env（此前仅解析未注入，
	// 曾导致 DeepSeek key 打到官方 api.anthropic.com → 403 "Failed to authenticate"）
	stub := filepath.Join(t.TempDir(), "stub-claude.sh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s' \"$ANTHROPIC_BASE_URL\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Binary: stub, WorkingDir: t.TempDir()}
	res, err := m.RunTask(context.Background(), "demo task", &RunOptions{
		APIKey:  "sk-config-key-123",
		BaseURL: "https://api.deepseek.com/anthropic",
	})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if got := strings.TrimSpace(res.Stdout); got != "https://api.deepseek.com/anthropic" {
		t.Errorf("child env ANTHROPIC_BASE_URL = %q, want config base_url injected", got)
	}
}
