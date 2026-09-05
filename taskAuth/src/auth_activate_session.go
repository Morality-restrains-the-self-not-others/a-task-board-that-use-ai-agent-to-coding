package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

// handleActivateSession re-activates a browser session for an already-issued
// auth token (multi-account switcher). Does not revoke other users' tokens.
func handleActivateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tokenKey := tokenFromRequest(r)
	if tokenKey == "" || !strings.HasPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Token ") {
		writeError(w, r, http.StatusUnauthorized, "未提供有效令牌")
		return
	}

	userID, err := resolveTokenUserIDWithIP(tokenKey, resolveClientIP(r))
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[taskAuth] activate-session: invalid token")
			writeError(w, r, http.StatusUnauthorized, "令牌无效或已过期")
			return
		}
		if err == errTokenIPMismatch {
			// IP-bound token presented from a different network — an auth failure,
			// not a server fault. Client should re-login from the current network.
			log.Printf("[taskAuth] activate-session: token ip mismatch")
			writeErrorDetail(w, r, http.StatusUnauthorized, "登录环境已变化，请重新登录")
			return
		}
		log.Printf("[taskAuth] activate-session db error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if strings.TrimSpace(userID) == "" {
		writeError(w, r, http.StatusUnauthorized, "令牌无效或已过期")
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	expectedUID := strField(body, "user_id")
	if expectedUID != "" && expectedUID != userID {
		log.Printf("[taskAuth] activate-session user_id mismatch token_user=%s body_user=%s", userID, expectedUID)
		writeError(w, r, http.StatusBadRequest, "账号与令牌不匹配")
		return
	}

	active, err := userIsActive(userID)
	if err != nil {
		log.Printf("[taskAuth] activate-session userIsActive: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !active {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"non_field_errors": []string{"账号已被禁用"},
		})
		return
	}

	touchLastLogin(userID)
	recordSuccessfulLoginFromRequest(r, userID, userID, "session", "session_activate", "")
	resp := buildActivateSessionResponse(userID, tokenKey)
	// Set userId cookie as HttpOnly for cross-port SSO bridge (OIDC).
	// This replaces the JS-writable cookie previously set by the frontend.
	// OPT-20260807-004: 值改为签名形态 <id>.<ts>.<hmac>（裸 ID 可被同域侧伪造）；
	// 解析侧验签 + 7 天时效，详见 user_id_cookie.go。签名失败时不落 cookie
	//（token cookie 与响应体仍是主认证路径，不影响登录）。
	signedUID, sigErr := signUserIDCookieValue(userID)
	if sigErr != nil {
		log.Printf("[taskAuth] activate-session: sign userId cookie: %v", sigErr)
		signedUID = ""
	}
	// Domain: ssoCookieDomain() — 种到父域（.daydaymoney.com），保证 www 与裸域
	// （GitHub OAuth 回调落地 daydaymoney.com）共享会话 cookie；否则回调页 SPA
	// 请求不带 token → 401 → 跳登录页（OPT-20260807-071 根因）。见 sso_session_cookie.go。
	http.SetCookie(w, &http.Cookie{
		Name:     "userId",
		Value:    signedUID,
		Path:     "/",
		Domain:   ssoCookieDomain(),
		MaxAge:   2592000, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // Allow HTTP in dev; set to true in production via proxy
	})
	// Set token cookie alongside userId so the Chrome plugin can auto-detect
	// web login state via chrome.cookies API.  Same token, cookie transport.
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenKey,
		Path:     "/",
		Domain:   ssoCookieDomain(),
		MaxAge:   2592000, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // Allow HTTP in dev; set to true in production via proxy
	})
	// 清除 071 修复前旧登录遗留的 host-only 影子 cookie（HttpOnly，前端 JS 无法删除，
	// 只能服务端清除）。影子 token 更旧（可能已失效），且 host-only 在 www 上优先于
	// 域 cookie 被发送 → 影子不除，上面新种的域 cookie 在 www 形同虚设（旧 token
	// 失效 → www 永久 401，而裸域/其他子域正常）。登录即清，保证本次会话在任意
	// 子域立即生效。
	for _, name := range []string{"userId", "token"} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
	}
	log.Printf("[taskAuth] activate-session ok user=%s", userID)
	writeJSON(w, http.StatusOK, resp)
}
