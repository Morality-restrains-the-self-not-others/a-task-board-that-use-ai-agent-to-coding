package main

import (
	"authz"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

// ═══════════════════════════════════════════════════════════════
// 租户自定义角色 CRUD + 平台角色分配 (v63 design §6 taskAuth 1-9)
// ═══════════════════════════════════════════════════════════════

type roleRow struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Level       string   `json:"level"`
	Priority    int      `json:"priority"`
	IsSystem    bool     `json:"is_system"`
	CompanyID   *string  `json:"company_id"`
	Permissions []string `json:"permissions"`
}

func scanRoleWithPerms(rows *sql.Rows) ([]roleRow, error) {
	out := make([]roleRow, 0)
	idx := map[string]int{}
	for rows.Next() {
		var r roleRow
		var isSystem int
		var companyID sql.NullString
		var perm sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &r.DisplayName, &r.Level, &r.Priority,
			&isSystem, &companyID, &perm); err != nil {
			return nil, err
		}
		r.IsSystem = isSystem == 1
		if companyID.Valid {
			r.CompanyID = &companyID.String
		}
		if i, ok := idx[r.ID]; ok {
			if perm.Valid {
				out[i].Permissions = append(out[i].Permissions, perm.String)
			}
			continue
		}
		if perm.Valid {
			r.Permissions = []string{perm.String}
		} else {
			r.Permissions = []string{}
		}
		idx[r.ID] = len(out)
		out = append(out, r)
	}
	return out, rows.Err()
}

const rolePermSelect = `
SELECT r.id, r.name, r.display_name, r.level, r.priority, r.is_system,
       COALESCE(r.company_id,''), COALESCE(p.codename,'')
FROM auth_role r
LEFT JOIN auth_role_permission rp ON rp.role_id = r.id
LEFT JOIN auth_permission p ON p.id = rp.permission_id`

// handleCreateRole — POST /api/auth/roles/ 创建租户自定义角色
// body: {"company_id": "...", "display_name": "...", "permissions": ["task:view", ...]}
func handleCreateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		CompanyID   string   `json:"company_id"`
		DisplayName string   `json:"display_name"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid body")
		return
	}
	if body.CompanyID == "" || strings.TrimSpace(body.DisplayName) == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "company_id and display_name required")
		return
	}
	if !authz.RequirePerm(w, r, authz.PermMemberManage, body.CompanyID) {
		return
	}
	// 仅允许 tenant 级权限码（不含组码 — 组码仅内置角色）
	for _, p := range body.Permissions {
		if !isTenantResourcePerm(p) {
			writeErrorDetail(w, r, http.StatusBadRequest, "illegal permission: "+p)
			return
		}
	}
	name := "custom_" + body.CompanyID + "_" + randSuffix()
	id := "role-custom-" + randSuffix()
	if _, err := db.Exec(`INSERT INTO auth_role (id, name, display_name, level, priority, is_system, company_id, description)
		VALUES (?, ?, ?, 'tenant', 50, 0, ?, ?)`, id, name, body.DisplayName, body.CompanyID, "租户自定义角色"); err != nil {
		if isDuplicateKey(err) {
			writeErrorDetail(w, r, http.StatusConflict, "display_name 已存在")
			return
		}
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if err := replaceRolePermissions(id, body.Permissions); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	// 发布 RoleChanged → 缓存失效
	publishRoleChanged(body.CompanyID)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name})
}

// handleUpdateRole — PUT /api/auth/roles/role_id/{rid}/ 更新权限码（内置角色 403）
func handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	rid := r.PathValue("rid")
	var body struct {
		DisplayName string   `json:"display_name"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid body")
		return
	}
	var companyID string
	var isSystem int
	if err := db.QueryRow(`SELECT COALESCE(company_id,''), is_system FROM auth_role WHERE id=?`, rid).
		Scan(&companyID, &isSystem); err != nil {
		writeErrorDetail(w, r, http.StatusNotFound, "角色不存在")
		return
	}
	if isSystem == 1 {
		writeErrorDetail(w, r, http.StatusForbidden, "系统内置角色不可修改")
		return
	}
	if !authz.RequirePerm(w, r, authz.PermMemberManage, companyID) {
		return
	}
	for _, p := range body.Permissions {
		if !isTenantResourcePerm(p) {
			writeErrorDetail(w, r, http.StatusBadRequest, "illegal permission: "+p)
			return
		}
	}
	if body.DisplayName != "" {
		if _, err := db.Exec(`UPDATE auth_role SET display_name=? WHERE id=?`, body.DisplayName, rid); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := replaceRolePermissions(rid, body.Permissions); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	publishRoleChanged(companyID)
	writeErrorDetail(w, r, http.StatusOK, "ok")
}

