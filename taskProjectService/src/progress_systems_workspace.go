package main

import (
	"database/sql"
	"net/http"
	"strings"
)

// ============================================================================
// Workspace progress system binding
// ============================================================================

func handleWorkspaceProgressSystem(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	// path: /api/tenant/{tenantID}/workspaces/{workspaceID}/progress-system/
	// also: /api/tenant/{tenantID}/workspaces/{workspaceID}/column-system/
	switch r.Method {
	case http.MethodGet:
		var targetType, targetID, fallbackLevel string
		err := db.QueryRow(
			"SELECT target_type, target_id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
			tenantID, workspaceID).Scan(&targetType, &targetID)
		if err == sql.ErrNoRows {
			// Fallback 1: tenant default progress system
			err2 := db.QueryRow(
				"SELECT target_type, target_id FROM project_progress_systems_default_tenant WHERE tenant_id=?",
				tenantID).Scan(&targetType, &targetID)
			if err2 == sql.ErrNoRows {
				// Fallback 2: system default progress system
				var sysID string
				err3 := db.QueryRow(
					"SELECT id FROM project_progress_systems WHERE is_default=1 ORDER BY created_at LIMIT 1",
				).Scan(&sysID)
				if err3 == sql.ErrNoRows {
					writeJSON(w, 200, map[string]interface{}{
						"status":             "success",
						"progress_system_id": nil,
						"columns":            []interface{}{},
						"fallback_level":     "none",
					})
					return
				}
				if err3 != nil {
					writeError(w, r, 500, "查询系统默认进度体系失败: "+err3.Error())
					return
				}
				targetType = "system"
				targetID = sysID
				fallbackLevel = "system_default"
			} else if err2 != nil {
				writeError(w, r, 500, "查询租户默认进度体系失败: "+err2.Error())
				return
			} else {
				fallbackLevel = "tenant_default"
			}
		} else if err != nil {
			writeError(w, r, 500, "查询工作空间进度体系失败: "+err.Error())
			return
		}
		columns, _ := resolveProgressSystemColumns(targetType, targetID)
		writeJSON(w, 200, map[string]interface{}{
			"status":             "success",
			"progress_system_id": targetID,
			"columns":            columns,
			"fallback_level":     fallbackLevel,
		})
	case http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid JSON")
			return
		}
		psID := strField(body, "progress_system_id")
		if psID == "" {
			// clear
			db.Exec("DELETE FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?", tenantID, workspaceID)
			writeJSON(w, 200, map[string]interface{}{
				"status":             "success",
				"message":            "已清除工作空间进度体系",
				"progress_system_id": nil,
				"columns":            []interface{}{},
			})
			return
		}
		// resolve whether it's a system or tenant progress system
		targetType, err := resolveProgressSystemType(psID, tenantID)
		if err != nil {
			writeError(w, r, 400, err.Error())
			return
		}
		// upsert workspace binding
		var existingID string
		db.QueryRow("SELECT id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
			tenantID, workspaceID).Scan(&existingID)
		if existingID != "" {
			db.Exec("UPDATE project_progress_systems_workspace SET target_type=?, target_id=?, updated_at=CURRENT_TIMESTAMP WHERE tenant_id=? AND workspace_id=?",
				targetType, psID, tenantID, workspaceID)
		} else {
			db.Exec("INSERT INTO project_progress_systems_workspace(id,tenant_id,workspace_id,target_type,target_id) VALUES(?,?,?,?,?)",
				genID("psw"), tenantID, workspaceID, targetType, psID)
		}
		columns, _ := resolveProgressSystemColumns(targetType, psID)
		writeJSON(w, 200, map[string]interface{}{
			"status":             "success",
			"message":            "工作空间进度体系设置成功",
			"progress_system_id": psID,
			"columns":            columns,
		})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

// ============================================================================
// Tenant default progress system settings
// ============================================================================

func handleTenantDefaultProgressSystem(w http.ResponseWriter, r *http.Request, tenantID string) {
	// path: /api/tenant/{tenantID}/settings/default-progress-system/
	switch r.Method {
	case http.MethodGet:
		var targetType, targetID string
		err := db.QueryRow(
			"SELECT target_type, target_id FROM project_progress_systems_default_tenant WHERE tenant_id=?",
			tenantID).Scan(&targetType, &targetID)
		if err == sql.ErrNoRows {
			writeJSON(w, 200, map[string]interface{}{
				"status":          "success",
				"progress_system": nil,
			})
			return
		}
		ps, _ := resolveProgressSystemTarget(targetType, targetID)
		if ps == nil {
			writeJSON(w, 200, map[string]interface{}{
				"status":          "success",
				"progress_system": nil,
			})
			return
		}
		columns, _ := resolveProgressSystemColumns(targetType, targetID)
		ps["columns"] = columns
		writeJSON(w, 200, map[string]interface{}{
			"status":          "success",
			"progress_system": ps,
		})
	case http.MethodPost:
		// supports both JSON and form-urlencoded
		var systemID string
		if ct := r.Header.Get("Content-Type"); strings.Contains(ct, "application/json") {
			body, _ := readJSONBody(r)
			systemID = strField(body, "system_id")
		} else {
			r.ParseForm()
			systemID = strings.TrimSpace(r.PostForm.Get("system_id"))
		}
		if systemID == "" {
			writeError(w, r, 400, "system_id 不能为空")
			return
		}
		targetType, err := resolveProgressSystemType(systemID, tenantID)
		if err != nil {
			writeError(w, r, 400, err.Error())
			return
		}
		// upsert
		var existingID string
		db.QueryRow("SELECT id FROM project_progress_systems_default_tenant WHERE tenant_id=?", tenantID).Scan(&existingID)
		if existingID != "" {
			db.Exec("UPDATE project_progress_systems_default_tenant SET target_type=?, target_id=?, updated_at=CURRENT_TIMESTAMP WHERE tenant_id=?",
				targetType, systemID, tenantID)
		} else {
			db.Exec("INSERT INTO project_progress_systems_default_tenant(id,tenant_id,target_type,target_id) VALUES(?,?,?,?)",
				genID("psdt"), tenantID, targetType, systemID)
		}
		ps, _ := resolveProgressSystemTarget(targetType, systemID)
		columns, _ := resolveProgressSystemColumns(targetType, systemID)
		if ps != nil {
			ps["columns"] = columns
		}
		writeJSON(w, 200, map[string]interface{}{
			"status":          "success",
			"message":         "默认进度体系设置成功",
			"progress_system": ps,
		})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

// ============================================================================
// Legacy manage-progress-column (form-urlencoded compat)
// ============================================================================

func handleLegacyManageProgressColumn(w http.ResponseWriter, r *http.Request, tenantID string) {
	// path: /api/tenant/{tenantID}/manage-progress-column/
	switch r.Method {
	case http.MethodGet:
		workspaceID := r.URL.Query().Get("workspace_id")
		var targetID string
		if workspaceID != "" {
			db.QueryRow("SELECT target_id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
				tenantID, workspaceID).Scan(&targetID)
		}
		writeJSON(w, 200, map[string]interface{}{
			"status":                     "success",
			"current_progress_system_id": targetID,
		})
	case http.MethodPost:
		r.ParseForm()
		action := strings.TrimSpace(r.PostForm.Get("action"))
		workspaceID := strings.TrimSpace(r.PostForm.Get("workspace_id"))
		systemID := strings.TrimSpace(r.PostForm.Get("progress_system_id"))
		if action != "update_progress_system" || workspaceID == "" {
			writeError(w, r, 400, "参数错误")
			return
		}
		if systemID == "" {
			db.Exec("DELETE FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?", tenantID, workspaceID)
			writeJSON(w, 200, map[string]interface{}{
				"status":             "success",
				"progress_system_id": nil,
				"columns":            []interface{}{},
			})
			return
		}
		targetType, err := resolveProgressSystemType(systemID, tenantID)
		if err != nil {
			writeError(w, r, 400, err.Error())
			return
		}
		var existingID string
		db.QueryRow("SELECT id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
			tenantID, workspaceID).Scan(&existingID)
		if existingID != "" {
			db.Exec("UPDATE project_progress_systems_workspace SET target_type=?, target_id=?, updated_at=CURRENT_TIMESTAMP WHERE tenant_id=? AND workspace_id=?",
				targetType, systemID, tenantID, workspaceID)
		} else {
			db.Exec("INSERT INTO project_progress_systems_workspace(id,tenant_id,workspace_id,target_type,target_id) VALUES(?,?,?,?,?)",
				genID("psw"), tenantID, workspaceID, targetType, systemID)
		}
		columns, _ := resolveProgressSystemColumns(targetType, systemID)
		writeJSON(w, 200, map[string]interface{}{
			"status":             "success",
			"progress_system_id": systemID,
			"columns":            columns,
		})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

// ============================================================================
// Internal first-progress-column resolver (called by taskTaskService for forks,
// OPT-20260818-001). Returns the workspace progress-system's first column so the
// server can force Fork tasks into the board's first column regardless of what a
// non-SPA client sends.
// ============================================================================

func handleInternalResolveFirstProgressColumn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	if tenantID == "" || workspaceID == "" {
		writeError(w, r, 400, "tenant_id, workspace_id required")
		return
	}
	colID, colName, err := resolveFirstProgressColumn(tenantID, workspaceID)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"ok":                   true,
		"progress_column_id":   colID,
		"progress_column_name": colName,
	})
}

// resolveFirstProgressColumn resolves the workspace's progress-system first column
// (workspace binding → tenant default → system default, same fallback as the
// workspace progress-system endpoint). Empty id means no progress system / no columns.
func resolveFirstProgressColumn(tenantID, workspaceID string) (string, string, error) {
	var targetType, targetID string
	err := db.QueryRow(
		"SELECT target_type, target_id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
		tenantID, workspaceID).Scan(&targetType, &targetID)
	if err == sql.ErrNoRows || targetID == "" {
		err2 := db.QueryRow(
			"SELECT target_type, target_id FROM project_progress_systems_default_tenant WHERE tenant_id=?",
			tenantID).Scan(&targetType, &targetID)
		if err2 == sql.ErrNoRows || targetID == "" {
			err3 := db.QueryRow(
				"SELECT id FROM project_progress_systems WHERE is_default=1 ORDER BY created_at LIMIT 1",
			).Scan(&targetID)
			if err3 == sql.ErrNoRows {
				return "", "", nil
			}
			if err3 != nil {
				return "", "", err3
			}
			targetType = "system"
		} else if err2 != nil {
			return "", "", err2
		}
	} else if err != nil {
		return "", "", err
	}

	var colID, colName string
	if targetType == "system" {
		err = db.QueryRow("SELECT id,name FROM project_progress_columns WHERE system_id=? ORDER BY order_num LIMIT 1", targetID).Scan(&colID, &colName)
	} else {
		err = db.QueryRow("SELECT id,name FROM project_progress_columns_tenant WHERE system_id=? ORDER BY order_num LIMIT 1", targetID).Scan(&colID, &colName)
	}
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	return colID, colName, nil
}

// ============================================================================
// Internal validation endpoint (called by taskTaskService)
// ============================================================================

func handleInternalValidateProgressColumn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	progressColumnID := strField(body, "progress_column_id")
	if tenantID == "" || workspaceID == "" || progressColumnID == "" {
		writeError(w, r, 400, "tenant_id, workspace_id, progress_column_id required")
		return
	}
	// look up workspace progress system
	var targetType, targetID string
	err = db.QueryRow(
		"SELECT target_type, target_id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
		tenantID, workspaceID).Scan(&targetType, &targetID)
	if err == sql.ErrNoRows || targetID == "" {
		writeJSON(w, 400, map[string]interface{}{"progress_column_id": []string{"进度列不属于当前工作空间"}})
		return
	}
	// check column exists in the workspace's progress system
	var colName string
	if targetType == "system" {
		db.QueryRow("SELECT name FROM project_progress_columns WHERE id=? AND system_id=?", progressColumnID, targetID).Scan(&colName)
	} else {
		db.QueryRow("SELECT name FROM project_progress_columns_tenant WHERE id=? AND system_id=?", progressColumnID, targetID).Scan(&colName)
	}
	if colName == "" {
		writeJSON(w, 400, map[string]interface{}{"progress_column_id": []string{"进度列不属于当前工作空间"}})
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"ok":                   true,
		"progress_column_name": colName,
	})
}

