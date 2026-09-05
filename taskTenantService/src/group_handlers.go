package main

import (
	"authz"
	"net/http"

	"snowflake"
)

// requireGroupManage — 创建/改名/删除分组的权限判定（OPT-20260811-077）。
// 与 FE PeopleGroups 页门禁一致：member:manage 任意组穿透 / group:manage 分组管理 / group-members:manage 组管理员。
func requireGroupManage(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	if authz.HasPerm(r, tenantID, authz.PermMemberManage) ||
		authz.HasPerm(r, tenantID, authz.PermGroupManage) ||
		authz.HasPerm(r, tenantID, authz.PermGroupMembersManage) {
		return true
	}
	writeError(w, r, http.StatusForbidden, "仅租户管理员、分组管理员或组管理员可管理分组")
	return false
}

func handleGroupsListOrCreate(w http.ResponseWriter, r *http.Request, tenantID string) {
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`
			SELECT g.id, g.name, COALESCE(g.description,''),
			       (SELECT COUNT(*) FROM tenant_company_group_member m WHERE m.group_id=g.id) AS cnt
			FROM tenant_company_group g WHERE g.company_id=? ORDER BY g.created_at DESC`, tenantID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "查询失败")
			return
		}
		defer rows.Close()
		list := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id, name, desc string
			var cnt int
			if err := rows.Scan(&id, &name, &desc, &cnt); err != nil {
				continue
			}
			list = append(list, map[string]interface{}{
				"id": id, "name": name, "description": desc, "memberCount": cnt,
			})
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		if !requireGroupManage(w, r, tenantID) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		name := strField(body, "name")
		if name == "" {
			writeError(w, r, http.StatusBadRequest, "分组名称不能为空")
			return
		}
		desc := strField(body, "description")
		id := snowflake.GenerateIDString()
		_, err = db.Exec(`
			INSERT INTO tenant_company_group (id, name, description, company_id, created_by_id, created_at, updated_at)
			VALUES (?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
			id, name, desc, tenantID, userID,
		)
		if err != nil {
			logError("create group: "+err.Error(), "")
			writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{"error": "创建分组失败", "detail": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"id": id, "name": name, "description": desc, "memberCount": 0,
		})
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleGroupDelete(w http.ResponseWriter, r *http.Request, tenantID, groupID string) {
	if r.Method != http.MethodDelete {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	if !requireGroupManage(w, r, tenantID) {
		return
	}
	var cid string
	err := db.QueryRow(`SELECT company_id FROM tenant_company_group WHERE id=?`, groupID).Scan(&cid)
	if err != nil || cid != tenantID {
		writeError(w, r, http.StatusNotFound, "分组不存在")
		return
	}
	_, _ = db.Exec(`DELETE FROM tenant_company_group_member WHERE group_id=?`, groupID)
	_, _ = db.Exec(`DELETE FROM tenant_company_group WHERE id=?`, groupID)
	w.WriteHeader(http.StatusNoContent)
}

// handleGroupUpdate — PUT/PATCH /api/tenant/{tid}/accounts/groups/{gid}/ 更新名称与描述
func handleGroupUpdate(w http.ResponseWriter, r *http.Request, tenantID, groupID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	if !requireGroupManage(w, r, tenantID) {
		return
	}
	if !groupInTenant(groupID, tenantID) {
		writeError(w, r, http.StatusNotFound, "分组不存在")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	name := strField(body, "name")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "分组名称不能为空")
		return
	}
	desc := strField(body, "description")
	var memberCount int
	err = db.QueryRow(`
		SELECT (SELECT COUNT(*) FROM tenant_company_group_member m WHERE m.group_id=g.id)
		FROM tenant_company_group g WHERE g.id=?`, groupID).Scan(&memberCount)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "分组不存在")
		return
	}
	_, err = db.Exec(`
		UPDATE tenant_company_group
		SET name=?, description=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND company_id=?`,
		name, desc, groupID, tenantID,
	)
	if err != nil {
		logError("update group: "+err.Error(), "")
		writeError(w, r, http.StatusInternalServerError, "更新分组失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": groupID, "name": name, "description": desc, "memberCount": memberCount,
	})
}

func handleGroupMembers(w http.ResponseWriter, r *http.Request, tenantID, groupID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	if !groupInTenant(groupID, tenantID) {
		writeError(w, r, http.StatusNotFound, "分组不存在")
		return
	}
	rows, err := db.Query(`
		SELECT id, user_id, created_at FROM tenant_company_group_member WHERE group_id=?`, groupID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "查询失败")
		return
	}
	defer rows.Close()
	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, uid, created string
		if err := rows.Scan(&id, &uid, &created); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{
			"id": id, "group": groupID, "user": uid,
			"username": displayNameFallback("", uid), "email": nil, "created_at": created,
		})
	}
	writeJSON(w, http.StatusOK, list)
}

// groupMemberManageAllowed — v63 组内成员管理判定:
// tenant_admin（member:manage，任意组穿透）或 group_admin（group-members:manage，限本组）。
func groupMemberManageAllowed(w http.ResponseWriter, r *http.Request, userID, tenantID, groupID string) bool {
	if authz.HasPerm(r, tenantID, authz.PermMemberManage) {
		return true
	}
	if !authz.HasPerm(r, tenantID, authz.PermGroupMembersManage) {
		writeErrorDetail(w, r, http.StatusForbidden, "权限不足，需要权限码: "+string(authz.PermMemberManage)+" 或 "+string(authz.PermGroupMembersManage))
		return false
	}
	// group_admin 作用域: 仅限其管理的小组
	var n int
	if err := db.QueryRow(`SELECT COUNT(1) FROM tenant_group_admin WHERE group_id=? AND user_id=?`, groupID, userID).Scan(&n); err != nil || n == 0 {
		writeErrorDetail(w, r, http.StatusForbidden, "仅本组管理员可管理组内成员")
		return false
	}
	return true
}

func handleGroupAddMember(w http.ResponseWriter, r *http.Request, tenantID, groupID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	if !groupMemberManageAllowed(w, r, userID, tenantID, groupID) {
		return
	}
	if !groupInTenant(groupID, tenantID) {
		writeError(w, r, http.StatusNotFound, "分组不存在")
		return
	}
	body, _ := readJSONBody(r)
	targetUID := strField(body, "user_id")
	if targetUID == "" {
		writeError(w, r, http.StatusBadRequest, "用户ID不能为空")
		return
	}
	m, _ := getMember(targetUID, tenantID)
	if m == nil {
		writeError(w, r, http.StatusBadRequest, "用户不存在")
		return
	}
	var existing string
	_ = db.QueryRow(`SELECT id FROM tenant_company_group_member WHERE group_id=? AND user_id=?`, groupID, targetUID).Scan(&existing)
	if existing != "" {
		writeError(w, r, http.StatusBadRequest, "用户已在分组中")
		return
	}
	id := snowflake.GenerateIDString()
	_, err := db.Exec(`INSERT INTO tenant_company_group_member (id, group_id, user_id, created_at) VALUES (?,?,?,CURRENT_TIMESTAMP)`,
		id, groupID, targetUID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "添加失败")
		return
	}
	// 事件失效: 入组后用户获得该组角色/组资源伪码 → incr rev 使 PDP 缓存失效
	incrMembershipRev(targetUID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id": id, "group": groupID, "user": targetUID,
		"username": displayNameFallback("", targetUID), "email": nil,
	})
}

func handleGroupRemoveMember(w http.ResponseWriter, r *http.Request, tenantID, groupID string) {
	if r.Method != http.MethodDelete {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	if !groupMemberManageAllowed(w, r, userID, tenantID, groupID) {
		return
	}
	if !groupInTenant(groupID, tenantID) {
		writeError(w, r, http.StatusNotFound, "分组不存在")
		return
	}
	body, _ := readJSONBody(r)
	targetUID := strField(body, "user_id")
	if targetUID == "" {
		writeError(w, r, http.StatusBadRequest, "用户ID不能为空")
		return
	}
	res, err := db.Exec(`DELETE FROM tenant_company_group_member WHERE group_id=? AND user_id=?`, groupID, targetUID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "移除失败")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, r, http.StatusBadRequest, "成员不在分组中")
		return
	}
	// 事件失效: 移出组 → 失去组角色/组资源伪码 → incr rev
	incrMembershipRev(targetUID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "成员已移除"})
}

func groupInTenant(groupID, tenantID string) bool {
	var cid string
	err := db.QueryRow(`SELECT company_id FROM tenant_company_group WHERE id=?`, groupID).Scan(&cid)
	return err == nil && cid == tenantID
}

func handleGroupsRoute(w http.ResponseWriter, r *http.Request, tenantID string, parts []string) {
	if len(parts) == 0 {
		handleGroupsListOrCreate(w, r, tenantID)
		return
	}
	groupID := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodDelete:
			handleGroupDelete(w, r, tenantID, groupID)
		case http.MethodPut, http.MethodPatch:
			handleGroupUpdate(w, r, tenantID, groupID)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	switch parts[1] {
	case "members":
		handleGroupMembers(w, r, tenantID, groupID)
	case "add_member":
		handleGroupAddMember(w, r, tenantID, groupID)
	case "remove_member":
		handleGroupRemoveMember(w, r, tenantID, groupID)
	default:
		writeError(w, r, http.StatusNotFound, "not found")
	}
}