// handleDeleteRole — DELETE /api/auth/roles/role_id/{rid}/ 删除自定义角色
func handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	rid := r.PathValue("rid")
	var companyID string
	var isSystem int
	if err := db.QueryRow(`SELECT COALESCE(company_id,''), is_system FROM auth_role WHERE id=?`, rid).
		Scan(&companyID, &isSystem); err != nil {
		writeErrorDetail(w, r, http.StatusNotFound, "角色不存在")
		return
	}
	if isSystem == 1 {
		writeErrorDetail(w, r, http.StatusForbidden, "系统内置角色不可删除")
		return
	}
	if !authz.RequirePerm(w, r, authz.PermMemberManage, companyID) {
		return
	}
	var roleName string
	_ = db.QueryRow(`SELECT name FROM auth_role WHERE id=?`, rid).Scan(&roleName)
	// 平台侧引用检查（租户侧引用由 taskTenant 级联失效，PDP 查无角色自然无码）
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_user_role WHERE role_id=? AND company_id=?`, rid, companyID).Scan(&n); err == nil && n > 0 {
		writeErrorDetail(w, r, http.StatusUnprocessableEntity, "角色仍有成员引用，请先解绑")
		return
	}
	if _, err := db.Exec(`DELETE FROM auth_role_resource_group WHERE role_id=?`, rid); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := db.Exec(`DELETE FROM auth_role_permission WHERE role_id=?`, rid); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := db.Exec(`DELETE FROM auth_role WHERE id=?`, rid); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	publishRoleChanged(companyID)
	log.Printf("[taskAuth] event=rbac_role_deleted status=ok role_id=%s name=%s company_id=%s", rid, roleName, companyID)
	writeErrorDetail(w, r, http.StatusOK, "ok")
}

// handleListRoles — GET /api/auth/roles/company_id/{cid}/ 角色列表（内置 + 自定义）
func handleListRoles(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if !authz.RequireTenantMember(w, r, cid) {
		return
	}
	rows, err := db.Query(rolePermSelect+`
		WHERE (r.company_id IS NULL OR r.company_id = ?) AND r.level = 'tenant'
		ORDER BY r.is_system DESC, r.priority DESC`, cid)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	roles, err := scanRoleWithPerms(rows)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

// handleAssignPlatformRole — POST /api/auth/user-roles/user_id/{uid}/
// body: {"role_name": "employee"} 仅 platform 级角色，super_admin 专属
func handleAssignPlatformRole(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")
	if !authz.RequirePlatformPerm(w, r, authz.PermEmployeeManage) {
		return
	}
	var body struct {
		RoleName string `json:"role_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RoleName == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "role_name required")
		return
	}
	var roleID, level string
	if err := db.QueryRow(`SELECT id, level FROM auth_role WHERE name=? AND company_id IS NULL`, body.RoleName).
		Scan(&roleID, &level); err != nil || level != "platform" {
		writeErrorDetail(w, r, http.StatusBadRequest, "平台角色不存在: "+body.RoleName)
		return
	}
	if err := insertPlatformUserRole(uid, roleID, "system"); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	// v63 反向双写：RBAC 指派同步遗留标志（loadUserAuthFlags / isSuperAdminUser
	// 等标志消费方仍依赖），保证读侧单一来源 loadPlatformRoles 判定一致
	syncLegacyFlagsFromPlatformRoles(uid)
	publishPlatformRoleChanged(uid)
	writeErrorDetail(w, r, http.StatusOK, "ok")
}

// handleRevokePlatformRole — DELETE /api/auth/user-roles/user_id/{uid}/role_name/{name}/
func handleRevokePlatformRole(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")
	name := r.PathValue("name")
	if !authz.RequirePlatformPerm(w, r, authz.PermEmployeeManage) {
		return
	}
	if _, err := db.Exec(`DELETE aur FROM auth_user_role aur
		JOIN auth_role r ON r.id = aur.role_id
		WHERE aur.user_id=? AND r.name=? AND aur.company_id IS NULL`, uid, name); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	// v63 反向双写：撤销后按剩余 RBAC 行重算遗留标志
	syncLegacyFlagsFromPlatformRoles(uid)
	publishPlatformRoleChanged(uid)
	writeErrorDetail(w, r, http.StatusOK, "ok")
}

