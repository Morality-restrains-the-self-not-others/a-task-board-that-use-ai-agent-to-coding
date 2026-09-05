package main

import (
	"net/http"
	"strings"
)

// ============================================================================
// System-level progress systems (admin)
// ============================================================================

func handleSystemProgressSystems(w http.ResponseWriter, r *http.Request) {
	// path: /api/system-admin/progress-systems/ or /api/system/progress-systems/
	switch r.Method {
	case http.MethodGet:
		systems, err := listSystemProgressSystems()
		if err != nil {
			writeError(w, r, 500, "查询失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
	case http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid JSON")
			return
		}
		name := strings.TrimSpace(strField(body, "name"))
		if name == "" {
			writeError(w, r, 400, "名称不能为空")
			return
		}
		sysID := genID("ps")
		tx, _ := db.Begin()
		if _, err := tx.Exec("INSERT INTO project_progress_systems(id,name) VALUES(?,?)", sysID, name); err != nil {
			tx.Rollback()
			writeError(w, r, 500, "创建进度体系失败: "+err.Error())
			return
		}
		cols, _ := body["columns"].([]interface{})
		for i, c := range cols {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			colName := strings.TrimSpace(strField(cm, "name"))
			if colName == "" {
				continue
			}
			colID := genID("pc")
			tx.Exec("INSERT INTO project_progress_columns(id,system_id,name,order_num) VALUES(?,?,?,?)", colID, sysID, colName, i)
		}
		tx.Commit()
		systems, _ := listSystemProgressSystems()
		writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

func handleSystemProgressSystemByID(w http.ResponseWriter, r *http.Request) {
	// path: /api/system-admin/progress-systems/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	systemID := parts[len(parts)-1]

	switch r.Method {
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid JSON")
			return
		}
		name := strings.TrimSpace(strField(body, "name"))
		if name == "" {
			writeError(w, r, 400, "名称不能为空")
			return
		}
		var exists bool
		db.QueryRow("SELECT 1 FROM project_progress_systems WHERE id=?", systemID).Scan(&exists)
		if !exists {
			writeError(w, r, 404, "进度体系不存在")
			return
		}
		tx, _ := db.Begin()
		tx.Exec("UPDATE project_progress_systems SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", name, systemID)
		tx.Exec("DELETE FROM project_progress_columns WHERE system_id=?", systemID)
		cols, _ := body["columns"].([]interface{})
		for i, c := range cols {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			colName := strings.TrimSpace(strField(cm, "name"))
			if colName == "" {
				continue
			}
			colID := genID("pc")
			tx.Exec("INSERT INTO project_progress_columns(id,system_id,name,order_num) VALUES(?,?,?,?)", colID, systemID, colName, i)
		}
		tx.Commit()
		systems, _ := listSystemProgressSystems()
		writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
	case http.MethodDelete:
		var isDefault bool
		db.QueryRow("SELECT is_default FROM project_progress_systems WHERE id=?", systemID).Scan(&isDefault)
		if isDefault {
			writeError(w, r, 400, "不能删除默认进度体系")
			return
		}
		tx, _ := db.Begin()
		tx.Exec("DELETE FROM project_progress_columns WHERE system_id=?", systemID)
		tx.Exec("DELETE FROM project_progress_systems WHERE id=?", systemID)
		tx.Commit()
		systems, _ := listSystemProgressSystems()
		writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

func handleSystemProgressSystemSetDefault(w http.ResponseWriter, r *http.Request) {
	// path: /api/system-admin/progress-systems/{id}/set-default
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	systemID := parts[len(parts)-2] // .../{id}/set-default

	tx, _ := db.Begin()
	tx.Exec("UPDATE project_progress_systems SET is_default=0")
	tx.Exec("UPDATE project_progress_systems SET is_default=1, updated_at=CURRENT_TIMESTAMP WHERE id=?", systemID)
	tx.Commit()
	systems, _ := listSystemProgressSystems()
	writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
}

// ============================================================================
// Tenant-level progress systems
// ============================================================================

func handleTenantProgressSystems(w http.ResponseWriter, r *http.Request, tenantID string, segs []string) {
	// path: /api/tenant/{tenantID}/progress-systems/[/{id}/set-default]
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodGet:
			workspaceID := r.URL.Query().Get("workspace_id")
			systems, err := listTenantProgressSystems(tenantID)
			if err != nil {
				writeError(w, r, 500, "查询失败: "+err.Error())
				return
			}
			// annotate with is_default + is_workspace_default
			var defaultTargetID string
			var defaultTargetType string
			db.QueryRow("SELECT target_type, target_id FROM project_progress_systems_default_tenant WHERE tenant_id=?",
				tenantID).Scan(&defaultTargetType, &defaultTargetID)
			var wsTargetID string
			if workspaceID != "" {
				db.QueryRow("SELECT target_id FROM project_progress_systems_workspace WHERE tenant_id=? AND workspace_id=?",
					tenantID, workspaceID).Scan(&wsTargetID)
			}
			for i, s := range systems {
				s["is_default"] = defaultTargetType == "tenant" && s["id"] == defaultTargetID
				if workspaceID != "" {
					s["is_workspace_default"] = s["id"] == wsTargetID
				}
				systems[i] = s
			}
			resp := map[string]interface{}{
				"status":                   "success",
				"project_progress_systems": systems,
			}
			if workspaceID != "" {
				resp["workspace_id"] = workspaceID
				resp["workspace_progress_system_id"] = wsTargetID
			}
			writeJSON(w, 200, resp)
		case http.MethodPost:
			body, err := readJSONBody(r)
			if err != nil {
				writeError(w, r, 400, "invalid JSON")
				return
			}
			name := strings.TrimSpace(strField(body, "name"))
			if name == "" {
				writeError(w, r, 400, "名称不能为空")
				return
			}
			// check uniqueness within tenant
			var dup string
			db.QueryRow("SELECT id FROM project_progress_systems_tenant WHERE tenant_id=? AND name=?", tenantID, name).Scan(&dup)
			if dup != "" {
				writeError(w, r, 400, "该租户下已存在同名进度体系")
				return
			}
			sysID := genID("pts")
			tx, _ := db.Begin()
			tx.Exec("INSERT INTO project_progress_systems_tenant(id,tenant_id,name) VALUES(?,?,?)", sysID, tenantID, name)
			cols, _ := body["columns"].([]interface{})
			for i, c := range cols {
				cm, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				colName := strings.TrimSpace(strField(cm, "name"))
				if colName == "" {
					continue
				}
				colID := genID("ptc")
				tx.Exec("INSERT INTO project_progress_columns_tenant(id,system_id,name,order_num) VALUES(?,?,?,?)", colID, sysID, colName, i)
			}
			tx.Commit()
			systems, _ := listTenantProgressSystems(tenantID)
			writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
		default:
			writeError(w, r, 405, "method not allowed")
		}
		return
	}

	systemID := segs[0]
	// /{id}/set-default
	if len(segs) >= 2 && segs[1] == "set-default" {
		handleTenantProgressSystemSetDefault(w, r, tenantID, systemID)
		return
	}

	// /{id} — update/delete
	switch r.Method {
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid JSON")
			return
		}
		name := strings.TrimSpace(strField(body, "name"))
		if name == "" {
			writeError(w, r, 400, "名称不能为空")
			return
		}
		var exists bool
		db.QueryRow("SELECT 1 FROM project_progress_systems_tenant WHERE id=? AND tenant_id=?", systemID, tenantID).Scan(&exists)
		if !exists {
			writeError(w, r, 404, "进度体系不存在")
			return
		}
		// uniqueness check (exclude self)
		var dup string
		db.QueryRow("SELECT id FROM project_progress_systems_tenant WHERE tenant_id=? AND name=? AND id!=?", tenantID, name, systemID).Scan(&dup)
		if dup != "" {
			writeError(w, r, 400, "该租户下已存在同名进度体系")
			return
		}
		tx, _ := db.Begin()
		tx.Exec("UPDATE project_progress_systems_tenant SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", name, systemID)
		tx.Exec("DELETE FROM project_progress_columns_tenant WHERE system_id=?", systemID)
		cols, _ := body["columns"].([]interface{})
		for i, c := range cols {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			colName := strings.TrimSpace(strField(cm, "name"))
			if colName == "" {
				continue
			}
			colID := genID("ptc")
			tx.Exec("INSERT INTO project_progress_columns_tenant(id,system_id,name,order_num) VALUES(?,?,?,?)", colID, systemID, colName, i)
		}
		tx.Commit()
		systems, _ := listTenantProgressSystems(tenantID)
		writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
	case http.MethodDelete:
		// check not default
		var defTargetID string
		db.QueryRow("SELECT target_id FROM project_progress_systems_default_tenant WHERE tenant_id=?", tenantID).Scan(&defTargetID)
		if defTargetID == systemID {
			writeError(w, r, 400, "不能删除默认进度体系")
			return
		}
		tx, _ := db.Begin()
		tx.Exec("DELETE FROM project_progress_columns_tenant WHERE system_id=?", systemID)
		tx.Exec("DELETE FROM project_progress_systems_tenant WHERE id=? AND tenant_id=?", systemID, tenantID)
		tx.Commit()
		systems, _ := listTenantProgressSystems(tenantID)
		writeJSON(w, 200, map[string]interface{}{"status": "success", "project_progress_systems": systems})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

func handleTenantProgressSystemSetDefault(w http.ResponseWriter, r *http.Request, tenantID, systemID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	// verify system belongs to tenant
	var exists bool
	db.QueryRow("SELECT 1 FROM project_progress_systems_tenant WHERE id=? AND tenant_id=?", systemID, tenantID).Scan(&exists)
	if !exists {
		writeError(w, r, 404, "进度体系不存在")
		return
	}
	// upsert default
	var existingID string
	db.QueryRow("SELECT id FROM project_progress_systems_default_tenant WHERE tenant_id=?", tenantID).Scan(&existingID)
	if existingID != "" {
		db.Exec("UPDATE project_progress_systems_default_tenant SET target_type='tenant', target_id=?, updated_at=CURRENT_TIMESTAMP WHERE tenant_id=?",
			systemID, tenantID)
	} else {
		db.Exec("INSERT INTO project_progress_systems_default_tenant(id,tenant_id,target_type,target_id) VALUES(?,?,'tenant',?)",
			genID("psdt"), tenantID, systemID)
	}
	systems, _ := listTenantProgressSystems(tenantID)
	// mark the default
	for i, s := range systems {
		s["is_default"] = s["id"] == systemID
		systems[i] = s
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":                   "success",
		"message":                  "默认进度体系设置成功",
		"project_progress_systems": systems,
	})
}
