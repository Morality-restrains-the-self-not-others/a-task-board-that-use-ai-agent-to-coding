package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"runAll/src/domain"
)

// ScriptObservabilityStorageResetter runs AiMonitor reset_observability_storage.sh.
type ScriptObservabilityStorageResetter struct {
	ScriptPath string
	WorkDir    string
	Runner     func(ctx context.Context, dir, script string) (string, error)
}

func NewScriptObservabilityStorageResetter(scriptPath, workDir string) *ScriptObservabilityStorageResetter {
	return &ScriptObservabilityStorageResetter{
		ScriptPath: scriptPath,
		WorkDir:    workDir,
		Runner:     runResetScript,
	}
}

func (r *ScriptObservabilityStorageResetter) Reset(ctx context.Context) (domain.StorageResetOutcome, error) {
	outcome := domain.StorageResetOutcome{
		LokiReset:       "skipped",
		PromtailReset:   "skipped",
		TempoReset:      "skipped",
		PrometheusReset: "skipped",
	}
	if r == nil || strings.TrimSpace(r.ScriptPath) == "" {
		return outcome, fmt.Errorf("reset script path is required")
	}
	script := strings.TrimSpace(r.ScriptPath)
	if _, err := os.Stat(script); err != nil {
		errMsg := fmt.Sprintf("error: script not found: %v", err)
		outcome.LokiReset = errMsg
		outcome.PromtailReset = errMsg
		outcome.TempoReset = errMsg
		outcome.PrometheusReset = errMsg
		return outcome, err
	}

	runner := r.Runner
	if runner == nil {
		runner = runResetScript
	}
	output, err := runner(ctx, r.WorkDir, script)
	if err != nil {
		msg := strings.TrimSpace(output)
		if msg == "" {
			msg = err.Error()
		}
		outcome.LokiReset = "error: " + msg
		outcome.PromtailReset = outcome.LokiReset
		outcome.TempoReset = outcome.LokiReset
		outcome.PrometheusReset = outcome.LokiReset
		return outcome, err
	}

	outcome.LokiReset = "ok"
	outcome.PromtailReset = "ok"
	outcome.TempoReset = "ok"
	outcome.PrometheusReset = "ok"
	return outcome, nil
}

func runResetScript(ctx context.Context, workDir, script string) (string, error) {
	bashPath := "/usr/bin/bash"
	for _, p := range []string{"/bin/bash", "/usr/bin/bash"} {
		if _, err := os.Stat(p); err == nil {
			bashPath = p
			break
		}
	}
	if path, err := exec.LookPath("bash"); err == nil {
		if _, err := os.Stat(path); err == nil {
			bashPath = path
		}
	}
	cmd := exec.CommandContext(ctx, bashPath, script)
	if strings.TrimSpace(workDir) != "" {
		cmd.Dir = workDir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		combined := strings.TrimSpace(stderr.String())
		if combined == "" {
			combined = err.Error()
		}
		return combined, fmt.Errorf("reset script failed: %w", err)
	}
	return strings.TrimSpace(stderr.String()), nil
}

func ResolveObservabilityResetScript(configPath string) string {
	if env := strings.TrimSpace(os.Getenv("AIMONITOR_RESET_SCRIPT")); env != "" {
		return env
	}

	// Candidates ordered by priority — first match wins.
	// The runAll binary may be launched from the repo root or from its own directory;
	// cover both scenarios plus binary-relative and config-relative fallbacks.
	candidates := []string{
		// Repo-root CWD (runAll started from repo root, e.g. ./runAll/bin/runAll)
		"AiMonitor/scripts/reset_observability_storage.sh",
		// Legacy: CWD is runAll/ directory
		"../AiMonitor/scripts/reset_observability_storage.sh",
		"../../AiMonitor/scripts/reset_observability_storage.sh",
	}

	// Binary-relative fallback: resolve from the runAll executable location.
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe) // e.g. runAll/bin/
		candidates = append(candidates,
			// exeDir/../../AiMonitor/scripts/… → repo root (binary in runAll/bin/)
			filepath.Clean(filepath.Join(exeDir, "..", "..", "AiMonitor", "scripts", "reset_observability_storage.sh")),
			// exeDir/../AiMonitor/scripts/…     → binary directly in runAll/
			filepath.Clean(filepath.Join(exeDir, "..", "AiMonitor", "scripts", "reset_observability_storage.sh")),
		)
	}

	// Config-file-relative candidates.
	if configPath != "" {
		base := filepath.Dir(configPath)
		for _, rel := range []string{
			"../AiMonitor/scripts/reset_observability_storage.sh",
			"../../AiMonitor/scripts/reset_observability_storage.sh",
			"AiMonitor/scripts/reset_observability_storage.sh",
		} {
			candidates = append(candidates, filepath.Clean(filepath.Join(base, rel)))
		}
	}

	// Dedup while preserving priority order.
	seen := map[string]bool{}
	unique := make([]string, 0, len(candidates))
	for _, p := range candidates {
		cleaned := filepath.Clean(p)
		if !seen[cleaned] {
			seen[cleaned] = true
			unique = append(unique, cleaned)
		}
	}

	for _, path := range unique {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback: return the first repo-root-relative path for a clear error message.
	return filepath.Clean("AiMonitor/scripts/reset_observability_storage.sh")
}
