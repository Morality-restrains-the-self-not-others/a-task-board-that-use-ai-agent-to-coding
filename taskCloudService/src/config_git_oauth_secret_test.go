package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveGitOauthBridgeSecretPlaceholderFallback 回归 OPT-20260809-015：
// 配置为本地占位符时回退到 SSOT ssoJwtSecret（防 taskGitOauth 401）。
func TestResolveGitOauthBridgeSecretPlaceholderFallback(t *testing.T) {
	got := resolveGitOauthBridgeSecret(localBridgePlaceholder, "ssot-secret-abc")
	if got != "ssot-secret-abc" {
		t.Fatalf("placeholder must fall back to SSOT, got %q", got)
	}
}

// TestResolveGitOauthBridgeSecretAligned — 配置与 SSOT 一致时保持原值，不误替换。
func TestResolveGitOauthBridgeSecretAligned(t *testing.T) {
	got := resolveGitOauthBridgeSecret("ssot-secret-abc", "ssot-secret-abc")
	if got != "ssot-secret-abc" {
		t.Fatalf("aligned secret must stay, got %q", got)
	}
}

// TestResolveGitOauthBridgeSecretDrift — 配置与 SSOT 不一致时保留显式配置（漂移告警由日志承担）。
func TestResolveGitOauthBridgeSecretDrift(t *testing.T) {
	got := resolveGitOauthBridgeSecret("explicit-secret", "ssot-secret-abc")
	if got != "explicit-secret" {
		t.Fatalf("explicit config must be kept on drift, got %q", got)
	}
}

// TestResolveGitOauthBridgeSecretEmpty — 空配置 + 无 SSOT → 本地占位符（原有兜底）。
func TestResolveGitOauthBridgeSecretEmpty(t *testing.T) {
	got := resolveGitOauthBridgeSecret("", "")
	if got != localBridgePlaceholder {
		t.Fatalf("empty + no SSOT must fall back to placeholder, got %q", got)
	}
	// 空配置 + SSOT 可用 → SSOT
	got = resolveGitOauthBridgeSecret("", "ssot-secret-abc")
	if got != "ssot-secret-abc" {
		t.Fatalf("empty must fall back to SSOT, got %q", got)
	}
}

// TestLoadGitOauthSSOSecretMergesConfLocal — tracked skeleton + conf-local overlay.
func TestLoadGitOauthSSOSecretMergesConfLocal(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "conf", "auth", "git-oauth")
	local := filepath.Join(root, "conf-local", "auth", "git-oauth")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "conf", "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "django.yaml"), []byte("ssoJwtSecret: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "django.yaml"), []byte("ssoJwtSecret: fixture-sso\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadGitOauthSSOSecret(root)
	if got != "fixture-sso" {
		t.Fatalf("got %q want fixture-sso", got)
	}
}

// TestLoadGitOauthSSOSecretFromDjango — 本机 conf-local 有密钥时可读且非占位符。
func TestLoadGitOauthSSOSecretFromDjango(t *testing.T) {
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	path := filepath.Join(root, "conf", "auth", "git-oauth", "django.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("missing conf/auth/git-oauth/django.yaml: %v", err)
	}
	got := loadGitOauthSSOSecret(root)
	if got == "" {
		t.Skip("no conf-local SSO overlay on this machine")
	}
	if got == localBridgePlaceholder {
		t.Fatalf("SSOT must not be the dev placeholder")
	}
}
