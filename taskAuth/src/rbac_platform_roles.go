package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

// ═══════════════════════════════════════════════════════════════
// taskAuth 平台角色解析 (v63 RBAC 收尾)
//
// 单一读路径: loadPlatformRoles = RBAC 表 (auth_user_role) 优先；
// 无平台角色行时回退遗留标志位 (is_superuser/is_staff)。
// 双写策略: 所有平台角色写路径（RBAC 指派/撤销 + 遗留提升路径）同步
// 更新两张源，标志位成为 RBAC 表的纯镜像，读侧仅依赖 loadPlatformRoles。
//
// 用途:
//   - GET /api/accounts/users/me/ → platform_roles 字段
//   - GET /api/auth/user-roles/   → 平台角色兜底（防止遗漏回填用户菜单错误）
//   - GET /api/internal/users/id/{uid}/platform-roles/ → taskEvents 等内部服务
//   - 网关 forward-auth → X-User-Roles 注入（RBAC 权威，标志仅兜底）
// ═══════════════════════════════════════════════════════════════

// loadPlatformRoles 返回用户的平台级角色名列表（company_id IS NULL）。
// RBAC 表优先（v63 权威）；无任何平台角色行时回退遗留 is_superuser/is_staff
// 标志（superuser→super_admin，staff→employee，与 platformRolesFromFlags 一致），
// 覆盖 028 回填迁移之后才被提升（且未走 RBAC 指派）的用户。
func loadPlatformRoles(userID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT r.name FROM auth_user_role aur
		JOIN auth_role r ON r.id = aur.role_id
		WHERE aur.user_id = ? AND r.level = 'platform' AND aur.company_id IS NULL
		GROUP BY r.name
		ORDER BY MAX(r.priority) DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(roles) > 0 {
		return roles, nil
	}

	// 过渡期回退：遗留标志位推导
	_, isSuperuser, isStaff, _, err := loadUserAuthFlags(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []string{}, nil
		}
		return nil, err
	}
	return platformRolesFromFlags(isSuperuser, isStaff), nil
}

// hasPlatformRole 判断用户是否持有指定平台角色（RBAC 表 + 遗留标志回退）。
func hasPlatformRole(userID, roleName string) (bool, error) {
	roles, err := loadPlatformRoles(userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == roleName {
			return true, nil
		}
	}
	return false, nil
}

// handleInternalUserPlatformRoles — GET /api/internal/users/id/{user_id}/platform-roles/
// 内部端点：供 taskEvents 等服务判定用户是否平台角色（决定是否自动建公司等）。
// 仅接受内部密钥，不暴露用户角色全量信息。
func handleInternalUserPlatformRoles(w http.ResponseWriter, r *http.Request) {
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
	roles, err := loadPlatformRoles(userID)
	if err != nil {
		log.Printf("[taskAuth] event=platform_roles status=error user_id=%s err=%v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if roles == nil {
		roles = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"user_id": userID, "roles": roles})
}

