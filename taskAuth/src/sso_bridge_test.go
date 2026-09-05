package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"confload"
)

// ssoTruthSourceValue 读取 conf/core/sso/config.yaml 真源 ssoJwtSecret。
// 动态解析而非硬编码，密钥轮换（OPT-20260806-062）后测试自动跟随，不再需要改码。
func ssoTruthSourceValue(t *testing.T, root string) string {
	t.Helper()
	var c struct {
		SSOJwtSecret string `yaml:"ssoJwtSecret"`
	}
	if err := confload.ReadAppConfig(root, "core/sso", &c); err != nil {
		t.Fatalf("read conf/core/sso: %v", err)
	}
	if c.SSOJwtSecret == "" {
		t.Skip("no conf-local SSO overlay on this machine")
	}
	return c.SSOJwtSecret
}

// TestResolveSSOJwtSecretFromSsoTruthSource guards the signer-side key contract:
// without env overrides, the SSO bridge signing secret must resolve to the
// conf/core/sso/config.yaml truth source value — the same value taskAiProvider
// verifies with. A divergence yields "无效的 bridge" on every exchange.
func TestResolveSSOJwtSecretFromSsoTruthSource(t *testing.T) {
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "")
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	want := ssoTruthSourceValue(t, root)
	got := resolveSSOJwtSecret(root)
	if got != want {
		t.Fatalf("resolveSSOJwtSecret = %q, want %q — 签发/验证密钥不一致，provider 将返回「无效的 bridge」", got, want)
	}
	// ssoBridgeSecret must use the resolved config value (not InternalSecret)。
	// 生产路径 loadConfig 会把 resolveSSOJwtSecret 结果写入 cfg.SSOJwtSecret，
	// ssoBridgeSecret 优先读它；此处模拟该赋值（测试进程未跑 loadConfig）。
	prev := cfg.SSOJwtSecret
	cfg.SSOJwtSecret = want
	defer func() { cfg.SSOJwtSecret = prev }()
	if s := ssoBridgeSecret(); s != want {
		t.Fatalf("ssoBridgeSecret = %q, want %q", s, want)
	}
}

// TestResolveSSOJwtSecretEnvOverride verifies TASK2APP_SSO_JWT_SECRET env
// overrides the yaml truth source (production key rotation path).
func TestResolveSSOJwtSecretEnvOverride(t *testing.T) {
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "prod-sso-secret-xyz")
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("findMonorepoRoot: %v", err)
	}
	if got := resolveSSOJwtSecret(root); got != "prod-sso-secret-xyz" {
		t.Fatalf("env override not honored: got %q", got)
	}
}

// TestSSOBridgeSecretIgnoresInternalSecret ensures the SSO signing secret is
// decoupled from the internal-API secret: setting TASKAUTH_INTERNAL_SECRET must
// NOT change the SSO bridge secret (two independent security domains).
func TestSSOBridgeSecretIgnoresInternalSecret(t *testing.T) {
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "")
	t.Setenv("TASKAUTH_INTERNAL_SECRET", "internal-api-secret")
	prev := cfg.InternalSecret
	cfg.InternalSecret = os.Getenv("TASKAUTH_INTERNAL_SECRET")
	defer func() { cfg.InternalSecret = prev }()

	got := ssoBridgeSecret()
	if got == "internal-api-secret" {
		t.Fatalf("ssoBridgeSecret must not use InternalSecret — got %q", got)
	}
	if got != "task2app-local-sso-bridge-dev-do-not-use-in-prod" {
		t.Fatalf("ssoBridgeSecret = %q, want truth-source default", got)
	}
}

// ── vendor_bridge email 解析（OPT-20260806-061：微信扫码用户无邮箱） ──

// decodeBridgeClaims 解码 taskAuth 签发的 bridge JWT payload（不验签 — 密钥契约
// 由 TestResolveSSOJwtSecretFromSsoTruthSource 与 provider 端 SSO 测试守护）。
func decodeBridgeClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("bridge token malformed: %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode bridge payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal bridge claims: %v", err)
	}
	return claims
}

