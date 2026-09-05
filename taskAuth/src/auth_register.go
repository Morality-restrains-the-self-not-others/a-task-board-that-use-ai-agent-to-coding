package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
)

func handleEmailRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	// 邮箱注册仅限邀请制：必须提供有效的 invite_token
	inviteToken := strings.TrimSpace(strField(body, "invite_token"))
	if inviteToken == "" {
		writeError(w, r, http.StatusBadRequest, "邮箱注册需要邀请，请使用手机号注册或联系管理员获取邀请")
		return
	}

	// 验证邀请 token
	var inviteEmail, inviteStatus string
	var inviteAssignedRole, inviteAccountExpires sql.NullString
	err = db.QueryRow(`
		SELECT email, status, COALESCE(assigned_role, ''), account_expires_at
		FROM auth_email_registration_invite
		WHERE token = ? AND status = 'pending' AND expires_at > ?`,
		inviteToken, timeNowUTC(),
	).Scan(&inviteEmail, &inviteStatus, &inviteAssignedRole, &inviteAccountExpires)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "邀请链接无效或已过期")
		return
	}

	email := strings.TrimSpace(strField(body, "email"))
	password := strField(body, "password")
	if email == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"email": []string{"邮箱和密码是必填项"}})
		return
	}

	// 邮箱必须与邀请中的邮箱一致
	if !strings.EqualFold(email, inviteEmail) {
		writeError(w, r, http.StatusBadRequest, "邮箱地址与邀请不符")
		return
	}

	if len(password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"password": []string{"密码至少 8 位"}})
		return
	}

	// 检查邮箱是否已注册
	existing, err := findLoginMethodByEmail(email)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if existing != nil {
		active := true
		if existing.ObjectID != "" {
			active, _ = userIsActive(existing.ObjectID)
		}
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"error":        "该邮箱已被注册，请直接登录",
			"user_existed": true,
			"is_active":    active,
		})
		return
	}

	n, err := voidStaleEmailLoginMethods(email)
	if err != nil {
		slog.ErrorContext(r.Context(), "email_register_void_stale_failed", "err", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if n > 0 {
		slog.InfoContext(r.Context(), "email_register_reclaimed_stale_identifier", "voided_count", n)
	}

	username := strField(body, "username")
	if username == "" {
		username = strings.Split(email, "@")[0]
	}

	// 创建用户（邮箱已验证，无需激活）
	userID, _, err := createUserWithEmailLogin(email, password)
	if err != nil {
		log.Printf("[taskAuth] email invite register error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "register failed")
		return
	}

	// 立即激活（邀请注册无需邮件激活）
	lm, err := findLoginMethodByEmail(email)
	if err == nil && lm != nil && !lm.IsVerified {
		_ = activateLoginMethod(lm.ID)
	}

	// 标记邀请为已接受
	now := timeNowUTC()
	_, _ = db.Exec(`
		UPDATE auth_email_registration_invite
		SET status = 'accepted', accepted_at = ?, accepted_user_id = ?
		WHERE token = ? AND status = 'pending'`,
		now, userID, inviteToken,
	)

	// 应用邀请中预设的角色和账号有效期
	applyInviteRoleAndExpiry(userID, inviteAssignedRole.String, inviteAccountExpires.String)

	// 生成 token 并发布事件
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, resolveClientIP(r))
	if err != nil {
		log.Printf("[taskAuth] email invite token error: %v", err)
	}

	// 发布 USER_CREATED 事件
	publishUserCreatedAsync(userID, "", email, username)
	bindReferralAfterRegisterAsync(userID, extractAccessCodeFromRegisterBody(body))

	touchLastLogin(userID)
	recordSuccessfulLoginFromRequest(r, userID, email, "email", "register", "")
	resp := buildLoginResponse(userID, token)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":      fmt.Sprintf("注册成功，欢迎 %s", email),
		"token":        token,
		"user":         resp["user"],
		"user_existed": false,
	})
}

// applyInviteRoleAndExpiry applies the preset role and account expiry from
// an accepted email invite to the newly created user.
func applyInviteRoleAndExpiry(userID, assignedRole, accountExpiresAt string) {
	if assignedRole == "" && accountExpiresAt == "" {
		return
	}

	// Apply role flags
	switch assignedRole {
	case "superuser":
		_, _ = db.Exec(`UPDATE auth_user SET is_superuser = 1, is_staff = 1 WHERE id = ?`, userID)
		_ = ensureSuperAdminRow(userID) // 内部已双写 role-super-admin RBAC 行
		log.Printf("[taskAuth] invite register: user %s granted superuser role", userID)
	case "staff":
		_, _ = db.Exec(`UPDATE auth_user SET is_staff = 1 WHERE id = ?`, userID)
		// v63 RBAC 双写：避免后续 /api/auth/user-roles/ 平台角色为空
		ensurePlatformRoleRows(userID, "role-employee")
		log.Printf("[taskAuth] invite register: user %s granted staff role", userID)
	case "tenant":
		_, _ = db.Exec(`UPDATE auth_user SET is_tenant = 1 WHERE id = ?`, userID)
		log.Printf("[taskAuth] invite register: user %s set as tenant", userID)
	case "member", "":
		// default: no extra flags
	default:
		log.Printf("[taskAuth] invite register: unknown assigned_role=%q for user %s, skipping", assignedRole, userID)
	}

	// Apply account expiry
	if accountExpiresAt != "" {
		_, dbErr := db.Exec(`UPDATE auth_user SET account_expires_at = ? WHERE id = ?`, accountExpiresAt, userID)
		if dbErr != nil {
			log.Printf("[taskAuth] invite register: failed to set account_expires_at for user %s: %v", userID, dbErr)
		} else {
			log.Printf("[taskAuth] invite register: user %s account_expires_at = %s", userID, accountExpiresAt)
		}
	}
}