// platformRoleNamesFromTable 仅查 RBAC 表（不含标志回退）——供标志重算使用，
// 避免 loadPlatformRoles 的标志回退造成"用标志算标志"的循环。
func platformRoleNamesFromTable(userID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT r.name FROM auth_user_role aur
		JOIN auth_role r ON r.id = aur.role_id
		WHERE aur.user_id = ? AND r.level = 'platform' AND aur.company_id IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// syncLegacyFlagsFromPlatformRoles 由 RBAC 平台角色行重算遗留标志（反向双写）。
//
// 解决"反向双写缺口"：此前 RBAC 指派/撤销只写 auth_user_role，不更新
// is_superuser/is_staff 与 auth_super_admin，导致 RBAC 指派但无标志的用户
// 在网关 forward-auth（旧按标志推导 X-User-Roles）缺失平台权限。
//
// 规则（RBAC 权威，标志为镜像）:
//   - super_admin 存在 → is_superuser=1, is_staff=1 + auth_super_admin 行
//   - 仅 employee 存在 → is_superuser=0, is_staff=1，删除 auth_super_admin
//   - 无平台角色      → 两标志清零，删除 auth_super_admin
func syncLegacyFlagsFromPlatformRoles(userID string) {
	names, err := platformRoleNamesFromTable(userID)
	if err != nil {
		log.Printf("[taskAuth] event=rbac_sync_flags status=error user_id=%s err=%v", userID, err)
		return
	}
	hasSuper, hasEmp := false, false
	for _, n := range names {
		switch n {
		case "super_admin":
			hasSuper = true
		case "employee":
			hasEmp = true
		}
	}
	switch {
	case hasSuper:
		_, _ = db.Exec(`INSERT IGNORE INTO auth_super_admin (user_id) VALUES (?)`, userID)
		_, _ = db.Exec(`UPDATE auth_user SET is_superuser = 1, is_staff = 1 WHERE id = ?`, userID)
	case hasEmp:
		_, _ = db.Exec(`DELETE FROM auth_super_admin WHERE user_id = ?`, userID)
		_, _ = db.Exec(`UPDATE auth_user SET is_superuser = 0, is_staff = 1 WHERE id = ?`, userID)
	default:
		_, _ = db.Exec(`DELETE FROM auth_super_admin WHERE user_id = ?`, userID)
		_, _ = db.Exec(`UPDATE auth_user SET is_superuser = 0, is_staff = 0 WHERE id = ?`, userID)
	}
}

// uniqueUserRoleItems 按 (role, level, company_id) 折叠重复赋值行。
// MySQL UNIQUE(user_id, role_id, company_id) 对 NULL company_id 不防重，读路径必须去重。
func uniqueUserRoleItems(items []map[string]any) []map[string]any {
	seen := make(map[string]struct{}, len(items))
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		role, _ := it["role"].(string)
		level, _ := it["level"].(string)
		cid, _ := it["company_id"].(string)
		key := role + "\x1f" + level + "\x1f" + cid
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, it)
	}
	return out
}

// loadMyUserRoleItems 返回当前用户去重后的角色列表，以及折叠前的原始行数。
func loadMyUserRoleItems(userID string) ([]map[string]any, int, error) {
	rows, err := db.Query(`SELECT r.name, r.level, COALESCE(aur.company_id,'') FROM auth_user_role aur
		JOIN auth_role r ON r.id = aur.role_id WHERE aur.user_id=?`, userID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var raw []map[string]any
	for rows.Next() {
		var name, level, cid string
		if err := rows.Scan(&name, &level, &cid); err != nil {
			return nil, 0, err
		}
		raw = append(raw, map[string]any{"role": name, "level": level, "company_id": cid})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return uniqueUserRoleItems(raw), len(raw), nil
}

// insertPlatformUserRole 幂等写入平台角色行（company_id IS NULL）。
// 先查再插：uk_urc 对 NULL 无效，INSERT IGNORE 只挡主键冲突。
func insertPlatformUserRole(userID, roleID, assignedBy string) error {
	if userID == "" || roleID == "" {
		return nil
	}
	if assignedBy == "" {
		assignedBy = "system"
	}
	var existing string
	err := db.QueryRow(`SELECT id FROM auth_user_role
		WHERE user_id=? AND role_id=? AND company_id IS NULL LIMIT 1`, userID, roleID).Scan(&existing)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = db.Exec(`
		INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
		VALUES (?, ?, ?, NULL, ?, NOW())`,
		"aur-"+randSuffix(), userID, roleID, assignedBy)
	return err
}

// ensurePlatformRoleRows 幂等写入平台角色行（v63 RBAC 与遗留标志双写）。
// 供遗留提升路径调用：ensureSuperAdminRow / applyInviteRoleAndExpiry /
// system-admin 用户管理 PATCH，保证后续 /api/auth/user-roles/ 与 /me/ 一致。
func ensurePlatformRoleRows(userID string, roleIDs ...string) {
	for _, roleID := range roleIDs {
		if roleID == "" {
			continue
		}
		if err := insertPlatformUserRole(userID, roleID, "system"); err != nil {
			log.Printf("[taskAuth] event=rbac_ensure_role status=error user_id=%s role_id=%s err=%v", userID, roleID, err)
		}
	}
}
