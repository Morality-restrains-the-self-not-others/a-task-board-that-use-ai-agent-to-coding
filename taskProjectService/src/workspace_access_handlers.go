package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

func handleProjectsWorkspaceAccess(w http.ResponseWriter, r *http.Request, tenantID string, segs []string) {
	if len(segs) == 0 {
		writeError(w, r, 404, "not found")
		return
	}
	action := segs[0]
	switch action {
	case "workspace-permissions":
		handleWorkspacePermissions(w, r, tenantID)
	case "workspace-collaborators":
		handleWorkspaceCollaborators(w, r, tenantID)
	case "set-permission":
		handleSetWorkspacePermission(w, r, tenantID)
	case "remove-permission":
		handleRemoveWorkspacePermission(w, r, tenantID)
	default:
		writeError(w, r, 404, "not found")
	}
}

// checkWorkspaceAccess returns true when userID is permitted to access the workspace.
// Semantics: if project_workspace_accesses has rows → userID must match a row (direct user_id
// or via group_id membership); if no rows → workspace is open to all tenant members.
// This mirrors the SQL filter in handleListWorkspaces and hasWorkspaceAccess in taskTaskService.
// OPT-20260726-034: Added group_id membership check for parity with taskTaskService.
func checkWorkspaceAccess(workspaceID, userID string) bool {
	if userID == "internal" {
		return true
	}
	rows, err := loadWorkspaceAccessRows(workspaceID)
	if err != nil {
		return false
	}
	if len(rows) == 0 {
		return true // open to all tenant members
	}
	for _, row := range rows {
		if uid, _ := row["user_id"].(string); uid == userID {
			return true
		}
		// Check group_id membership — user may have access through a group
		if gid, _ := row["group_id"].(string); gid != "" && gid != "0" && gid != "<nil>" {
			memberIDs, err := tenantGroupMemberUserIDs(gid)
			if err != nil {
				continue // fail-open for this group entry
			}
			for _, mid := range memberIDs {
				if mid == userID {
					return true
				}
			}
		}
	}
	return false
}

func loadWorkspaceAccessRows(workspaceID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		"SELECT id,workspace_id,user_id,group_id,permission,created_at FROM project_workspace_accesses WHERE workspace_id=?",
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []map[string]interface{}{}
	for rows.Next() {
		var id, wsID, userID, groupID, perm string
		var ca interface{}
		if err := rows.Scan(&id, &wsID, &userID, &groupID, &perm, &ca); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{
			"id": id, "workspace_id": wsID, "user_id": userID, "group_id": groupID,
			"permission": perm, "role": perm, "created_at": ca,
		})
	}
	return list, nil
}