// requestVendorBridge 以 gateway 前向认证头（X-User-Id）模拟微信扫码登录态用户
// 点击「厂商门户 SSO」。
func requestVendorBridge(t *testing.T, userID string) *httptest.ResponseRecorder {
	t.Helper()
	prev := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://gw.example.test"
	t.Cleanup(func() { cfg.GatewayPublicBase = prev })
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/sso/ai-provider/vendor/", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	handleSSOBridge(rec, req)
	return rec
}

// bridgeFromRedirect 断言 302 到 provider 换票端点并解码 bridge claims。
func bridgeFromRedirect(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	const prefix = "https://gw.example.test/api/auth/sso/exchange/?bridge="
	if !strings.HasPrefix(loc, prefix) {
		t.Fatalf("expected provider exchange redirect, got %s", loc)
	}
	return decodeBridgeClaims(t, strings.TrimPrefix(loc, prefix))
}

// mustAddEmailLoginMethod 为用户绑定邮箱登录方式（后续绑定场景）。
func mustAddEmailLoginMethod(t *testing.T, userID, email string) {
	t.Helper()
	now := timeNowUTC()
	if _, err := db.Exec(
		`INSERT INTO auth_login_method (id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at)
		 VALUES (?, ?, ?, 'email', ?, '', 1, ?, ?)`,
		generateSnowflakeID(), cfg.UserContentTypeID, userID, email, now, now); err != nil {
		t.Fatalf("insert email login method: %v", err)
	}
}

// TestSSOVendorBridgeRejectsNoEmail（OPT-20260806-065 行为反转）：微信扫码登录
// 用户（仅 method_type='wechat' 行，无邮箱）点击厂商门户 SSO 不得签发 vendor_bridge
// —— 重定向到主站个人中心（sso_error=email_required）引导绑定邮箱。此前（OPT-061）
// 行为是携带合成邮箱放行并自动建号，已按产品决策收紧。
func TestSSOVendorBridgeRejectsNoEmail(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-openid-sso-email-required-001")

	rec := requestVendorBridge(t, userID)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect to profile, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	// 与 taskFE emailBindingDeepLink.js 契约对齐（OPT-20260812-016）
	const want = "https://gw.example.test" + emailBindingDeepLinkSuffix
	if loc != want {
		t.Fatalf("expected profile redirect with email-binding deep link, got %s", loc)
	}
}

// TestSSOVendorBridgeEmailUsesRealLoginMethod：有邮箱登录方式的用户优先使用真实邮箱。
func TestSSOVendorBridgeEmailUsesRealLoginMethod(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-openid-sso-real-001")
	mustAddEmailLoginMethod(t, userID, "SSO-REAL@Example.com")

	claims := bridgeFromRedirect(t, requestVendorBridge(t, userID))
	want := "sso-real@example.com"
	if got := fmt.Sprint(claims["email"]); got != want {
		t.Fatalf("email claim = %q, want %q", got, want)
	}
}

// TestSSOVendorBridgeEmailRealBeatsSynthetic：微信 + 邮箱并存时真实邮箱优先
// （用户后续绑定邮箱后自动切换为真实邮箱，不再使用合成邮箱）。
func TestSSOVendorBridgeEmailRealBeatsSynthetic(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-openid-sso-mix-001")
	mustAddEmailLoginMethod(t, userID, "mix@example.com")

	claims := bridgeFromRedirect(t, requestVendorBridge(t, userID))
	if got := fmt.Sprint(claims["email"]); got != "mix@example.com" {
		t.Fatalf("email claim = %q, want real email %q", got, "mix@example.com")
	}
}

// TestSSOBridgeNoSessionRedirectsToLogin（OPT-20260901-021）：路由已改 public，
// 无 Cookie/Authorization 的浏览器导航由 handleSSOBridge 直接 302 到登录页，而不是
// forward-auth 时期的 401 JSON「无法解析登录凭据」。
func TestSSOBridgeNoSessionRedirectsToLogin(t *testing.T) {
	prev := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://gw.example.test"
	t.Cleanup(func() { cfg.GatewayPublicBase = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/sso/ai-provider/admin/", nil)
	rec := httptest.NewRecorder()
	handleSSOBridge(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 to login, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "https://gw.example.test/auth/login/" {
		t.Fatalf("Location=%q want login URL", loc)
	}
}
