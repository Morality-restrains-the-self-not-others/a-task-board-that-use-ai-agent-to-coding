package main

import (
	"os"
	"testing"
)

// OPT-20260808-022: TASKAUTH_PORT 环境变量显式设置时必须优先于 conf/auth/task-auth
// config.yaml 的 port（dev 实例 8004 与生产 8003 并存的前提）。
func TestConfigPortEnvOverridesYAML(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	prevCfg := cfg
	prevPort := os.Getenv("TASKAUTH_PORT")
	_ = os.Setenv("TASKAUTH_PORT", "8004")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("TASKAUTH_PORT", prevPort)
	})

	cfg = Config{}
	loadConfig(repoRoot)
	if cfg.Port != 8004 {
		t.Fatalf("TASKAUTH_PORT=8004 set, got cfg.Port=%d (yaml port 覆盖了 env)", cfg.Port)
	}
}

func TestConfigPortFallsBackToYAMLWithoutEnv(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	prevCfg := cfg
	prevPort := os.Getenv("TASKAUTH_PORT")
	_ = os.Unsetenv("TASKAUTH_PORT")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("TASKAUTH_PORT", prevPort)
	})

	cfg = Config{}
	loadConfig(repoRoot)
	if cfg.Port != 8003 {
		t.Fatalf("no env set, got cfg.Port=%d, want 8003 (yaml fallback)", cfg.Port)
	}
}
