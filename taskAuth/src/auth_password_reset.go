package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"tracelog"
)

func handleSendPasswordResetLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"detail": "method not allowed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	email := strField(body, "email")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "邮箱不能为空", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}

	lm, err := findLoginMethodByEmail(email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "db error", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if lm == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "用户未注册", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}

	token, err := generatePasswordResetToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "token generation failed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if err := setPasswordResetToken(lm.ID, token, expiresAdd24h()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "update failed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	// OPT-20260807-018 修复: reset_url 此前为裸相对路径，邮件客户端无正确基址，
	// 点击链接解析到邮件服务商域名 → 404。与邀请邮件一致，拼上 FrontendBase。
	if _, err := publishEmailSent(r.Context(), email, "密码重置 - SaaS平台", "password_reset", map[string]interface{}{
		"reset_url": buildUserFacingURL(fmt.Sprintf("/auth/reset-password/%s/", token)),
	}); err != nil {
		log.Printf("[taskAuth] password-reset email publish: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{"error": "发送密码重置链接失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "密码重置链接已发送"})
}

func handleResetPasswordWithLinkPrefix(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/users/reset-password-with-link/")
	token := strings.Trim(path, "/")
	handleResetPasswordWithLink(w, r, token)
}

func handleResetPasswordWithLink(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"detail": "method not allowed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "无效的密码重置链接", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	newPassword := strField(body, "new_password")
	if newPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"new_password": []string{"该字段是必填项"},
			"trace_id":     tracelog.TraceIDFromContext(r.Context()),
		})
		return
	}

	valid, lm := isPasswordResetTokenValid(token)
	if !valid || lm == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "无效的密码重置链接", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if err := updatePasswordHashClearResetToken(lm.ID, newPassword); err != nil {
		log.Printf("[taskAuth] reset password with link: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "密码重置失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	// 用户已通过邮箱链接自行设置新密码，清除强制改密标记（随机密码引导流程）
	if err := clearMustChangePassword(lm.ObjectID); err != nil {
		log.Printf("[taskAuth] reset password with link: clear must_change_password: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "密码重置成功"})
}

func handleGetResetUserInfoPrefix(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/users/get-reset-user-info/")
	token := strings.Trim(path, "/")
	handleGetResetUserInfo(w, r, token)
}

func handleGetResetUserInfo(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"detail": "method not allowed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "无效的重置链接", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	valid, lm := isPasswordResetTokenValid(token)
	if !valid || lm == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "无效的重置链接", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"identifier": lm.Identifier})
}

func handleSendPasswordResetCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"detail": "method not allowed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	phone := strField(body, "phone")
	email := strField(body, "email")
	if phone == "" && email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "必须提供手机号或邮箱", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if phone != "" {
		n, err := livePhoneBindingCount(phone)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "db error", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
		if n >= 2 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":    "该手机号绑定了多个账号，请使用邮箱重置密码",
				"code":     "phone_ambiguous",
				"detail":   "该手机号绑定了多个账号，请使用邮箱重置密码",
				"trace_id": tracelog.TraceIDFromContext(r.Context()),
			})
			return
		}
		lm, err := findLoginMethodForPasswordReset(phone, "")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "db error", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
		if lm == nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "用户未注册", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
		if _, err := sendPhoneVerificationCode(r.Context(), phone, smsKindPasswordReset); err != nil {
			log.Printf("[taskAuth] send password-reset sms: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "发送验证码失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "密码重置验证码已发送"})
		return
	}
	lm, err := findLoginMethodForPasswordReset("", email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "db error", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if lm == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "用户未注册", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if _, err := sendEmailVerificationCode(r.Context(), email, smsKindPasswordReset); err != nil {
		log.Printf("[taskAuth] send password-reset email: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "发送验证码失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "密码重置验证码已发送"})
}

func handleResetPasswordWithCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"detail": "method not allowed", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	phone := strField(body, "phone")
	email := strField(body, "email")
	code := strField(body, "code")
	newPassword := strField(body, "new_password")
	if code == "" || newPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "验证码和新密码不能为空", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	var ok bool
	if phone != "" {
		n, cntErr := livePhoneBindingCount(phone)
		if cntErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "db error", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
		if n >= 2 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":    "该手机号绑定了多个账号，请使用邮箱重置密码",
				"code":     "phone_ambiguous",
				"detail":   "该手机号绑定了多个账号，请使用邮箱重置密码",
				"trace_id": tracelog.TraceIDFromContext(r.Context()),
			})
			return
		}
		ok, err = verifyPhoneVerificationCode(phone, code, "")
		if err != nil {
			log.Printf("[taskAuth] verify password-reset code: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "验证码校验失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
	} else {
		ok, err = verifyEmailVerificationCode(email, code, "")
		if err != nil {
			log.Printf("[taskAuth] verify password-reset email code: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "验证码校验失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
			return
		}
	}
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "验证码无效或已过期", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	lm, err := findLoginMethodForPasswordReset(phone, email)
	if err != nil || lm == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "用户不存在", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	if err := updatePasswordHashOnly(lm.ID, newPassword); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "密码重置失败，请稍后重试", "trace_id": tracelog.TraceIDFromContext(r.Context())})
		return
	}
	// 用户已通过验证码自行设置新密码，清除强制改密标记（随机密码引导流程）
	if err := clearMustChangePassword(lm.ObjectID); err != nil {
		log.Printf("[taskAuth] reset password with code: clear must_change_password: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "密码重置成功"})
}

// buildUserFacingURL 拼出邮件正文可点击的绝对链接。裸相对路径在邮件客户端
// 无正确基址（解析到邮件服务商域名 → 404），与邀请邮件一致拼上 FrontendBase
// （OPT-20260807-018）。frontendBase 缺失时回退 http://localhost:4000。
func buildUserFacingURL(path string) string {
	frontendBase := cfg.FrontendBase
	if frontendBase == "" {
		frontendBase = "http://localhost:4000"
	}
	return strings.TrimRight(frontendBase, "/") + path
}

func generatePasswordResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
