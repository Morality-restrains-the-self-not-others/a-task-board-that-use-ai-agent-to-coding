package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

func requireInternalSecret(r *http.Request) bool {
	if cfg.InternalSecret == "" {
		return true
	}
	return r.Header.Get("X-TaskAuth-Internal-Secret") == cfg.InternalSecret
}

func tokenFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Token ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Token "))
	}
	return auth
}

// resolveTokenUserID moved to db.go (wraps resolveTokenUserIDWithIP for internal callers).

func isSuperAdminUser(userID string) (bool, error) {
	var marker int
	err := db.QueryRow(`SELECT 1 FROM auth_super_admin WHERE user_id = ?`, userID).Scan(&marker)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func ensureSuperAdminRow(userID string) error {
	_, err := db.Exec(`INSERT IGNORE INTO auth_super_admin (user_id) VALUES (?)`, userID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE auth_user SET is_superuser = 1, is_staff = 1 WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	// v63 RBAC 双写：确保 /api/auth/user-roles/ 与 /me/ 平台角色一致
	ensurePlatformRoleRows(userID, "role-super-admin")
	return nil
}

func userExists(userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM auth_user WHERE id = ?)`, userID).Scan(&exists)
	return exists, err
}

func buildUserDetailJSON(userID string) (map[string]interface{}, error) {
	var isActive, isArchived bool
	var isTenant, isTester bool
	err := db.QueryRow(`SELECT is_active, COALESCE(is_archived, 0), COALESCE(is_tenant, 0), COALESCE(is_tester, 0) FROM auth_user WHERE id = ?`, userID).Scan(&isActive, &isArchived, &isTenant, &isTester)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	isSuper, err := isSuperAdminUser(userID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT method_type, identifier, is_verified, COALESCE(phone_country_calling_code, '') FROM auth_login_method
		WHERE object_id = ? AND binding_voided_at IS NULL
		ORDER BY method_type`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loginMethods := []interface{}{}
	email := ""
	for rows.Next() {
		var methodType, identifier, countryCallingCode string
		var verified bool
		if err := rows.Scan(&methodType, &identifier, &verified, &countryCallingCode); err != nil {
			return nil, err
		}
		lm := map[string]interface{}{
			"method_type": methodType,
			"identifier":  identifier,
			"is_verified": verified,
		}
		if countryCallingCode != "" {
			lm["country_calling_code"] = countryCallingCode
		}
		loginMethods = append(loginMethods, lm)
		if methodType == "email" && email == "" {
			email = identifier
		}
	}

	// Fetch profile fields: username + avatar_url (frontend Navbar depends on these)
	var username, avatarURL string
	_ = db.QueryRow(
		`SELECT COALESCE(username, ''), COALESCE(avatar, '') FROM auth_user_profile WHERE user_id = ?`,
		userID,
	).Scan(&username, &avatarURL)
	// Not an error if profile row doesn't exist yet — defaults to empty strings.

	// Fetch company memberships for companies/current_company/current_workspace fields
	// (frontend Navbar and router guard depend on these for tenant routing)
	companyNicknames := fetchCompanyNicknames(userID)
	companies := buildCompaniesFromNicknames(companyNicknames)
	var currentCompany map[string]interface{}
	if len(companies) > 0 {
		currentCompany = companies[0]
	}

	// Derive current_workspace from first member's workspace_id (same pattern as current_company)
	var currentWorkspace map[string]interface{}
	for _, cn := range companyNicknames {
		if m, ok := cn.(map[string]interface{}); ok {
			if wsID, ok := m["workspace_id"]; ok && wsID != nil && wsID != "" {
				currentWorkspace = map[string]interface{}{"id": wsID}
				break
			}
		}
	}

	// v63 RBAC: 平台角色（RBAC 表优先，遗留标志回退），供前端 Navbar 判定
	// 「系统管理」入口（isPlatformStaff）。读取失败时返回空列表而非报错，
	// 避免平台角色数据异常影响用户信息主流程。
	platformRoles, _ := loadPlatformRoles(userID)
	if platformRoles == nil {
		platformRoles = []string{}
	}

	return map[string]interface{}{
		"id":                     userID,
		"is_active":              isActive,
		"is_archived":            isArchived,
		"is_tenant":              isTenant,
		"is_tester":              isTester,
		"is_superuser":           isSuper,
		"platform_roles":         platformRoles,
		"email":                  email,
		"username":               username,
		"avatar_url":             avatarURL,
		"login_methods":          loginMethods,
		"companies":              companies,
		"current_company":        currentCompany,
		"current_workspace":      currentWorkspace,
		"pending_privacy_policy": fetchPendingPrivacyPolicy(userID),
	}, nil
}

// fetchPendingPrivacyPolicy returns the latest active privacy policy that the user
// hasn't consented to yet. Returns nil if all active policies are consented, or if
// the task_bill legal tables are not yet available.
// Queries task_bill database via cross-database SQL (same MySQL instance).
func fetchPendingPrivacyPolicy(userID string) interface{} {
	// Query latest active privacy policy from task_bill
	var policyID, title, content, version, createdAt, updatedAt string
	var isMaterialChange int
	err := db.QueryRow(`
		SELECT id, title, content, version, is_material_change, created_at, updated_at
		FROM task_bill.billing_privacy_policies
		WHERE is_active = 1
		ORDER BY created_at DESC LIMIT 1
	`).Scan(&policyID, &title, &content, &version, &isMaterialChange, &createdAt, &updatedAt)
	if err != nil {
		return nil // No active policy, or task_bill tables not yet created
	}

	// Check if user has already consented to this policy version
	var consentExists int
	err = db.QueryRow(`
		SELECT 1 FROM task_bill.billing_user_privacy_policy_consents
		WHERE user_id = ? AND privacy_policy_id = ?
		LIMIT 1
	`, userID, policyID).Scan(&consentExists)
	if err == nil {
		return nil // Already consented
	}

	return map[string]interface{}{
		"id":                 policyID,
		"title":              title,
		"content":            content,
		"version":            version,
		"is_material_change": isMaterialChange == 1,
		"created_at":         createdAt,
		"updated_at":         updatedAt,
	}
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	// "me" 自指语义：前端 10+ 处调用 /api/accounts/users/me/（Navbar、WorkPanel、
	// PeopleManage 等）。自 Django 退役（OPT-049）后该路径落入 {user_id} 字面量匹配
	// 导致 404，此处按请求凭据解析当前用户。已由凭据解析出身份，跳过下方
	// token-vs-user_id 一致性校验（cookie 会话无 bearer token 时亦放行）。
	isMe := userID == "me"
	if isMe {
		resolved, err := resolveTokenUserIDFromRequest(r)
		if err != nil || resolved == "" {
			// 与 profile OPT-006 对齐：Navbar 优先打 /me/，须在写 401 体前清残留 cookie
			clearResidualAuthCookies(w)
			writeErrorDetail(w, r, http.StatusUnauthorized, "请先登录")
			return
		}
		userID = resolved
	} else {
		tokenUserID, err := resolveTokenUserIDWithIP(tokenFromRequest(r), resolveClientIP(r))
		if err != nil {
			// No bearer token — allow internal services with valid secret
			if !requireInternalSecret(r) {
				writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
				return
			}
			// Internal service with valid secret — proceed directly
		} else if tokenUserID != userID && !requireInternalSecret(r) {
			writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
			return
		}
	}
	payload, err := buildUserDetailJSON(userID)
	if err == sql.ErrNoRows {
		// 清库后偶发「凭据解析出已删除用户」：/me/ 404 亦清残留会话 cookie
		if isMe {
			clearResidualAuthCookies(w)
		}
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// handleGetUserLegacyPath handles the legacy Django path /api/user/{user_id}/accounts/users/me/
// that was previously served by the now-decommissioned saas-backend (OPT-049, 2026-07-30).
// Frontend still calls this path from 13+ locations; this handler bridges the gap until
// the frontend can be migrated to the canonical /api/accounts/users/{user_id}/ path.
func handleGetUserLegacyPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	// Auth: accept bearer token (user must match requested user_id) or internal secret.
	tokenUserID, err := resolveTokenUserIDWithIP(tokenFromRequest(r), resolveClientIP(r))
	if err != nil {
		// No bearer token — check gateway forward-auth X-User-Id header.
		gwUserID := strings.TrimSpace(r.Header.Get("X-User-Id"))
		if gwUserID == "" && !requireInternalSecret(r) {
			// 遗留 /api/user/{id}/accounts/users/me/ 浏览器探测路径：与 canonical /me/ 对齐清 cookie
			clearResidualAuthCookies(w)
			writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
			return
		}
		if gwUserID != "" && gwUserID != userID && !requireInternalSecret(r) {
			writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
			return
		}
	} else if tokenUserID != userID && !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	payload, err := buildUserDetailJSON(userID)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleBatchResolveUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rawIDs, ok := body["user_ids"]
	if !ok {
		writeError(w, r, http.StatusBadRequest, "user_ids required")
		return
	}
	idList, ok := rawIDs.([]interface{})
	if !ok {
		writeError(w, r, http.StatusBadRequest, "user_ids must be array")
		return
	}
	results := make([]map[string]interface{}, 0, len(idList))
	for _, item := range idList {
		var userID string
		switch v := item.(type) {
		case string:
			userID = strings.TrimSpace(v)
		default:
			userID = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if userID == "" {
			continue
		}
		exists, err := userExists(userID)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		entry := map[string]interface{}{
			"user_id": userID,
			"found":   exists,
		}
		results = append(results, entry)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func loadUserAuthFlags(userID string) (isActive, isSuperuser, isStaff, isArchived bool, err error) {
	err = db.QueryRow(`
		SELECT is_active, is_superuser, is_staff, COALESCE(is_archived, 0) FROM auth_user WHERE id = ?`, userID,
	).Scan(&isActive, &isSuperuser, &isStaff, &isArchived)
	if err != nil {
		return false, false, false, false, err
	}
	isSuper, err := isSuperAdminUser(userID)
	if err != nil {
		return false, false, false, false, err
	}
	if isSuper {
		isSuperuser = true
		isStaff = true
	}
	return isActive, isSuperuser, isStaff, isArchived, nil
}

func handleResolveToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	tokenKey := strField(body, "token")
	if tokenKey == "" {
		writeError(w, r, http.StatusBadRequest, "token required")
		return
	}
	userID, err := resolveTokenUserID(tokenKey)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "invalid token")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	isActive, isSuperuser, isStaff, isArchived, err := loadUserAuthFlags(userID)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !isActive || isArchived {
		writeErrorDetail(w, r, http.StatusForbidden, "user inactive")
		return
	}
	var isTenant bool
	db.QueryRow(`SELECT COALESCE(is_tenant, 0) FROM auth_user WHERE id = ?`, userID).Scan(&isTenant)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":     userID,
		"is_active":   isActive,
		"is_archived": isArchived,
		"is_tenant":   isTenant,

		"is_superuser": isSuperuser,
		"is_staff":     isStaff,
	})
}

func handleGetSuperAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	isSuper, err := isSuperAdminUser(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":        userID,
		"is_super_admin": isSuper,
	})
}