// handleRoleUsers — GET /api/auth/role-users/role_name/{name}/ 角色成员
func handleRoleUsers(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !authz.RequirePlatformPerm(w, r, authz.PermEmployeeManage) {
		return
	}
	rows, err := db.Query(`SELECT aur.user_id FROM auth_user_role aur
		JOIN auth_role r ON r.id = aur.role_id
		WHERE r.name=? AND aur.company_id IS NULL`, name)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var users []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			users = append(users, uid)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"role_name": name, "user_ids": users})
}

// handleMyUserRoles — GET /api/auth/user-roles/ 当前用户角色
// v63 RBAC: 平台角色走 RBAC 表 + 遗留标志回退（loadPlatformRoles），
// 避免 028 回填迁移后经遗留路径（邀请/系统管理 PATCH）提升的用户角色为空，
// 导致前端菜单错误显示「开始使用」而非「系统管理」。
func handleMyUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil || userID == "" {
		writeErrorDetail(w, r, http.StatusUnauthorized, "请先登录")
		return
	}
	roles, rawCount, err := loadMyUserRoleItems(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	hasPlatformRole := false
	for _, item := range roles {
		level, _ := item["level"].(string)
		cid, _ := item["company_id"].(string)
		if level == "platform" && cid == "" {
			hasPlatformRole = true
			break
		}
	}
	// 过渡期兜底：无任何平台角色行时回退遗留 is_superuser/is_staff 标志
	if !hasPlatformRole {
		if fallback, err := loadPlatformRoles(userID); err == nil {
			for _, name := range fallback {
				roles = append(roles, map[string]any{"role": name, "level": "platform", "company_id": ""})
			}
			roles = uniqueUserRoleItems(roles)
		} else {
			slog.ErrorContext(r.Context(), "my_user_roles_fallback",
				"trace_id", tracelog.TraceIDFromContext(r.Context()),
				"user_id", userID, "error", err.Error())
		}
	}
	if rawCount > len(roles) {
		slog.WarnContext(r.Context(), "my_user_roles_duplicates_collapsed",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"user_id", userID, "raw_role_rows", rawCount, "role_count", len(roles))
	} else {
		slog.InfoContext(r.Context(), "my_user_roles",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"user_id", userID, "role_count", len(roles))
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

// handleMyPermissions — GET /api/auth/user-permissions/ 当前用户权限码（前端判定用）
func handleMyPermissions(w http.ResponseWriter, r *http.Request) {
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil || userID == "" {
		writeErrorDetail(w, r, http.StatusUnauthorized, "请先登录")
		return
	}
	perms, err := computePermSets(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadGateway, "授权状态不可用")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "tenant_perms": perms})
}

// isTenantResourcePerm 校验权限码 ∈ 14 个 tenant 资源码（组码禁止自定义角色使用）
func isTenantResourcePerm(code string) bool {
	for _, p := range tenantResourcePerms() {
		if string(p) == code {
			return true
		}
	}
	return false
}

func tenantResourcePerms() []string {
	return []string{
		"company:manage", "company:view",
		"member:manage", "member:view",
		"group:manage",
		"project:manage", "project:view",
		"task:manage", "task:view",
		"cloud:manage", "cloud:view",
		"billing:manage", "billing:view",
		"workspace:manage",
	}
}

// replaceRolePermissions 全量替换角色→权限关联
func replaceRolePermissions(roleID string, perms []string) error {
	if _, err := db.Exec(`DELETE FROM auth_role_permission WHERE role_id=?`, roleID); err != nil {
		return err
	}
	for _, code := range perms {
		var pid string
		if err := db.QueryRow(`SELECT id FROM auth_permission WHERE codename=?`, code).Scan(&pid); err != nil {
			continue // 未知码跳过（前置校验已拦截，此处防御）
		}
		if _, err := db.Exec(`INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
			VALUES (?, ?, ?)`, "rp-"+randSuffix(), roleID, pid); err != nil {
			return err
		}
	}
	return nil
}

// handleRoleExists — GET /api/internal/authz/role-exists?company_id=&role_name=
// 供 taskTenantService 校验角色名合法性（内置租户角色或该租户自定义角色）
func handleRoleExists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	cid := r.URL.Query().Get("company_id")
	name := r.URL.Query().Get("role_name")
	if cid == "" || name == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "company_id and role_name required")
		return
	}
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM auth_role WHERE name=? AND (company_id IS NULL OR company_id=?) AND level='tenant'`,
		name, cid).Scan(&n)
	exists := err == nil && n > 0
	writeJSON(w, http.StatusOK, map[string]any{"exists": exists})
}
