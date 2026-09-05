package main

import (
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// System-admin deliverable systems
// ============================================================================

func handleSystemDeliverableSystems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		systems, err := listSystemDeliverableSystems()
		if err != nil {
			writeError(w, r, 500, "查询失败: "+err.Error())
			return
		}
		writeJSON(w, 200, systems)
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
		desc := strings.TrimSpace(strField(body, "description"))
		sysID := genID("ds")
		tx, _ := db.Begin()
		if _, err := tx.Exec(
			"INSERT INTO project_deliverable_systems(id,name,description,company_id,is_system,is_default) VALUES(?,?,?,?,1,0)",
			sysID, name, desc, "",
		); err != nil {
			tx.Rollback()
			writeError(w, r, 500, "创建交付物体系失败: "+err.Error())
			return
		}
		levelNames := extractLevelNames(body)
		for i, ln := range levelNames {
			colID := genID("dc")
			tx.Exec("INSERT INTO project_deliverable_columns(id,system_id,name,order_num) VALUES(?,?,?,?)",
				colID, sysID, ln, i)
		}
		tx.Commit()
		systems, _ := listSystemDeliverableSystems()
		writeJSON(w, 201, systems)
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

func handleSystemDeliverableSystemByID(w http.ResponseWriter, r *http.Request, systemID string) {
	switch r.Method {
	case http.MethodGet:
		systems, err := listSystemDeliverableSystems()
		if err != nil {
			writeError(w, r, 500, "查询失败: "+err.Error())
			return
		}
		for _, s := range systems {
			if s["id"] == systemID {
				writeJSON(w, 200, s)
				return
			}
		}
		writeError(w, r, 404, "交付物体系不存在")
	case http.MethodPut, http.MethodPatch:
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
		db.QueryRow("SELECT 1 FROM project_deliverable_systems WHERE id=? AND is_system=1", systemID).Scan(&exists)
		if !exists {
			writeError(w, r, 404, "交付物体系不存在")
			return
		}
		desc := strings.TrimSpace(strField(body, "description"))
		tx, _ := db.Begin()
		tx.Exec("UPDATE project_deliverable_systems SET name=?, description=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			name, desc, systemID)
		levelNames := extractLevelNames(body)
		if len(levelNames) > 0 {
			tx.Exec("DELETE FROM project_deliverable_columns WHERE system_id=?", systemID)
			for i, ln := range levelNames {
				colID := genID("dc")
				tx.Exec("INSERT INTO project_deliverable_columns(id,system_id,name,order_num) VALUES(?,?,?,?)",
					colID, systemID, ln, i)
			}
		}
		tx.Commit()
		systems, _ := listSystemDeliverableSystems()
		writeJSON(w, 200, systems)
	case http.MethodDelete:
		var isDef bool
		db.QueryRow("SELECT is_default FROM project_deliverable_systems WHERE id=?", systemID).Scan(&isDef)
		if isDef {
			writeError(w, r, 400, "不能删除默认交付物体系")
			return
		}
		var exists bool
		db.QueryRow("SELECT 1 FROM project_deliverable_systems WHERE id=? AND is_system=1", systemID).Scan(&exists)
		if !exists {
			writeError(w, r, 404, "交付物体系不存在")
			return
		}
		tx, _ := db.Begin()
		tx.Exec("DELETE FROM project_deliverable_columns WHERE system_id=?", systemID)
		tx.Exec("DELETE FROM project_deliverable_systems WHERE id=?", systemID)
		tx.Commit()
		systems, _ := listSystemDeliverableSystems()
		writeJSON(w, 200, systems)
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

func listSystemDeliverableSystems() ([]map[string]interface{}, error) {
	rows, err := db.Query(
		"SELECT id,name,description,is_system,is_default,created_at,updated_at FROM project_deliverable_systems WHERE is_system=1 ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var systems []map[string]interface{}
	for rows.Next() {
		var id, name, desc string
		var isSys, isDef bool
		var ca, ua time.Time
		if err := rows.Scan(&id, &name, &desc, &isSys, &isDef, &ca, &ua); err != nil {
			continue
		}
		columns := loadDeliverableColumns(id)
		levelNames := make([]string, len(columns))
		for i, c := range columns {
			levelNames[i] = c["name"].(string)
		}
		systems = append(systems, map[string]interface{}{
			"id":              id,
			"name":            name,
			"description":     desc,
			"is_system":       isSys,
			"is_default":      isDef,
			"level_names":     levelNames,
			"level_structure": strings.Join(levelNames, " > "),
			"created_at":      ca.UTC().Format(time.RFC3339Nano),
			"updated_at":      ua.UTC().Format(time.RFC3339Nano),
		})
	}
	if systems == nil {
		systems = []map[string]interface{}{}
	}
	return systems, nil
}
