package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// --- Workspace Handlers ---

func handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	currentWS := r.URL.Query().Get("workspace_id")
	if currentWS == "" {
		currentWS = getMemberWorkspaceID(userID, tenantID)
	}
	// 对已认证的非内部用户，默认仅返回有权限的工作空间（与 taskTaskService
	// hasWorkspaceAccess 语义对齐：有 access 行则须命中 user_id；无 access 行
	// 则视为对租户开放）。服务间调用可通过 Header 设置 X-Auth-User-Id: internal
	// 获取全量列表。
	applyMine := !isInternalCall(r)
	// 显式传 mine=0 可用于调试场景绕过裁剪（需配合 internal 身份）
	if r.URL.Query().Get("mine") == "0" && isInternalCall(r) {
		applyMine = false
	}

	const wsSelectCols = `id,name,description,company_id,deliverable_system_id,deliverable_system_from,is_default,task_archive_tier,allow_personal_feature_params,container_image_at_mode_enabled,created_at,updated_at`
	query := `SELECT ` + wsSelectCols + ` FROM project_workspace_entries w WHERE w.company_id=?`
	args := []interface{}{tenantID}
	if applyMine {
		// OPT-20260726-034: Expand filter to include group_id membership.
		// SQL alone cannot resolve group membership (requires API call), so we use a
		// two-phase approach: inner EXISTS checks direct user_id access, then
		// Go-level post-filter via checkWorkspaceAccess adds group_id resolution.
		// Workspaces with zero access rows remain open to all tenant members.
		query += ` AND (
			EXISTS (SELECT 1 FROM project_workspace_accesses a WHERE a.workspace_id=w.id AND a.user_id=?)
			OR EXISTS (SELECT 1 FROM project_workspace_accesses a WHERE a.workspace_id=w.id AND a.group_id IS NOT NULL AND a.group_id != '' AND a.group_id != '0')
			OR NOT EXISTS (SELECT 1 FROM project_workspace_accesses a WHERE a.workspace_id=w.id)
		)`
		args = append(args, userID)
	}
	if idsParam := r.URL.Query().Get("ids"); idsParam != "" {
		parts := strings.Split(idsParam, ",")
		placeholders := make([]string, 0, len(parts))
		for _, p := range parts {
			id := strings.TrimSpace(p)
			if id != "" {
				placeholders = append(placeholders, "?")
				args = append(args, id)
			}
		}
		if len(placeholders) > 0 {
			query += ` AND w.id IN (` + strings.Join(placeholders, ",") + `)`
		}
	}
	query += " ORDER BY w.name"
	rows, err := db.Query(query, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()
	workspaces := []map[string]interface{}{}
	for rows.Next() {
		var id, name, desc, cid, dsid, dsfrom, tier string
		var isDef, allowPP, containerImageAtMode bool
		var ca, ua time.Time
		if err := rows.Scan(&id, &name, &desc, &cid, &dsid, &dsfrom, &isDef, &tier, &allowPP, &containerImageAtMode, &ca, &ua); err != nil {
			continue
		}
		// Post-filter: for workspaces that have group_id-based access rows, verify
		// the user actually belongs to one of those groups via checkWorkspaceAccess.
		if applyMine {
			if !checkWorkspaceAccess(id, userID) {
				continue
			}
		}
		workspaces = append(workspaces, map[string]interface{}{
			"id": id, "name": name, "description": desc,
			// Company display name is owned by accounts; do not echo company_id here.
			"company_id": cid, "company_name": "",
			"deliverable_system_id": dsid, "deliverable_system_from": dsfrom,
			"is_default": isDef, "is_current": currentWS != "" && id == currentWS,
			"task_archive_tier": tier, "allow_personal_feature_params": allowPP,
			"container_image_at_mode_enabled": containerImageAtMode,
			"created_at":                      ca.UTC().Format(time.RFC3339Nano),
			"updated_at":                      ua.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, 200, workspaces)
}

func handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	body, _ := readJSONBody(r)
	id := strField(body, "id")
	if id == "" {
		id = genID("ws")
	}
	name := strField(body, "name")
	if name == "" {
		writeError(w, r, 400, "name is required")
		return
	}
	isDefault := 0
	if _, ok := body["is_default"]; ok && boolFieldDefault(body, "is_default", false) {
		isDefault = 1
	}
	tier := strField(body, "task_archive_tier")
	if tier == "" {
		tier = "7d"
	}
	containerImageAtMode := 1 // 默认开启容器镜像 @ 模式
	if v, ok := body["container_image_at_mode_enabled"]; ok {
		if b, ok := v.(bool); ok && !b {
			containerImageAtMode = 0
		}
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()
	if isDefault == 1 {
		if err := unsetTenantWorkspaceDefaultsTx(tx, tenantID); err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
	}
	_, err = tx.Exec(
		`INSERT INTO project_workspace_entries(id,name,description,company_id,deliverable_system_id,deliverable_system_from,is_default,task_archive_tier,container_image_at_mode_enabled) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, name, strField(body, "description"), tenantID,
		strField(body, "deliverable_system_id"), strField(body, "deliverable_system_from"),
		isDefault, tier, containerImageAtMode,
	)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	logInfo("workspace created: "+id, r.Header.Get("X-Trace-Id"))
	row := db.QueryRow(`SELECT id,name,COALESCE(description,''),company_id,deliverable_system_id,deliverable_system_from,is_default,task_archive_tier,allow_personal_feature_params,container_image_at_mode_enabled,created_at,updated_at FROM project_workspace_entries WHERE id=?`, id)
	ws, _ := loadWorkspaceRow(row)
	writeJSON(w, 200, ws)
}

func handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	wid := resolveRequestID(idAliasKindWorkspace, r.Header.Get("X-Resource-Id"), r.Header.Get("X-Trace-Id"))
	row := db.QueryRow(`SELECT id,name,COALESCE(description,''),company_id,deliverable_system_id,deliverable_system_from,is_default,task_archive_tier,allow_personal_feature_params,container_image_at_mode_enabled,created_at,updated_at FROM project_workspace_entries WHERE id=?`, wid)
	ws, err := loadWorkspaceRow(row)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	writeJSON(w, 200, ws)
}

func handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	wid := resolveRequestID(idAliasKindWorkspace, r.Header.Get("X-Resource-Id"), r.Header.Get("X-Trace-Id"))
	body, _ := readJSONBody(r)
	if v := strField(body, "name"); v != "" {
		db.Exec("UPDATE project_workspace_entries SET name=? WHERE id=?", v, wid)
	}
	if v := strField(body, "description"); v != "" {
		db.Exec("UPDATE project_workspace_entries SET description=? WHERE id=?", v, wid)
	}
	if v, ok := body["deliverable_system_id"]; ok {
		db.Exec("UPDATE project_workspace_entries SET deliverable_system_id=? WHERE id=?", strField(body, "deliverable_system_id"), wid)
		_ = v
	}
	if v := strField(body, "deliverable_system_from"); v != "" {
		db.Exec("UPDATE project_workspace_entries SET deliverable_system_from=? WHERE id=?", v, wid)
	}
	if v, ok := body["allow_personal_feature_params"]; ok {
		flag := 0
		if b, ok := v.(bool); ok && b {
			flag = 1
		}
		db.Exec("UPDATE project_workspace_entries SET allow_personal_feature_params=? WHERE id=?", flag, wid)
	}
	if v, ok := body["container_image_at_mode_enabled"]; ok {
		flag := 0
		if b, ok := v.(bool); ok && b {
			flag = 1
		}
		db.Exec("UPDATE project_workspace_entries SET container_image_at_mode_enabled=? WHERE id=?", flag, wid)
		logInfo("workspace container_image_at_mode_enabled updated: "+wid, r.Header.Get("X-Trace-Id"))
	}
	if _, ok := body["is_default"]; ok {
		flag := boolFieldDefault(body, "is_default", false)
		if err := applyWorkspaceDefaultFlag(getAuthTenant(r), wid, flag); err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		logInfo(fmt.Sprintf("workspace default updated: %s is_default=%v", wid, flag), r.Header.Get("X-Trace-Id"))
	}
	if v := strField(body, "task_archive_tier"); v != "" {
		db.Exec("UPDATE project_workspace_entries SET task_archive_tier=? WHERE id=?", v, wid)
	}
	handleGetWorkspace(w, r)
}

func handleInternalGetWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	workspaceID = resolveRequestID(idAliasKindWorkspace, workspaceID, r.Header.Get("X-Trace-Id"))
	row := db.QueryRow(`SELECT id,name,COALESCE(description,''),company_id,deliverable_system_id,deliverable_system_from,is_default,task_archive_tier,allow_personal_feature_params,container_image_at_mode_enabled,created_at,updated_at FROM project_workspace_entries WHERE id=?`, workspaceID)
	ws, err := loadWorkspaceRow(row)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

func handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	wid := resolveRequestID(idAliasKindWorkspace, r.Header.Get("X-Resource-Id"), r.Header.Get("X-Trace-Id"))
	var isDefault bool
	err := db.QueryRow(`SELECT is_default FROM project_workspace_entries WHERE id=?`, wid).Scan(&isDefault)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if isDefault {
		writeError(w, r, 400, "默认工作空间不可删除")
		return
	}
	db.Exec("DELETE FROM project_workspace_accesses WHERE workspace_id=?", wid)
	db.Exec("DELETE FROM project_workspaces WHERE workspace_id=?", wid)
	db.Exec("DELETE FROM project_workspace_entries WHERE id=?", wid)
	logInfo("workspace deleted: "+wid, r.Header.Get("X-Trace-Id"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func handleWorkspaceAccess(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, _ := readJSONBody(r)
		wsID := strField(body, "workspace_id")
		userID := strField(body, "user_id")
		groupID := strField(body, "group_id")
		perm := strField(body, "permission")
		if perm == "" {
			perm = strField(body, "role")
		}
		if wsID == "" || perm == "" {
			writeError(w, r, 400, "workspace_id and permission are required")
			return
		}
		if userID == "" && groupID == "" {
			writeError(w, r, 400, "user_id or group_id is required")
			return
		}
		id := genID("wa")
		_, err := db.Exec(`INSERT INTO project_workspace_accesses(id,workspace_id,user_id,group_id,permission) VALUES(?,?,?,?,?)`,
			id, wsID, userID, groupID, perm)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 201, map[string]interface{}{
			"id": id, "workspace_id": wsID, "user_id": userID, "group_id": groupID, "permission": perm,
		})
	case http.MethodGet:
		wsFilter := r.URL.Query().Get("workspace_id")
		query := "SELECT id,workspace_id,user_id,group_id,permission,created_at FROM project_workspace_accesses"
		args := []interface{}{}
		if wsFilter != "" {
			query += " WHERE workspace_id=?"
			args = append(args, wsFilter)
		}
		rows, err := db.Query(query, args...)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		defer rows.Close()
		list := []map[string]interface{}{}
		for rows.Next() {
			var id, wsID, userID, groupID, perm string
			var ca time.Time
			rows.Scan(&id, &wsID, &userID, &groupID, &perm, &ca)
			list = append(list, map[string]interface{}{
				"id": id, "workspace_id": wsID, "user_id": userID, "group_id": groupID,
				"permission": perm, "created_at": ca.UTC().Format(time.RFC3339Nano),
			})
		}
		writeJSON(w, 200, list)
	case http.MethodDelete:
		body, _ := readJSONBody(r)
		wsID := strField(body, "workspace_id")
		userID := strField(body, "user_id")
		groupID := strField(body, "group_id")
		if wsID == "" {
			writeError(w, r, 400, "workspace_id is required")
			return
		}
		if userID != "" {
			db.Exec("DELETE FROM project_workspace_accesses WHERE workspace_id=? AND user_id=?", wsID, userID)
		} else if groupID != "" {
			db.Exec("DELETE FROM project_workspace_accesses WHERE workspace_id=? AND group_id=?", wsID, groupID)
		} else {
			writeError(w, r, 400, "user_id or group_id is required")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "deleted"})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

// handleInternalWorkspacesBatchGet serves POST /api/internal/workspaces/batch-get/
// Body: {"workspace_ids": ["id1","id2",...]}
// Returns: {"workspaces": [{id, name, company_id, description, is_default}, ...]}
// Max 500 workspace_ids per request.
func handleInternalWorkspacesBatchGet(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, _ := body["workspace_ids"].([]interface{})
	if len(rawIDs) == 0 {
		writeJSON(w, 200, map[string]interface{}{"workspaces": []interface{}{}})
		return
	}

	// Deduplicate and collect
	ids := make([]string, 0, len(rawIDs))
	seen := map[string]bool{}
	for _, raw := range rawIDs {
		wid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		wid = resolveStoredID(idAliasKindWorkspace, wid)
		if wid == "" || wid == "<nil>" || seen[wid] {
			continue
		}
		seen[wid] = true
		ids = append(ids, wid)
		if len(ids) >= 500 {
			break
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]interface{}{"workspaces": []interface{}{}})
		return
	}

	// Build IN query
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := db.Query(`SELECT id, name, COALESCE(description,''), company_id, is_default
		FROM project_workspace_entries WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()

	workspaces := make([]map[string]interface{}, 0, len(ids))
	for rows.Next() {
		var id, name, desc, companyID string
		var isDefault bool
		if err := rows.Scan(&id, &name, &desc, &companyID, &isDefault); err != nil {
			continue
		}
		workspaces = append(workspaces, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": desc,
			"company_id":  companyID,
			"is_default":  isDefault,
		})
	}
	if workspaces == nil {
		workspaces = []map[string]interface{}{}
	}
	writeJSON(w, 200, map[string]interface{}{"workspaces": workspaces})
}
