package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---- SSO bridge: /api/accounts/sso/ai-provider/{vendor|admin}/ ---- //
// OPT-049: Migrated from Django sso_bridge_token.py (retired 2026-07-30).
// Generates a JWT signed with HMAC-SHA256 containing the user's ID as sub,
// then redirects the browser to the AI provider's SSO exchange endpoint.
// The AI provider validates the JWT and issues a local auth token.
// 签名密钥契约：secret=ssoJwtSecret（conf/core/sso/config.yaml 真源）——
// 与 taskAiProvider 的 SSOJwtSecret 解析链（TASK2APP_SSO_JWT_SECRET env → 同文件 → SecretKey 兜底）
// 必须收敛到同一值，否则 provider 拒绝 bridge（「无效的 bridge」）。

// SSO bridge config defaults match the AI provider's SSOJwtIssuer/SSOAudience.
const (
	ssoJwtIssuer  = "task2app-sso"
	ssoJwtAudience = "saas-ai-provider"
	ssoJwtTTL     = 60 // seconds
)

// emailBindingDeepLinkSuffix：无邮箱用户引导绑定邮箱的 302 Location 后缀。
// 与 taskFE app/src/utils/emailBindingDeepLink.js 的 EMAIL_BINDING_RG_KEY /
// EMAIL_REQUIRED_SSO_ERROR 共用同一深链契约（OPT-20260812-016），拼接在
// cfg.GatewayPublicBase 之后；变更两处需同步。
const emailBindingDeepLinkSuffix = "/profile/?sso_error=email_required#rg=profile.email_binding"

func signHS256(secret string, claims map[string]any) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mid := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + mid))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + mid + "." + sig, nil
}

func ssoBridgeSecret() string {
	// 契约（2026-07-16 ai-provider Go 迁移设计）：SSO bridge secret = ssoJwtSecret。
	// 真源 conf/core/sso/config.yaml（OPT-20260806-057 后 runAll conf_loader 亦读此文件），
	// 由 loadConfig 的 resolveSSOJwtSecret 解析（TASK2APP_SSO_JWT_SECRET env 优先）。
	// 注意：不复用 InternalSecret —— 那是内部 API 认证密钥（X-Internal-Secret），与 SSO
	// 签名分属独立安全域；复用会在生产单独配置 InternalSecret 时静默断开 SSO
	// （签发/验证密钥不一致 → provider 返回「无效的 bridge」）。
	if s := strings.TrimSpace(cfg.SSOJwtSecret); s != "" {
		return s
	}
	return "task2app-local-sso-bridge-dev-do-not-use-in-prod"
}

func aiProviderPublicBase() string {
	// The AI provider's public base URL (SSO redirect target).
	// Uses the AI provider's own subdomain (bypasses APISIX gateway),
	// with fallback to GatewayPublicBase for backward compatibility.
	if s := strings.TrimSpace(cfg.AiProviderPublicBase); s != "" {
		return strings.TrimRight(s, "/")
	}
	return strings.TrimRight(cfg.GatewayPublicBase, "/")
}

// resolveUserEmail 返回用户主邮箱（auth_login_method method_type='email' 且未解绑）。
// 无邮箱登录方式的用户（典型：微信扫码登录，仅 method_type='wechat' 行）返回
// 基于用户 ID 的确定性合成邮箱 —— 位于 RFC 2606 保留的 .invalid TLD，永不可投递、
// 明确非用户数据。厂商门户 vendor_bridge 交换依赖 email claim 建号/匹配
// （provider UpsertVendorFromBridge 以 saas_user_id 为主键，email 仅作兜底匹配
// 与展示，见 taskAiProvider/infrastructure/store_auth.go）。用户后续绑定真实邮箱后
// 真实邮箱优先，provider 侧 "email changed" 分支会同步更新厂商档案邮箱。
func resolveUserEmail(userID string) string {
	var email string
	err := db.QueryRow(
		`SELECT identifier FROM auth_login_method
		 WHERE object_id = ? AND method_type = 'email' AND binding_voided_at IS NULL
		 LIMIT 1`, userID,
	).Scan(&email)
	if err == nil && strings.TrimSpace(email) != "" {
		return strings.ToLower(strings.TrimSpace(email))
	}
	log.Printf("[taskAuth] event=sso_bridge_synthetic_email user_id=%s", userID)
	return "sso-" + userID + "@sso.invalid"
}

