package infrastructure

import (
	"testing"
	"time"

	"confload"
)

// confSsoTruthSourceValue is the SSO bridge shared secret from the conf truth
// source conf/core/sso/config.yaml (consumed by runAll conf_loader →
// task2appSsoJwtSecret). taskAuth signs SSO bridge JWTs with this value
// (taskAuth/src/sso_bridge.go ssoBridgeSecret fallback); the AI provider must
// verify with the same value or every exchange fails with "无效的 bridge".
//
// 动态解析 YAML 真源而非硬编码 —— 密钥轮换（OPT-20260806-062）后测试自动跟随。
func confSsoTruthSourceValue(t *testing.T, root string) string {
	t.Helper()
	var c struct {
		SSOJwtSecret string `yaml:"ssoJwtSecret"`
	}
	if err := confload.ReadAppConfig(root, "core/sso", &c); err != nil {
		t.Fatalf("conf/core/sso: %v", err)
	}
	if c.SSOJwtSecret == "" {
		t.Skip("no conf-local SSO overlay on this machine")
	}
	return c.SSOJwtSecret
}

// TestLoadConfigSSOJwtSecretMatchesSsoTruthSource guards the config-resolution
// chain: LoadConfig must resolve SSOJwtSecret from conf/core/sso/config.yaml
// (the retired conf/core/django/config.yaml and conf/ai/ai-provider/django.yaml
// paths are gone — falling back to SecretKey breaks the cross-service contract).
func TestLoadConfigSSOJwtSecretMatchesSsoTruthSource(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot: %v", err)
	}

	confSsoTruthSourceValue := confSsoTruthSourceValue(t, root)

	// Env overrides must not interfere with the default resolution.
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "")
	t.Setenv("DJANGO_SECRET_KEY", "")
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.SSOJwtSecret != confSsoTruthSourceValue {
		t.Fatalf("SSOJwtSecret = %q, want %q — signer/verifier secrets diverge, SSO bridge yields 无效的 bridge",
			cfg.SSOJwtSecret, confSsoTruthSourceValue)
	}
}

// TestLoadConfigSSOJwtSecretRejectsLegacyFallbackSecret pins the 2026-08-06
// production failure mode: before the fix, the SSOJwtSecret fallback chain
// resolved to SecretKey (DefaultSecretKey) — signing with that value must be
// REJECTED by the LoadConfig-resolved config (signer/verifier divergence).
func TestLoadConfigSSOJwtSecretRejectsLegacyFallbackSecret(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot: %v", err)
	}
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "")
	t.Setenv("DJANGO_SECRET_KEY", "")
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	now := time.Now().Unix()
	claims := map[string]any{
		"iss": "task2app-sso",
		"aud": "saas-ai-provider",
		"sub": "42",
		"typ": "staff_bridge",
		"iat": now,
		"exp": now + 60,
	}
	// The OLD broken fallback value (DefaultSecretKey) — must NOT verify.
	legacyBridge, err := SignHS256(DefaultSecretKey, claims)
	if err != nil {
		t.Fatalf("SignHS256: %v", err)
	}
	if _, err := ParseHS256JWT(legacyBridge, cfg.SSOJwtSecret, cfg.SSOJwtIssuer, cfg.SSOAudience, ""); err == nil {
		t.Fatal("bridge signed with DefaultSecretKey must be rejected — SSOJwtSecret fell back to SecretKey again (无效的 bridge regression)")
	}
}

// TestSSOBridgeCrossServiceContract signs a bridge JWT exactly like taskAuth
// (taskAuth/src/sso_bridge.go) and verifies it with the LoadConfig-resolved
// SSOJwtSecret + issuer/audience — the full signer→verifier round trip.
func TestSSOBridgeCrossServiceContract(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot: %v", err)
	}
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "")
	t.Setenv("DJANGO_SECRET_KEY", "")
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Reproduce the taskAuth bridge claims (sso_bridge.go constants).
	now := time.Now().Unix()
	claims := map[string]any{
		"iss": "task2app-sso",
		"aud": "saas-ai-provider",
		"sub": "42",
		"typ": "staff_bridge",
		"iat": now,
		"exp": now + 60,
	}
	bridge, err := SignHS256(confSsoTruthSourceValue(t, root), claims)
	if err != nil {
		t.Fatalf("SignHS256: %v", err)
	}

	parsed, err := ParseHS256JWT(bridge, cfg.SSOJwtSecret, cfg.SSOJwtIssuer, cfg.SSOAudience, "")
	if err != nil {
		t.Fatalf("taskAuth-signed bridge rejected by LoadConfig-resolved config: %v", err)
	}
	if ClaimString(parsed, "typ") != "staff_bridge" {
		t.Fatalf("typ = %q, want staff_bridge", ClaimString(parsed, "typ"))
	}
}
