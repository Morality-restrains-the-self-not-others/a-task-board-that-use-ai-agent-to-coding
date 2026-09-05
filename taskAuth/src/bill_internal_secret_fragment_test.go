package main

import (
	"os"
	"testing"
)

// OPT-20260821-026: taskAuth 读 taskBill internalSecret 走本目录 conf-sync 片段
// auth/task-auth/task-bill.yaml（规则 29），禁止直读 conf/billing/task-bill。

func TestLoadBillInternalSecretFromSyncedFragment(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	prevCfg := cfg
	prevEnv := os.Getenv("TASKBILL_INTERNAL_SECRET")
	_ = os.Unsetenv("TASKBILL_INTERNAL_SECRET")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("TASKBILL_INTERNAL_SECRET", prevEnv)
	})

	cfg = Config{}
	loadBillAndCloudServiceURLs(repoRoot)
	if cfg.BillInternalSecret == "" {
		t.Fatal("expected BillInternalSecret from auth/task-auth/task-bill.yaml fragment")
	}
	if cfg.BillInternalSecret != "taskbill-local-dev-secret-do-not-use-in-prod" {
		t.Fatalf("unexpected secret %q", cfg.BillInternalSecret)
	}
}

func TestBillInternalSecretEnvWins(t *testing.T) {
	prevCfg := cfg
	prevEnv := os.Getenv("TASKBILL_INTERNAL_SECRET")
	_ = os.Setenv("TASKBILL_INTERNAL_SECRET", "env-secret-abc")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("TASKBILL_INTERNAL_SECRET", prevEnv)
	})

	cfg = Config{}
	loadBillAndCloudServiceURLs("")
	if cfg.BillInternalSecret != "env-secret-abc" {
		t.Fatalf("env should win: got %q", cfg.BillInternalSecret)
	}
}