// ============================================================================
// Shared helpers
// ============================================================================

func listSystemProgressSystems() ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT id,name,is_default FROM project_progress_systems ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var systems []map[string]interface{}
	for rows.Next() {
		var id, name string
		var isDef bool
		rows.Scan(&id, &name, &isDef)
		cols, _ := getSystemColumns(id)
		systems = append(systems, map[string]interface{}{
			"id":         id,
			"name":       name,
			"is_default": isDef,
			"columns":    cols,
		})
	}
	if systems == nil {
		systems = []map[string]interface{}{}
	}
	return systems, nil
}

func listTenantProgressSystems(tenantID string) ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT id,name FROM project_progress_systems_tenant WHERE tenant_id=? ORDER BY created_at", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var systems []map[string]interface{}
	for rows.Next() {
		var id, name string
		rows.Scan(&id, &name)
		cols, _ := getTenantColumns(id)
		systems = append(systems, map[string]interface{}{
			"id":      id,
			"name":    name,
			"columns": cols,
		})
	}
	if systems == nil {
		systems = []map[string]interface{}{}
	}
	return systems, nil
}

func getSystemColumns(systemID string) ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT id,name,order_num FROM project_progress_columns WHERE system_id=? ORDER BY order_num", systemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []map[string]interface{}
	for rows.Next() {
		var id, name string
		var order int
		rows.Scan(&id, &name, &order)
		cols = append(cols, map[string]interface{}{
			"id":    id,
			"name":  name,
			"order": order,
		})
	}
	if cols == nil {
		cols = []map[string]interface{}{}
	}
	return cols, nil
}

