package main

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

// handleBindEmail POST /api/accounts/users/bind_email/
// 登录态绑定邮箱（OPT-20260806-065/066 厂商门户前置条件：微信扫码等无邮箱登录
// 方式的用户须先绑定真实邮箱方可申请/使用厂商门户；个人资料页邮箱绑定面板入口）。
// 语义与 handleBindPhone 对齐：「绑定」而非「注册」— 验证码确认后 upsert 到当前
// 账号；邮箱已被其他账号使用 → 409（errEmailTaken）。
// 绑定不设密码（password_hash 留空）：邮箱作为身份凭证（厂商资格校验 / SSO
// vendor bridge email claim），密码登录由既有密码重置链路补充。
// 已绑定邮箱换绑：同人更新 identifier（is_verified=1，重置 binding_voided_at）。
func handleBindEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.ToLower(strings.TrimSpace(strField(body, "email")))
	code := strings.TrimSpace(strField(body, "code"))
	if email == "" || code == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "必须提供邮箱与验证码")
		return
	}
	// 拒绝合成邮箱（sso-<id>@sso.invalid，RFC 2606 保留域）与明显非法格式
	if !strings.Contains(email, "@") || strings.HasSuffix(email, "@sso.invalid") {
		writeErrorDetail(w, r, http.StatusBadRequest, "邮箱格式无效")
		return
	}

	ok, err := verifyEmailVerificationCode(email, code, userID)
	if err != nil {
		log.Printf("[taskAuth] bind email verify db error user=%s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "内部错误")
		return
	}
	if !ok {
		writeErrorDetail(w, r, http.StatusBadRequest, "验证码无效或已过期")
		return
	}

	if err := upsertEmailLoginMethod(userID, email); err != nil {
		if errors.Is(err, errEmailTaken) {
			writeErrorDetail(w, r, http.StatusConflict, "该邮箱已绑定其他账号")
			return
		}
		log.Printf("[taskAuth] bind email upsert user=%s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "绑定失败")
		return
	}

	log.Printf("[taskAuth] email bound user=%s email=%s", userID, email)
	writeJSON(w, http.StatusOK, map[string]interface{}{"bound": true, "email": email})
}
