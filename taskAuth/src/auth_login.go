package main

import (
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"taskAuth/domain"
	"tracelog"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	// 手机号+验证码登录已移除（2026-08-24）：注册后须用手机号+密码登录；
	// 忘记密码走手机号+验证码重置（send_password_reset_code / reset_password_with_code）。
	// fail-closed：携带 code 且无 password 的请求一律 400，防止绕过前端直连，
	// 且不再支持验证码自动注册（须显式 phone_register）。
	if strField(body, "phone") != "" && strField(body, "code") != "" && strField(body, "password") == "" {
		// 留痕：验证码登录尝试（含直连 API 绕过）须可审计，与 not_found/password_mismatch 一致
		slog.WarnContext(r.Context(), "login rejected",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"reason", "phone_code_login_disabled",
		)
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"error":  "phone_code_login_disabled",
			"detail": "验证码登录已关闭，请使用手机号+密码登录",
		})
		return
	}

	email := strField(body, "email")
	username := strField(body, "username")
	password := strField(body, "password")
	phone := strField(body, "phone")

	identifier := email
	if identifier == "" {
		identifier = username
	}
	usePhoneLookup := identifier == "" && phone != ""
	if usePhoneLookup {
		identifier = phone
		// Gate phone+password login against region whitelist (local check, no Django)
		if cc, _ := splitCountryCallingCodeAndNational(canonicalPhoneForSMSAndLogin(phone)); cc != "" && !isPhoneCountryCodeAllowed(cc) {
			writeErrorMap(w, r, http.StatusForbidden, map[string]interface{}{
				"error":  "phone_region_not_allowed",
				"detail": "该地区手机号暂不支持登录",
			})
			return
		}
	}
	if identifier == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"non_field_errors": []string{"必须提供有效的登录方式"},
		})
		return
	}

	// Phone methods store identifier as national number + phone_country_calling_code.
	// Frontend submits E.164 (+86…); identifier exact-match would 400 before bcrypt.
	var lm *LoginMethodRow
	if usePhoneLookup {
		lm, err = matchPhoneLoginMethodByPassword(canonicalPhoneForSMSAndLogin(phone), password)
		if errors.Is(err, errPhoneAmbiguous) {
			slog.WarnContext(r.Context(), "login rejected",
				"trace_id", tracelog.TraceIDFromContext(r.Context()),
				"reason", "phone_ambiguous",
				"method", "phone",
			)
			writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
				"error":  "phone_ambiguous",
				"detail": "该手机号绑定了多个账号，请使用邮箱或用户名登录",
				"code":   "phone_ambiguous",
			})
			return
		}
	} else {
		lm, err = findLoginMethodByIdentifier(identifier)
	}
	if err != nil {
		log.Printf("[taskAuth] login db error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if lm == nil {
		loginMethod := "username"
		if email != "" {
			loginMethod = "email"
		} else if usePhoneLookup {
			loginMethod = "phone"
		}
		slog.WarnContext(r.Context(), "login rejected",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"reason", "not_found",
			"method", loginMethod,
		)
		writeError(w, r, http.StatusBadRequest, passwordLoginRejectMessage(email, phone))
		return
	}
	if lm.MethodType == "email" && !lm.IsVerified {
		recordFailedLoginFromRequest(r, lm.ObjectID, domain.LoginOutcomeEmailUnverified, lm.MethodType, domain.LoginEntryCustomer)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"non_field_errors": []string{"邮箱未验证，请先点击邮件中的激活链接激活账号"},
		})
		return
	}
	if lm.PasswordHash == "" || !checkPasswordHash(password, lm.PasswordHash) {
		loginMethod := lm.MethodType
		if loginMethod == "" {
			loginMethod = "username"
		}
		slog.WarnContext(r.Context(), "login rejected",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"reason", "password_mismatch",
			"method", loginMethod,
		)
		recordFailedLoginFromRequest(r, lm.ObjectID, domain.LoginOutcomePasswordMismatch, loginMethod, domain.LoginEntryCustomer)
		writeError(w, r, http.StatusBadRequest, passwordLoginRejectMessage(email, phone))
		return
	}

	// 入口分离（OPT-20260824-001）：客户登录入口拒绝管理员账号。
	// 凭据校验通过后才判角色 —— 密码错误与账号不存在保持同一口径，不暴露角色信息。
	if !loginEntryAllowedForRole(w, r, lm.ObjectID, "customer") {
		return
	}

	finalizeLogin(w, r, lm.ObjectID, identifier, lm.MethodType, "customer", "账号未激活，请先激活账号")
}