func getTenantColumns(systemID string) ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT id,name,order_num FROM project_progress_columns_tenant WHERE system_id=? ORDER BY order_num", systemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []map[string]interface{}
	for rows.Next() {
		var id, name string
		var order int
		rows.Scan(&id, &name, &order)
		cols = append(cols, map[string]interface{}{
			"id":    id,
			"name":  name,
			"order": order,
		})
	}
	if cols == nil {
		cols = []map[string]interface{}{}
	}
	return cols, nil
}

func resolveProgressSystemColumns(targetType, targetID string) ([]map[string]interface{}, error) {
	if targetType == "system" {
		return getSystemColumns(targetID)
	}
	return getTenantColumns(targetID)
}

func resolveProgressSystemTarget(targetType, targetID string) (map[string]interface{}, error) {
	if targetType == "system" {
		var id, name string
		var isDef bool
		err := db.QueryRow("SELECT id,name,is_default FROM project_progress_systems WHERE id=?", targetID).Scan(&id, &name, &isDef)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"id":         id,
			"name":       name,
			"is_default": isDef,
		}, nil
	}
	var id, name string
	err := db.QueryRow("SELECT id,name FROM project_progress_systems_tenant WHERE id=?", targetID).Scan(&id, &name)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":   id,
		"name": name,
	}, nil
}

// resolveProgressSystemType determines whether an ID refers to a system or tenant progress system.
// It checks system-level first, then tenant-level scoped to the given tenant.
func resolveProgressSystemType(progressSystemID, tenantID string) (string, error) {
	var exists bool
	db.QueryRow("SELECT 1 FROM project_progress_systems WHERE id=?", progressSystemID).Scan(&exists)
	if exists {
		return "system", nil
	}
	db.QueryRow("SELECT 1 FROM project_progress_systems_tenant WHERE id=? AND tenant_id=?", progressSystemID, tenantID).Scan(&exists)
	if exists {
		return "tenant", nil
	}
	return "", errNotFound("进度体系不存在")
}

func errNotFound(msg string) error {
	return &notFoundError{msg}
}

type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }
