package main

import (
	"authz"
	"net/http"

	"snowflake"
)

// ═══════════════════════════════════════════════════════════════
// 小组管理员指派 API (v63 design §6 taskTenant 15-17)
// ═══════════════════════════════════════════════════════════════

// handleSetGroupAdmin — PUT /api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/
// 指派/更换小组管理员（tenant_admin）
func handleSetGroupAdmin(w http.ResponseWriter, r *http.Request, cid, gid, uid string) {
	if !authz.RequirePerm(w, r, authz.PermMemberManage, cid) {
		return
	}
	if err := ensureGroupInCompany(gid, cid); err != nil {
		writeError(w, r, http.StatusNotFound, "小组不存在")
		return
	}
	// 目标用户须是该租户成员
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tenant_company_member WHERE user_id=? AND company_id=? AND is_active=1`, uid, cid).Scan(&n); err != nil || n == 0 {
		writeError(w, r, http.StatusBadRequest, "目标用户不是该公司有效成员")
		return
	}
	if _, err := db.Exec(`INSERT INTO tenant_group_admin (id, group_id, user_id, company_id, assigned_by, assigned_at)
		VALUES (?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE assigned_by = VALUES(assigned_by)`,
		snowflake.GenerateIDString(), gid, uid, cid, getAuthUser(r)); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	// 事件失效: 组管理员指派 → 该用户 PDP 缓存失效（隐含 group_admin 角色）
	incrMembershipRev(uid)
	publishTenantRoleChanged(cid)
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok", "group_id": gid, "user_id": uid})
}

// handleDeleteGroupAdmin — DELETE /api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/
func handleDeleteGroupAdmin(w http.ResponseWriter, r *http.Request, cid, gid, uid string) {
	if !authz.RequirePerm(w, r, authz.PermMemberManage, cid) {
		return
	}
	if _, err := db.Exec(`DELETE FROM tenant_group_admin WHERE group_id=? AND user_id=? AND company_id=?`, gid, uid, cid); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	// 事件失效: 撤销组管理员 → 该用户 PDP 缓存失效
	incrMembershipRev(uid)
	publishTenantRoleChanged(cid)
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok"})
}

// handleListGroupAdmins — GET /api/tenant/group-admin/company_id/{cid}/group_id/{gid}/
func handleListGroupAdmins(w http.ResponseWriter, r *http.Request, cid, gid string) {
	if !authz.RequireTenantMember(w, r, cid) {
		return
	}
	rows, err := db.Query(`SELECT ga.id, ga.user_id, m.member_name, ga.assigned_at
		FROM tenant_group_admin ga
		LEFT JOIN tenant_company_member m ON m.user_id = ga.user_id AND m.company_id = ga.company_id
		WHERE ga.group_id=? AND ga.company_id=?`, gid, cid)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, uid string
		var name, at string
		var nameNull interface{}
		if err := rows.Scan(&id, &uid, &nameNull, &at); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		name, _ = nameNull.(string)
		out = append(out, map[string]any{"id": id, "user_id": uid, "member_name": name, "assigned_at": at})
	}
	writeJSON(w, http.StatusOK, out)
}

func ensureGroupInCompany(gid, cid string) error {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM tenant_company_group WHERE id=? AND company_id=?`, gid, cid).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return errGroupNotFound
	}
	return nil
}

var errGroupNotFound = &groupNotFoundError{}

type groupNotFoundError struct{}

func (e *groupNotFoundError) Error() string { return "group not found" }

// handleGroupAdminRoute — 分发 /api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/
func handleGroupAdminRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	// 有 user_id: [group-admin, company_id, cid, group_id, gid, user_id, uid]
	if len(parts) == 7 && parts[1] == "company_id" && parts[3] == "group_id" && parts[5] == "user_id" {
		cid, gid, uid := parts[2], parts[4], parts[6]
		switch r.Method {
		case http.MethodPut:
			handleSetGroupAdmin(w, r, cid, gid, uid)
		case http.MethodDelete:
			handleDeleteGroupAdmin(w, r, cid, gid, uid)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	// 无 user_id: [group-admin, company_id, cid, group_id, gid]
	if len(parts) == 5 && parts[1] == "company_id" && parts[3] == "group_id" {
		cid, gid := parts[2], parts[4]
		if r.Method == http.MethodGet {
			handleListGroupAdmins(w, r, cid, gid)
			return
		}
	}
	writeError(w, r, 404, "not found")
}

// handleResourceGroupRoute — 分发 /api/tenant/resource-group/company_id/{cid}/...
func handleResourceGroupRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	// 列表: [resource-group, company_id, cid, group_id, gid]
	if len(parts) == 5 && parts[1] == "company_id" && parts[3] == "group_id" {
		cid, gid := parts[2], parts[4]
		if r.Method == http.MethodGet {
			handleListGroupResources(w, r, cid, gid)
			return
		}
	}
	// 资源归属: [resource-group, company_id, cid, resource_type, type, resource_id, rid]
	if len(parts) == 7 && parts[1] == "company_id" && parts[3] == "resource_type" && parts[5] == "resource_id" {
		cid, rtype, rid := parts[2], parts[4], parts[6]
		if r.Method == http.MethodGet {
			handleListResourceGroups(w, r, cid, rtype, rid)
			return
		}
	}
	// 分配/撤销: [resource-group, company_id, cid] + assignment_id
	if len(parts) == 3 && parts[1] == "company_id" {
		cid := parts[2]
		if r.Method == http.MethodPost {
			handleAssignResourceToGroup(w, r, cid)
			return
		}
	}
	if len(parts) == 5 && parts[1] == "company_id" && parts[3] == "assignment_id" {
		cid, aid := parts[2], parts[4]
		if r.Method == http.MethodDelete {
			handleRevokeResourceFromGroup(w, r, cid, aid)
			return
		}
	}
	writeError(w, r, 404, "not found")
}