func handleSSOBridge(w http.ResponseWriter, r *http.Request) {
	// Only GET — browser navigation
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Resolve user identity from gateway forward-auth headers or token.
	// OPT-20260901-021: 该路由已改为 public（无 forward-auth），浏览器导航仅带 Cookie、
	// 无 Authorization/X-User-Id —— 回退按 Cookie（token/userId）解析登录态。仍无会话则
	// 302 到登录页（而非 401 JSON，无痕/过期会话点击 SSO 能看到登录页）。
	userID, ok := resolveUserIDFromRequest(r)
	if !ok || userID == "" {
		if uid, err := resolveUserIDForForwardAuth(r); err == nil && uid != "" {
			userID, ok = uid, true
		}
	}
	if !ok || userID == "" {
		// Redirect to login page instead of returning 401 (browser navigation)
		loginURL := cfg.GatewayPublicBase + "/auth/login/"
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	// Determine SSO type from path: /api/accounts/sso/ai-provider/{vendor|admin}/
	// trim trailing slash then split: the last path segment is the type
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var ssoType string
	if len(segments) > 0 && segments[len(segments)-1] == "vendor" {
		ssoType = "vendor_bridge"
	} else {
		ssoType = "staff_bridge"
	}

	// Log if userID is not a numeric string — the AI provider's ExchangeBridge
	// prefers int64 sub, but we still forward non-numeric IDs. The provider
	// must handle both cases (see ExchangeBridge fallback in store_auth.go).
	if _, err := strconv.ParseInt(userID, 10, 64); err != nil {
		log.Printf("[taskAuth] event=sso_bridge_non_numeric_user_id user_id=%q err=%v", userID, err)
	}

	now := time.Now().Unix()
	claims := map[string]any{
		"iss": ssoJwtIssuer,
		"aud": ssoJwtAudience,
		"sub": userID,
		"typ": ssoType,
		"iat": now,
		"exp": now + ssoJwtTTL,
	}

	// Resolve email from login methods (required by AI provider vendor_bridge exchange).
	// OPT-20260806-065：厂商门户需绑定邮箱账号后方可申请/使用。无邮箱登录方式的
	// 用户（典型：微信扫码，仅 method_type='wechat' 行）resolveUserEmail 返回确定性
	// 合成邮箱（sso-<id>@sso.invalid，RFC 2606 保留域）——此时拒绝签发 vendor_bridge，
	// 重定向到主站个人中心引导绑定邮箱（绑定后后续请求自然放行，既有厂商档案经
	// provider "email changed" 分支同步更新邮箱）。
	if ssoType == "vendor_bridge" {
		email := resolveUserEmail(userID)
		if strings.HasSuffix(email, "@sso.invalid") {
			log.Printf("[taskAuth] event=sso_bridge_vendor_email_required user_id=%s", userID)
			profileURL := cfg.GatewayPublicBase + emailBindingDeepLinkSuffix
			http.Redirect(w, r, profileURL, http.StatusFound)
			return
		}
		claims["email"] = email
	}

	// Resolve username/display name from profile (required by AI provider staff_bridge exchange)
	// Without this, the AI provider falls back to the numeric user ID as the username.
	if ssoType == "staff_bridge" {
		var username string
		_ = db.QueryRow(
			`SELECT COALESCE(username, '') FROM auth_user_profile WHERE user_id = ?`,
			userID,
		).Scan(&username)
		if username != "" {
			claims["username"] = username
			claims["name"] = username
		}
	}

	token, err := signHS256(ssoBridgeSecret(), claims)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "sso bridge error")
		return
	}

	// Redirect browser to AI provider SSO exchange
	aiBase := aiProviderPublicBase()
	target := fmt.Sprintf("%s/api/auth/sso/exchange/?bridge=%s", aiBase, token)
	http.Redirect(w, r, target, http.StatusFound)
}
