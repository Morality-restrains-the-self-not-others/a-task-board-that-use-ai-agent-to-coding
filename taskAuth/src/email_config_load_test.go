package main

import (
	"os"
	"testing"
)

func TestLoadEmailConfigFromSyncedFragment(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	prevCfg := cfg
	prevUser := os.Getenv("EMAIL_HOST_USER")
	prevFrom := os.Getenv("EMAIL_DEFAULT_FROM")
	_ = os.Unsetenv("EMAIL_HOST_USER")
	_ = os.Unsetenv("EMAIL_DEFAULT_FROM")
	_ = os.Unsetenv("EMAIL_HOST")
	_ = os.Unsetenv("EMAIL_HOST_PASSWORD")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("EMAIL_HOST_USER", prevUser)
		_ = os.Setenv("EMAIL_DEFAULT_FROM", prevFrom)
	})

	cfg = Config{}
	loadEmailConfigFromSyncedFragment(repoRoot)
	if cfg.EmailHostUser == "" {
		t.Fatal("expected host_user from auth/task-auth/email.yaml")
	}
	if cfg.EmailDefaultFrom == "" {
		t.Fatal("expected default_from from synced email.yaml")
	}
	if cfg.EmailHost == "" {
		t.Fatal("expected smtp host from synced email.yaml")
	}
	smtp := loadSMTPConfig()
	if smtp.DefaultFrom == "" {
		t.Fatal("loadSMTPConfig DefaultFrom empty — would cause empty from address on EMAIL_SENT fallback")
	}
	if smtp.User == "" {
		t.Fatal("loadSMTPConfig User empty")
	}
	if smtp.DefaultFrom != cfg.EmailDefaultFrom && smtp.DefaultFrom != cfg.EmailHostUser {
		t.Fatalf("DefaultFrom=%q want synced from/user", smtp.DefaultFrom)
	}
}

func TestLoadSMTPConfigEnvOverridesSyncedYAML(t *testing.T) {
	prevCfg := cfg
	prevUser := os.Getenv("EMAIL_HOST_USER")
	prevFrom := os.Getenv("EMAIL_DEFAULT_FROM")
	cfg.EmailHostUser = "yaml@example.com"
	cfg.EmailDefaultFrom = "yaml-from@example.com"
	cfg.EmailHost = "smtp.example.com"
	_ = os.Setenv("EMAIL_HOST_USER", "env@example.com")
	_ = os.Unsetenv("EMAIL_DEFAULT_FROM")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("EMAIL_HOST_USER", prevUser)
		_ = os.Setenv("EMAIL_DEFAULT_FROM", prevFrom)
	})
	smtp := loadSMTPConfig()
	if smtp.User != "env@example.com" {
		t.Fatalf("env should win for User: got %q", smtp.User)
	}
	if smtp.DefaultFrom != "yaml-from@example.com" {
		t.Fatalf("DefaultFrom should keep yaml when EMAIL_DEFAULT_FROM unset: got %q", smtp.DefaultFrom)
	}
	_ = os.Setenv("EMAIL_DEFAULT_FROM", "env-from@example.com")
	smtp2 := loadSMTPConfig()
	if smtp2.DefaultFrom != "env-from@example.com" {
		t.Fatalf("EMAIL_DEFAULT_FROM should win: got %q", smtp2.DefaultFrom)
	}
}
