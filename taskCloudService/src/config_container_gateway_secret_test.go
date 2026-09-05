package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContainerGatewayInternalSecretFromConf(t *testing.T) {
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	cfgPath := filepath.Join(root, "conf", "gateway", "task-container-gateway", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("missing conf: %v", err)
	}
	prev := os.Getenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")
	_ = os.Unsetenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")
		} else {
			_ = os.Setenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET", prev)
		}
	})
	got := loadContainerGatewayInternalSecret(root)
	if got == "" {
		t.Fatal("expected non-empty container gateway internal secret from conf")
	}
	if got != "task-container-gateway-local-dev-secret-do-not-use-in-prod" {
		t.Fatalf("secret=%q", got)
	}
}

func TestLoadContainerGatewayInternalSecretEnvOverride(t *testing.T) {
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	prev := os.Getenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")
	_ = os.Setenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET", "env-override-secret")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")
		} else {
			_ = os.Setenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET", prev)
		}
	})
	got := loadContainerGatewayInternalSecret(root)
	if got != "env-override-secret" {
		t.Fatalf("secret=%q", got)
	}
}