// handleAdminLogin 是管理员专用登录入口（POST /api/auth/admin-login/）。
// 仅接受平台管理员/员工账号（super_admin/employee 平台角色或 is_superuser/is_staff
// 遗留标志）；普通客户账号即使凭据正确也一律 403，不允许走此入口。
func handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	email := strField(body, "email")
	username := strField(body, "username")
	password := strField(body, "password")
	identifier := email
	if identifier == "" {
		identifier = username
	}
	if identifier == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"non_field_errors": []string{"必须提供有效的登录方式"},
		})
		return
	}

	// 管理员入口仅支持邮箱/用户名+密码（无手机号/验证码路径），与 AdminLogin.vue 表单一致。
	lm, err := findLoginMethodByIdentifier(identifier)
	if err != nil {
		log.Printf("[taskAuth] admin login db error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if lm == nil {
		writeError(w, r, http.StatusBadRequest, passwordLoginRejectMessage(email, ""))
		return
	}
	if lm.MethodType == "email" && !lm.IsVerified {
		recordFailedLoginFromRequest(r, lm.ObjectID, domain.LoginOutcomeEmailUnverified, lm.MethodType, domain.LoginEntryAdmin)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"non_field_errors": []string{"邮箱未验证，请先点击邮件中的激活链接激活账号"},
		})
		return
	}
	if lm.PasswordHash == "" || !checkPasswordHash(password, lm.PasswordHash) {
		recordFailedLoginFromRequest(r, lm.ObjectID, domain.LoginOutcomePasswordMismatch, lm.MethodType, domain.LoginEntryAdmin)
		writeError(w, r, http.StatusBadRequest, passwordLoginRejectMessage(email, ""))
		return
	}

	// 入口分离：非管理员账号禁止使用管理员登录入口（凭据已校验通过才判角色）。
	adminStaff, err := isPlatformAdminStaff(lm.ObjectID)
	if err != nil {
		log.Printf("[taskAuth] admin login role check error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !adminStaff {
		slog.WarnContext(r.Context(), "admin login rejected",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"reason", "not_admin_staff",
			"user", lm.ObjectID,
		)
		recordFailedLoginFromRequest(r, lm.ObjectID, domain.LoginOutcomeNotAdminStaff, lm.MethodType, domain.LoginEntryAdmin)
		writeErrorMap(w, r, http.StatusForbidden, map[string]interface{}{
			"error":  "not_admin_account",
			"detail": "该账号非管理员账号，请使用普通用户登录入口",
		})
		return
	}

	finalizeLogin(w, r, lm.ObjectID, identifier, lm.MethodType, "admin", "账号未激活，请先激活账号")
}

// loginEntryAllowedForRole 校验账号角色与登录入口是否匹配（入口分离）：
//   - entry=="admin"：仅平台管理员/员工放行，其余 403（handleAdminLogin 内联实现）
//   - entry=="customer"：管理员账号拒绝（凭据已通过），其余放行
//
// 返回 false 表示已写错误响应。
func loginEntryAllowedForRole(w http.ResponseWriter, r *http.Request, userID, entry string) bool {
	adminStaff, err := isPlatformAdminStaff(userID)
	if err != nil {
		log.Printf("[taskAuth] login role check error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return false
	}
	if entry == "customer" && adminStaff {
		slog.WarnContext(r.Context(), "login rejected",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"reason", "admin_requires_admin_login",
			"user", userID,
		)
		recordFailedLoginFromRequest(r, userID, domain.LoginOutcomeAdminEntryMismatch, "", domain.LoginEntryCustomer)
		writeErrorMap(w, r, http.StatusForbidden, map[string]interface{}{
			"error":  "admin_requires_admin_login",
			"detail": "管理员账号请使用管理员登录入口",
		})
		return false
	}
	return true
}

// isPlatformAdminStaff 判定账号是否属于平台管理员/员工：
// super_admin / employee 平台角色，或遗留 is_superuser / is_staff 标志
// （is_staff ⇔ employee，见 rbac_platform_roles.go 标志同步逻辑）。
// 账号不存在（sql.ErrNoRows）视为非管理员 —— 登录前凭据已校验，属防御性兜底。
func isPlatformAdminStaff(userID string) (bool, error) {
	_, isSuper, isStaff, _, err := loadUserAuthFlags(userID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if isSuper || isStaff {
		return true, nil
	}
	roles, err := loadPlatformRoles(userID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role == "super_admin" || role == "employee" {
			return true, nil
		}
	}
	return false, nil
}

// finalizeLogin 执行登录成功后的公共收尾：激活校验 → 签发 token → 记录登录历史 →
// 发布 USER_LOGGED_IN 事件 → 更新 last_login → 组装响应。inactiveMsg 区分入口文案。
// 返回 false 表示已写错误响应。
func finalizeLogin(w http.ResponseWriter, r *http.Request, userID, identifier, methodType, entry, inactiveMsg string) bool {
	active, err := userIsActive(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return false
	}
	if !active {
		recordFailedLoginFromRequest(r, userID, domain.LoginOutcomeNotActive, methodType, entry)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"non_field_errors": []string{inactiveMsg},
		})
		return false
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, resolveClientIP(r))
	if err != nil {
		log.Printf("[taskAuth] token error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"detail":   "token error",
			"trace_id": tracelog.TraceIDFromContext(r.Context()),
		})
		return false
	}

	recordSuccessfulLoginFromRequest(r, userID, identifier, methodType, entry, "")
	touchLastLogin(userID)
	resp := buildLoginResponse(userID, token)

	// Check if user must change password (set by dataMigrate for bootstrap accounts)
	if mustChange, err := getUserMustChangePassword(userID); err == nil && mustChange {
		resp["must_change_password"] = true
	}

	writeJSON(w, http.StatusOK, resp)
	return true
}

func passwordLoginRejectMessage(email, phone string) string {
	if email != "" {
		return "邮箱或密码错误"
	}
	if phone != "" {
		return "手机号或密码错误"
	}
	return "用户名或密码错误"
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	auth := r.Header.Get("Authorization")
	key := strings.TrimPrefix(auth, "Token ")
	if key == auth {
		key = strings.TrimSpace(auth)
	}
	if key != "" && key != auth {
		_ = deleteTokenByKey(key)
	}
	// 同步清除 userId/token 会话 cookie（双变体）。服务端仅删 token 时，浏览器
	// 30 天 HttpOnly cookie 残留且无法被 JS 删除 → 各子域 reload 后仍带失效
	// cookie → forward-auth 401 → 反复跳登录。域变体清空保证一次登出全子域生效。
	clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
