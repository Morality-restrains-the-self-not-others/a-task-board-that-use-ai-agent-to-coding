package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Keys written by up.sh / prepare-ram-deploy.sh into $DEPLOY_ROOT/cutover.env.
var cutoverEnvKeys = []string{
	"DEPLOY_ROOT",
	"CONF_ROOT",
	"MONOREPO_ROOT",
	"DEPLOY_MODE",
	"SOURCE_ROOT",
	"RUNALL_SKIP_BUILD",
	"RUNALL_BIN",
	"RUNALL_CONFIG",
	"RUNALL_CONSOLE_LOG",
	"RUNALL_LOG_ROOT",
	"RUNALL_OWNERSHIP_STORE",
	"INFRA_HOST",
	"GODEBUG",
}

var cutoverEnvLoaded sync.Map

const cutoverSourceScript = `set -a
source "$1" || exit 1
for k in DEPLOY_ROOT CONF_ROOT MONOREPO_ROOT DEPLOY_MODE SOURCE_ROOT RUNALL_SKIP_BUILD RUNALL_BIN RUNALL_CONFIG RUNALL_CONSOLE_LOG RUNALL_LOG_ROOT RUNALL_OWNERSHIP_STORE INFRA_HOST GODEBUG; do
  if eval "[ -n \"\${$k+x}\" ]"; then
    eval "printf '%s=%s\0' \"$k\" \"\${$k}\""
  fi
done
`

func locateCutoverEnv(configPath string) string {
	if p := strings.TrimSpace(os.Getenv("CUTOVER_ENV")); p != "" {
		return p
	}
	if root := strings.TrimSpace(os.Getenv("DEPLOY_ROOT")); root != "" {
		p := filepath.Join(root, "cutover.env")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if strings.TrimSpace(configPath) == "" {
		return ""
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return ""
	}
	p := filepath.Join(filepath.Dir(filepath.Dir(abs)), "cutover.env")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// resolveSourceRoot returns SOURCE_ROOT from process env, then cutover.env.
// Process env may be stale if cutover.env was rewritten after runAll started
// (ADR-0056 clone-run: agents write $SOURCE_ROOT/.runall/, not the deploy tree).
func resolveSourceRoot(cfgPath string) string {
	if v := strings.TrimSpace(os.Getenv("SOURCE_ROOT")); v != "" {
		return v
	}
	return parseSourceRootFromCutoverFile(locateCutoverEnv(cfgPath))
}

func parseSourceRootFromCutoverFile(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "SOURCE_ROOT" {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		return strings.TrimSpace(val)
	}
	return ""
}

func loadCutoverEnvForConfig(configPath string) error {
	if testing.Testing() && strings.TrimSpace(os.Getenv("CUTOVER_ENV")) == "" {
		return nil
	}
	path := locateCutoverEnv(configPath)
	if path == "" {
		return nil
	}
	return applyCutoverEnvFile(path)
}

func applyCutoverEnvFile(path string) error {
	cmd := exec.Command("bash", "-c", cutoverSourceScript, "cutover-env", path)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}
		return fmt.Errorf("source cutover.env %s: %w: %s", path, err, bytes.TrimSpace(stderr))
	}
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		k, v, ok := bytes.Cut(rec, []byte("="))
		if !ok {
			continue
		}
		key := string(k)
		if !cutoverEnvKeyKnown(key) {
			continue
		}
		if err := os.Setenv(key, string(v)); err != nil {
			return fmt.Errorf("set %s from cutover.env: %w", key, err)
		}
	}
	if _, loaded := cutoverEnvLoaded.LoadOrStore(path, struct{}{}); !loaded {
		log.Printf("[runAll] sourced cutover.env %s", path)
	}
	return nil
}

func cutoverEnvKeyKnown(key string) bool {
	for _, k := range cutoverEnvKeys {
		if k == key {
			return true
		}
	}
	return false
}
