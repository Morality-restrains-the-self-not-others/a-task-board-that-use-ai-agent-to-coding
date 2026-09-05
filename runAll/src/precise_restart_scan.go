package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// fillPreciseRestartFromScanFn populates an empty registry from dirty runAll-hosted
// working trees (same rules as scripts/lib/register-precise-restart-scan.sh).
// Tests replace this to avoid exec.
var fillPreciseRestartFromScanFn = fillPreciseRestartFromScan

func fillPreciseRestartFromScan(cfgPath string) {
	path := preciseRestartFile(cfgPath)
	entries, err := readRegistrationEntries(path)
	if err != nil {
		log.Printf("[precise-restart] scan skipped: read registry %s: %v", path, err)
		return
	}
	if len(entries) > 0 {
		return
	}
	script := preciseRestartScanScript(cfgPath)
	if script == "" {
		log.Printf("[precise-restart] scan skipped: register-precise-restart-scan.sh not found")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", script)
	cmd.Env = append(os.Environ(), "RUNALL_PRECISE_RESTART_FILE="+path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[precise-restart] scan failed: %v", err)
		return
	}
	if len(out) > 0 {
		msg := string(out)
		if len(msg) > 512 {
			msg = msg[:512] + "…"
		}
		log.Printf("[precise-restart] scan: %s", msg)
	}
}

func preciseRestartScanScript(cfgPath string) string {
	var candidates []string
	if root := resolveSourceRoot(cfgPath); root != "" {
		candidates = append(candidates, filepath.Join(root, "scripts/lib/register-precise-restart-scan.sh"))
	}
	if cfgPath != "" {
		if abs, err := filepath.Abs(cfgPath); err == nil {
			root := filepath.Dir(filepath.Dir(abs))
			candidates = append(candidates, filepath.Join(root, "scripts/lib/register-precise-restart-scan.sh"))
		}
	}
	for _, p := range candidates {
		st, err := os.Stat(p)
		if err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
