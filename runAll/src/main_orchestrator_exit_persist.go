package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// lastOrchestratorExitFile is the durable exit fingerprint under .runall/
// (gitignored runtime state). Survives logs/runall-console.log truncate
// (OPT-20260905-005).
const lastOrchestratorExitFile = "last_orchestrator_exit.json"

// orchestratorExitFingerprint is the JSON written on graceful/fatal orchestrator exit.
type orchestratorExitFingerprint struct {
	ExitedAt  string `json:"exited_at"`
	PID       int    `json:"pid"`
	PPID      int    `json:"ppid"`
	PGID      int    `json:"pgid,omitempty"`
	Source    string `json:"source"`
	Detail    string `json:"detail"`
	Lifecycle string `json:"lifecycle,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
}

func lastOrchestratorExitPath(monorepoRoot string) string {
	return filepath.Join(monorepoRoot, ".runall", lastOrchestratorExitFile)
}

func buildOrchestratorExitFingerprint(source, detail, lifecycle string) orchestratorExitFingerprint {
	pid := os.Getpid()
	pgid, _ := syscall.Getpgid(pid)
	host, _ := os.Hostname()
	if source == "" {
		source = "unknown"
	}
	return orchestratorExitFingerprint{
		ExitedAt:  time.Now().Format(time.RFC3339),
		PID:       pid,
		PPID:      os.Getppid(),
		PGID:      pgid,
		Source:    source,
		Detail:    detail,
		Lifecycle: lifecycle,
		Hostname:  host,
	}
}

// writeOrchestratorExitFingerprint atomically replaces .runall/last_orchestrator_exit.json.
func writeOrchestratorExitFingerprint(monorepoRoot string, fp orchestratorExitFingerprint) error {
	if strings.TrimSpace(monorepoRoot) == "" {
		return fmt.Errorf("monorepo root is empty")
	}
	dir := filepath.Join(monorepoRoot, ".runall")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir .runall: %w", err)
	}
	path := lastOrchestratorExitPath(monorepoRoot)
	tmp := path + ".tmp"
	raw, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write temp fingerprint: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename fingerprint: %w", err)
	}
	return nil
}

// persistOrchestratorExitFingerprint best-effort records why the orchestrator exited.
func persistOrchestratorExitFingerprint(monorepoRoot, source, detail, lifecycle string) error {
	return writeOrchestratorExitFingerprint(monorepoRoot, buildOrchestratorExitFingerprint(source, detail, lifecycle))
}

// persistOrchestratorExitBestEffort resolves monorepo root and writes the fingerprint;
// failures are logged and never block process exit.
func persistOrchestratorExitBestEffort(runner *Runner, source, detail, lifecycle string) {
	if runner == nil {
		return
	}
	root, err := runner.monorepoRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		log.Printf("[runAll] skip exit fingerprint: monorepo root unavailable: %v", err)
		return
	}
	if err := persistOrchestratorExitFingerprint(root, source, detail, lifecycle); err != nil {
		log.Printf("[runAll] persist exit fingerprint: %v", err)
		return
	}
	log.Printf("[runAll] exit fingerprint written path=%s source=%s detail=%q",
		lastOrchestratorExitPath(root), source, detail)
}
