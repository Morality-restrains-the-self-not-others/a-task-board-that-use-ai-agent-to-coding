package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// proxyEnvVarNames lists environment variable names that control HTTP proxy behavior.
// These are stripped from child process environments when proxy is disabled.
var proxyEnvVarNames = []string{
	"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
	"http_proxy", "https_proxy", "all_proxy", "no_proxy",
}

func (r *Runner) buildServiceEnv(svc Service) []string {
	env := os.Environ()
	globalUseProxy := false
	if r.cfg != nil {
		globalUseProxy = r.cfg.UseProxy
	}
	if !svc.ShouldUseProxy(globalUseProxy) {
		env = stripProxyEnvVars(env)
	}
	if len(svc.Env) > 0 {
		for k, v := range svc.Env {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// stripProxyEnvVars removes proxy-related environment variables from the given env slice.
func stripProxyEnvVars(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key := ""
		if idx := strings.Index(e, "="); idx >= 0 {
			key = e[:idx]
		}
		isProxy := false
		for _, p := range proxyEnvVarNames {
			if strings.EqualFold(key, p) {
				isProxy = true
				break
			}
		}
		if !isProxy {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (r *Runner) runLifecycleCommand(ctx context.Context, svc *Service, shellCmd string) error {
	shellCmd = strings.TrimSpace(shellCmd)
	if shellCmd == "" {
		return fmt.Errorf("lifecycle command is empty")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cmd := exec.CommandContext(ctx, getBashPath(), "-c", shellCmd)
	if dir := lifecycleWorkDir(svc); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = r.buildServiceEnv(*svc)

	out, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(out))
	if trimmed != "" {
		log.Printf("[%s] lifecycle command output: %s", svc.Name, trimmed)
	}
	if err != nil {
		if trimmed != "" {
			return fmt.Errorf("lifecycle command failed: %w: %s", err, trimmed)
		}
		return fmt.Errorf("lifecycle command failed: %w", err)
	}
	return nil
}

func (r *Runner) runStopCommand(ctx context.Context, svc *Service) error {
	err := r.runLifecycleCommand(ctx, svc, svc.StopCommand)
	if err == nil {
		return nil
	}
	// A stop command that results in the process receiving a signal
	// (e.g., SIGTERM from pkill) is still successful — the goal was
	// to terminate running processes. The shell process itself may
	// be caught in the crossfire when using broad pkill patterns.
	if isSignalExitError(err) {
		log.Printf("[%s] stop command exited with signal (expected for stop): %v", svc.Name, err)
		return nil
	}
	return err
}

func isSignalExitError(err error) bool {
	if err == nil {
		return false
	}
	// When a Go exec.Command process is killed by a signal,
	// the error message contains "signal:" followed by the signal name.
	return strings.Contains(err.Error(), "signal: ")
}

// lifecycleWorkDir is cmd.Dir for stop/start recipes. Missing dirs (P4 has no
// go_relayToTrae/) must not make chdir fail — inherit the orchestrator cwd.
func lifecycleWorkDir(svc *Service) string {
	if svc == nil {
		return ""
	}
	dir := strings.TrimSpace(svc.WorkingDir)
	if dir == "" {
		return ""
	}
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return ""
	}
	return dir
}

// startCommandDir is cmd.Dir for launching a managed service.
// Missing working_dir + ./bin/<elf> (clone-run) inherits the orchestrator cwd so
// Go does not wrap chdir ENOENT as `fork/exec /usr/bin/bash: no such file or directory`.
// Other recipes fail with an explicit working_dir error instead of that fake bash miss.
func startCommandDir(svc *Service) (string, error) {
	if svc == nil {
		return "", nil
	}
	dir := strings.TrimSpace(svc.WorkingDir)
	if dir == "" {
		return "", nil
	}
	fi, err := os.Stat(dir)
	if err == nil && fi.IsDir() {
		return dir, nil
	}
	if isDeployFlatBinStart(svc.EffectiveStartCommand()) {
		log.Printf("[runAll] %s: working_dir %q missing; starting ./bin from orchestrator cwd", svc.Name, dir)
		return "", nil
	}
	return "", fmt.Errorf("working_dir %q does not exist", dir)
}

func (r *Runner) runStopCommandIfConfigured(ctx context.Context, svc *Service) error {
	if svc == nil || strings.TrimSpace(svc.StopCommand) == "" {
		return nil
	}
	return r.runStopCommand(ctx, svc)
}