func verifyWorkspaceInTenant(workspaceID, tenantID string) error {
	var companyID string
	err := db.QueryRow("SELECT company_id FROM project_workspace_entries WHERE id=?", workspaceID).Scan(&companyID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("workspace not found")
	}
	if companyID != tenantID {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func handleWorkspacePermissions(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	wsID := r.URL.Query().Get("workspace_id")
	if wsID == "" {
		writeError(w, r, 400, "工作空间ID不能为空")
		return
	}
	if err := verifyWorkspaceInTenant(wsID, tenantID); err != nil {
		writeError(w, r, 404, "工作空间不存在")
		return
	}
	rows, err := loadWorkspaceAccessRows(wsID)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	out, err := enrichWorkspaceAccessRows(tenantID, rows)
	if err != nil {
		writeError(w, r, 502, err.Error())
		return
	}
	writeJSON(w, 200, out)
}

func handleWorkspaceCollaborators(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	wsID := r.URL.Query().Get("workspace_id")
	if wsID == "" {
		writeError(w, r, 400, "工作空间ID不能为空")
		return
	}
	if err := verifyWorkspaceInTenant(wsID, tenantID); err != nil {
		if err.Error() == "forbidden" {
			writeError(w, r, 403, "无权访问该工作空间")
			return
		}
		writeError(w, r, 404, "工作空间不存在")
		return
	}
	rows, err := loadWorkspaceAccessRows(wsID)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	userID := getAuthUser(r)
	out, status, err := buildWorkspaceCollaborators(tenantID, userID, rows)
	if err != nil {
		if status == 0 {
			status = 502
		}
		writeError(w, r, status, err.Error())
		return
	}
	writeJSON(w, status, out)
}

func handleSetWorkspacePermission(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	wsID := strField(body, "workspace_id")
	role := strField(body, "role")
	if role == "" {
		role = strField(body, "permission")
	}
	memberID := strField(body, "company_member_id")
	groupID := strField(body, "group_id")
	directUserID := strField(body, "user_id")
	if wsID == "" || role == "" {
		writeError(w, r, 400, "工作空间ID和角色不能为空")
		return
	}
	if memberID == "" && groupID == "" && directUserID == "" {
		writeError(w, r, 400, "公司成员ID和分组ID中至少需要指定一个")
		return
	}
	if err := verifyWorkspaceInTenant(wsID, tenantID); err != nil {
		writeError(w, r, 404, "工作空间不存在")
		return
	}
	userID := ""
	if memberID != "" {
		m, err := tenantGetMemberByID(memberID, tenantID)
		if err != nil || m == nil {
			writeError(w, r, 400, "公司成员不存在")
			return
		}
		userID = strField(m, "user_id")
	} else if directUserID != "" {
		// internal 服务直传（taskEvents workspace 创建链）— 跳过成员解析
		userID = directUserID
	}
	var existingID string
	if userID != "" {
		db.QueryRow("SELECT id FROM project_workspace_accesses WHERE workspace_id=? AND user_id=?", wsID, userID).Scan(&existingID)
	} else {
		db.QueryRow("SELECT id FROM project_workspace_accesses WHERE workspace_id=? AND group_id=?", wsID, groupID).Scan(&existingID)
	}
	if existingID != "" {
		db.Exec("UPDATE project_workspace_accesses SET permission=? WHERE id=?", role, existingID)
	} else {
		existingID = genID("wa")
		db.Exec(`INSERT INTO project_workspace_accesses(id,workspace_id,user_id,group_id,permission) VALUES(?,?,?,?,?)`,
			existingID, wsID, userID, groupID, role)
	}
	rows, _ := loadWorkspaceAccessRows(wsID)
	for _, row := range rows {
		if fmt.Sprintf("%v", row["id"]) == existingID {
			writeJSON(w, 200, row)
			return
		}
	}
	writeJSON(w, 200, map[string]interface{}{"id": existingID, "workspace_id": wsID, "role": role})
}

func handleRemoveWorkspacePermission(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodDelete {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	wsID := strField(body, "workspace_id")
	memberID := strField(body, "company_member_id")
	groupID := strField(body, "group_id")
	if wsID == "" {
		writeError(w, r, 400, "工作空间ID不能为空")
		return
	}
	if memberID == "" && groupID == "" {
		writeError(w, r, 400, "公司成员ID和分组ID中至少需要指定一个")
		return
	}
	if err := verifyWorkspaceInTenant(wsID, tenantID); err != nil {
		writeError(w, r, 404, "工作空间不存在")
		return
	}
	userID := ""
	if memberID != "" {
		m, err := tenantGetMemberByID(memberID, tenantID)
		if err != nil || m == nil {
			writeError(w, r, 400, "公司成员不存在")
			return
		}
		userID = strField(m, "user_id")
	}
	if userID != "" {
		res, _ := db.Exec("DELETE FROM project_workspace_accesses WHERE workspace_id=? AND user_id=?", wsID, userID)
		if n, _ := res.RowsAffected(); n == 0 {
			writeError(w, r, 400, "权限记录不存在")
			return
		}
	} else if groupID != "" {
		res, _ := db.Exec("DELETE FROM project_workspace_accesses WHERE workspace_id=? AND group_id=?", wsID, groupID)
		if n, _ := res.RowsAffected(); n == 0 {
			writeError(w, r, 400, "权限记录不存在")
			return
		}
	}
	writeJSON(w, 200, map[string]string{"message": "权限已移除"})
}
