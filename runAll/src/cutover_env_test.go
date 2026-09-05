package main

import (
	"os"
	"path/filepath"
	"testing"
)

func restoreCutoverKeys(t *testing.T) {
	t.Helper()
	saved := make([]struct {
		key     string
		val     string
		present bool
	}, len(cutoverEnvKeys))
	for i, k := range cutoverEnvKeys {
		v, ok := os.LookupEnv(k)
		saved[i].key = k
		saved[i].val = v
		saved[i].present = ok
	}
	t.Cleanup(func() {
		for _, s := range saved {
			if s.present {
				_ = os.Setenv(s.key, s.val)
			} else {
				_ = os.Unsetenv(s.key)
			}
		}
	})
}

func TestLocateCutoverEnv_ConfigParent(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cutover := filepath.Join(root, "cutover.env")
	if err := os.WriteFile(cutover, []byte("export DEPLOY_MODE=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CUTOVER_ENV", "")
	t.Setenv("DEPLOY_ROOT", "")
	got := locateCutoverEnv(filepath.Join(confDir, "runAll.yaml"))
	if got != cutover {
		t.Fatalf("locateCutoverEnv=%q want %q", got, cutover)
	}
}

func TestLocateCutoverEnv_CUTOVER_ENVWins(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.env")
	if err := os.WriteFile(explicit, []byte("export DEPLOY_MODE=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CUTOVER_ENV", explicit)
	got := locateCutoverEnv("/no/such/conf/runAll.yaml")
	if got != explicit {
		t.Fatalf("locateCutoverEnv=%q want CUTOVER_ENV %q", got, explicit)
	}
}

func TestLocateCutoverEnv_EmptyConfigPathSkipsCwdGuess(t *testing.T) {
	t.Setenv("CUTOVER_ENV", "")
	t.Setenv("DEPLOY_ROOT", t.TempDir())
	if got := locateCutoverEnv(""); got != "" {
		t.Fatalf("locateCutoverEnv(\"\")=%q, want empty (must not guess from cwd)", got)
	}
}

func TestApplyCutoverEnvFile_ExpandsDefaultAndSetsDeployMode(t *testing.T) {
	restoreCutoverKeys(t)
	_ = os.Unsetenv("INFRA_HOST")
	_ = os.Unsetenv("DEPLOY_MODE")
	path := filepath.Join(t.TempDir(), "cutover.env")
	body := "export DEPLOY_MODE=1\nexport INFRA_HOST=${INFRA_HOST:-9.9.9.9}\nexport RUNALL_SKIP_BUILD=1\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applyCutoverEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DEPLOY_MODE") != "1" {
		t.Fatalf("DEPLOY_MODE=%q want 1", os.Getenv("DEPLOY_MODE"))
	}
	if os.Getenv("INFRA_HOST") != "9.9.9.9" {
		t.Fatalf("INFRA_HOST=%q want expanded default 9.9.9.9", os.Getenv("INFRA_HOST"))
	}
	if os.Getenv("RUNALL_SKIP_BUILD") != "1" {
		t.Fatalf("RUNALL_SKIP_BUILD=%q want 1", os.Getenv("RUNALL_SKIP_BUILD"))
	}
}

func TestApplyCutoverEnvFile_KeepsExistingInfraHost(t *testing.T) {
	restoreCutoverKeys(t)
	t.Setenv("INFRA_HOST", "192.168.1.10")
	path := filepath.Join(t.TempDir(), "cutover.env")
	body := "export DEPLOY_MODE=1\nexport INFRA_HOST=${INFRA_HOST:-10.2.150.68}\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applyCutoverEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("INFRA_HOST") != "192.168.1.10" {
		t.Fatalf("INFRA_HOST=%q want preserved 192.168.1.10", os.Getenv("INFRA_HOST"))
	}
}

func TestLoadConfig_SourcesCutoverWhenCUTOVER_ENVSet(t *testing.T) {
	restoreCutoverKeys(t)
	_ = os.Unsetenv("DEPLOY_MODE")
	dir := t.TempDir()
	cutover := filepath.Join(dir, "cutover.env")
	if err := os.WriteFile(cutover, []byte("export DEPLOY_MODE=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CUTOVER_ENV", cutover)
	yaml := `
version: "1"
groups:
  - name: infra
    services:
      - name: redis
        command: "redis-server"
        health_check:
          url: "http://localhost:6379"
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if os.Getenv("DEPLOY_MODE") != "1" {
		t.Fatalf("LoadConfig must source CUTOVER_ENV; DEPLOY_MODE=%q", os.Getenv("DEPLOY_MODE"))
	}
}
