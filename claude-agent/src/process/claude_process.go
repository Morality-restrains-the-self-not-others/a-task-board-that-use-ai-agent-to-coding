// Package process manages the Claude Code CLI subprocess.
package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RunOptions configures a Claude Code CLI invocation.
type RunOptions struct {
	Model           string
	MaxSteps        int
	ExtraArgs       []string
	Timeout         time.Duration
	SkipPermissions bool   // --dangerously-skip-permissions (autonomous pipelines)
	APIKey          string // 子进程注入 ANTHROPIC_API_KEY（优先于环境变量）
	BaseURL         string // 子进程注入 ANTHROPIC_BASE_URL（自定义网关/兼容端点，如 DeepSeek Anthropic 兼容接口）
}

// RunResult holds the output of a subprocess execution.
type RunResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

// Manager handles Claude Code CLI subprocess execution.
type Manager struct {
	Binary     string
	WorkingDir string
	ExtraEnv   []string
}

// NewManager creates a Manager, auto-detecting the claude binary.
func NewManager(workingDir string, extraEnv []string) (*Manager, error) {
	binary, err := FindBinary()
	if err != nil {
		return nil, err
	}
	if workingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working dir: %w", err)
		}
		workingDir = wd
	}
	return &Manager{
		Binary:     binary,
		WorkingDir: workingDir,
		ExtraEnv:   extraEnv,
	}, nil
}

// FindBinary locates the claude CLI binary. Search order:
//  1. ./node_modules/.bin/claude (bundled)
//  2. /opt/claude-agent/node_modules/.bin/claude (container)
//  3. PATH: claude
//  4. ~/.npm-global/bin/claude (global npm)
func FindBinary() (string, error) {
	// 1 — bundled in project
	candidates := []string{
		filepath.Join("node_modules", ".bin", "claude"),
		filepath.Join("/opt", "claude-agent", "node_modules", ".bin", "claude"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	// 2 — PATH
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}

	// 3 — npm global
	home, err := os.UserHomeDir()
	if err == nil {
		npmClaude := filepath.Join(home, ".npm-global", "bin", "claude")
		if _, err := os.Stat(npmClaude); err == nil {
			return npmClaude, nil
		}
	}

	// 4 — npx fallback
	if np, err := exec.LookPath("npx"); err == nil {
		return np + " @anthropic-ai/claude-code", nil
	}

	return "", fmt.Errorf("Claude Code CLI not found. Run: npm install -g @anthropic-ai/claude-code")
}

// IsAvailable returns true if the claude CLI can be found.
func IsAvailable() bool {
	_, err := FindBinary()
	return err == nil
}

// RunTask executes `claude -p <task>` synchronously.
func (m *Manager) RunTask(ctx context.Context, task string, opts *RunOptions) (*RunResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}

	// Build argument list: split binary for npx compatibility
	cmdParts := strings.Fields(m.Binary)
	cmdParts = append(cmdParts, "-p", task)

	if opts.Model != "" {
		cmdParts = append(cmdParts, "--model", opts.Model)
	}
	if opts.MaxSteps > 0 {
		cmdParts = append(cmdParts, "--max-turns", strconv.Itoa(opts.MaxSteps))
	}
	if opts.SkipPermissions {
		cmdParts = append(cmdParts, "--dangerously-skip-permissions")
	}
	cmdParts = append(cmdParts, opts.ExtraArgs...)

	var cmd *exec.Cmd
	if ctx != nil {
		cmd = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	} else {
		cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
	}

	cmd.Dir = m.WorkingDir
	cmd.Env = append(os.Environ(), m.ExtraEnv...)
	if opts.APIKey != "" {
		// config 显式 key 优先（覆盖环境变量），使后台可按 config 的 key 计量
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY="+opts.APIKey)
	}
	if opts.BaseURL != "" {
		// 自定义端点必须与 key 一起注入：CLI 默认打官方 api.anthropic.com，
		// 用第三方 key 会 403 "Failed to authenticate"（夜间修复闭环曾因此整体失败）
		cmd.Env = append(cmd.Env, "ANTHROPIC_BASE_URL="+opts.BaseURL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &RunResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		DurationMs: duration,
	}, err
}

// RunInteractive starts claude in interactive mode, passing through stdin/stdout/stderr.
func (m *Manager) RunInteractive(opts *RunOptions) (int, error) {
	if opts == nil {
		opts = &RunOptions{}
	}

	cmdParts := strings.Fields(m.Binary)
	if opts.Model != "" {
		cmdParts = append(cmdParts, "--model", opts.Model)
	}
	cmdParts = append(cmdParts, opts.ExtraArgs...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = m.WorkingDir
	cmd.Env = append(os.Environ(), m.ExtraEnv...)
	if opts.APIKey != "" {
		// config 显式 key 优先（覆盖环境变量），使后台可按 config 的 key 计量
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY="+opts.APIKey)
	}
	if opts.BaseURL != "" {
		// 自定义端点必须与 key 一起注入（同 RunTask，见上）
		cmd.Env = append(cmd.Env, "ANTHROPIC_BASE_URL="+opts.BaseURL)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}
