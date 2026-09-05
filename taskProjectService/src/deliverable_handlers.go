package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// Route dispatcher for deliverable systems
// ============================================================================

func handleDeliverableSystemsRoute(w http.ResponseWriter, r *http.Request, tenantID string, segs []string) {
	// segs: [] or [{id}] or [{id}, "set-default"]
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodGet:
			handleListDeliverableSystems(w, r)
		case http.MethodPost:
			handleCreateDeliverableSystem(w, r)
		default:
			writeError(w, r, 405, "method not allowed")
		}
		return
	}
	systemID := segs[0]
	if len(segs) >= 2 && segs[1] == "set-default" {
		handleSetDefaultDeliverableSystem(w, r, systemID)
		return
	}
	handleDeliverableSystemDetail(w, r, systemID)
}

// ============================================================================
// Company deliverable systems — list / create
// ============================================================================

func handleListDeliverableSystems(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	rows, err := db.Query(
		`SELECT id,name,COALESCE(description,''),company_id,is_system,is_default,created_at,updated_at
		 FROM project_deliverable_systems WHERE company_id=? OR is_system=1 ORDER BY name`, tenantID)
	if err != nil {
		writeError(w, r, 500, "查询失败: "+err.Error())
		return
	}
	defer rows.Close()

	// load tenant default deliverable system — 独立表存储（OPT-20260808-004：
	// 进度默认表 UNIQUE(tenant_id) 单行会被进度体系默认覆盖，导致租户默认交付物体系恒丢失）
	defaultTargetID := tenantDeliverableDefaultID(tenantID)

	systems := []map[string]interface{}{}
	for rows.Next() {
		var id, name, desc, cid string
		var isSys, isDef bool
		var ca, ua time.Time
		if err := rows.Scan(&id, &name, &desc, &cid, &isSys, &isDef, &ca, &ua); err != nil {
			continue
		}
		columns := loadDeliverableColumns(id)
		levelNames := make([]string, len(columns))
		for i, c := range columns {
			levelNames[i] = c["name"].(string)
		}
		systems = append(systems, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": desc,
			"company_id":  cid,
			"is_system":   isSys,
			"is_default":  id == defaultTargetID,
			"level_names": levelNames,
			"created_at":  ca.UTC().Format(time.RFC3339Nano),
			"updated_at":  ua.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, 200, systems)
}

func handleCreateDeliverableSystem(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
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
	// check uniqueness within company
	var dup string
	db.QueryRow("SELECT id FROM project_deliverable_systems WHERE company_id=? AND name=?", tenantID, name).Scan(&dup)
	if dup != "" {
		writeError(w, r, 400, "该公司下已存在同名交付物体系")
		return
	}

	id := genID("ds")
	desc := strings.TrimSpace(strField(body, "description"))
	tx, _ := db.Begin()
	tx.Exec("INSERT INTO project_deliverable_systems(id,name,description,company_id,is_system) VALUES(?,?,?,?,0)",
		id, name, desc, tenantID)

	// create columns (level_names)
	levelNames := extractLevelNames(body)
	for i, ln := range levelNames {
		colID := genID("dc")
		tx.Exec("INSERT INTO project_deliverable_columns(id,system_id,name,order_num) VALUES(?,?,?,?)",
			colID, id, ln, i)
	}
	tx.Commit()

	columns := loadDeliverableColumns(id)
	writeJSON(w, 201, map[string]interface{}{
		"status": "success", "id": id,
		"name": name, "description": desc, "company_id": tenantID,
		"is_system": false, "is_default": false,
		"level_names": extractLevelNames(body),
		"columns":     columns,
	})
}

// ============================================================================
// Company deliverable system — detail (GET / PUT / DELETE)
// ============================================================================

func handleDeliverableSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	tenantID := getAuthTenant(r)

	switch r.Method {
	case http.MethodGet:
		var id, name, desc, cid string
		var isSys, isDef bool
		var ca, ua time.Time
		err := db.QueryRow(
			`SELECT id,name,COALESCE(description,''),company_id,is_system,is_default,created_at,updated_at
			 FROM project_deliverable_systems WHERE id=?`, systemID).Scan(&id, &name, &desc, &cid, &isSys, &isDef, &ca, &ua)
		if err == sql.ErrNoRows {
			writeError(w, r, 404, "交付物体系不存在")
			return
		}
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		columns := loadDeliverableColumns(systemID)
		writeJSON(w, 200, map[string]interface{}{
			"id": id, "name": name, "description": desc, "company_id": cid,
			"is_system": isSys, "is_default": isDef,
			"columns":    columns,
			"created_at": ca.UTC().Format(time.RFC3339Nano),
			"updated_at": ua.UTC().Format(time.RFC3339Nano),
		})

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
		db.QueryRow("SELECT 1 FROM project_deliverable_systems WHERE id=?", systemID).Scan(&exists)
		if !exists {
			writeError(w, r, 404, "交付物体系不存在")
			return
		}
		// uniqueness check (exclude self)
		var dup string
		db.QueryRow("SELECT id FROM project_deliverable_systems WHERE company_id=? AND name=? AND id!=?",
			tenantID, name, systemID).Scan(&dup)
		if dup != "" {
			writeError(w, r, 400, "该公司下已存在同名交付物体系")
			return
		}
		desc := strings.TrimSpace(strField(body, "description"))
		tx, _ := db.Begin()
		tx.Exec("UPDATE project_deliverable_systems SET name=?, description=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			name, desc, systemID)
		// replace columns if level_names provided
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
		columns := loadDeliverableColumns(systemID)
		writeJSON(w, 200, map[string]interface{}{
			"status": "success", "id": systemID,
			"name": name, "description": desc, "company_id": tenantID,
			"is_system": false,
			"columns":   columns,
		})

	case http.MethodDelete:
		var isSys bool
		err := db.QueryRow("SELECT is_system FROM project_deliverable_systems WHERE id=?", systemID).Scan(&isSys)
		if err == sql.ErrNoRows {
			writeError(w, r, 404, "交付物体系不存在")
			return
		}
		if isSys {
			writeError(w, r, 400, "不能删除系统交付物体系")
			return
		}
		// check not default (OPT-20260808-004: 交付物默认独立表)
		if tenantDeliverableDefaultID(tenantID) == systemID {
			writeError(w, r, 400, "不能删除默认交付物体系")
			return
		}
		tx, _ := db.Begin()
		tx.Exec("DELETE FROM project_deliverable_columns WHERE system_id=?", systemID)
		tx.Exec("DELETE FROM project_deliverable_systems WHERE id=?", systemID)
		tx.Commit()
		writeJSON(w, 200, map[string]interface{}{
			"status":  "success",
			"message": "交付物体系已删除",
		})

	default:
		writeError(w, r, 405, "method not allowed")
	}
}

// ============================================================================
// Set default deliverable system for tenant
// ============================================================================

func handleSetDefaultDeliverableSystem(w http.ResponseWriter, r *http.Request, systemID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	tenantID := getAuthTenant(r)

	// verify system exists
	var exists bool
	db.QueryRow("SELECT 1 FROM project_deliverable_systems WHERE id=?", systemID).Scan(&exists)
	if !exists {
		writeError(w, r, 404, "交付物体系不存在")
		return
	}

	// upsert tenant default deliverable system — 独立表存储（OPT-20260808-004 根因：
	// 原实现写入 project_progress_systems_default_tenant（UNIQUE tenant_id），
	// 被进度体系默认（company-created intent 2）覆盖，默认交付物体系恒丢失）
	var existingID string
	db.QueryRow("SELECT id FROM project_deliverable_systems_default_tenant WHERE tenant_id=?", tenantID).Scan(&existingID)
	if existingID != "" {
		db.Exec("UPDATE project_deliverable_systems_default_tenant SET deliverable_id=?, updated_at=CURRENT_TIMESTAMP WHERE tenant_id=?",
			systemID, tenantID)
	} else {
		db.Exec("INSERT INTO project_deliverable_systems_default_tenant(id,tenant_id,deliverable_id) VALUES(?,?,?)",
			genID("ddst"), tenantID, systemID)
	}

	// return updated list with is_default annotations
	rows, _ := db.Query(
		`SELECT id,name,COALESCE(description,''),company_id,is_system,is_default,created_at,updated_at
		 FROM project_deliverable_systems WHERE company_id=? OR is_system=1 ORDER BY name`, tenantID)
	defer rows.Close()
	systems := []map[string]interface{}{}
	for rows.Next() {
		var id, name, desc, cid string
		var isSys, isDef bool
		var ca, ua time.Time
		if err := rows.Scan(&id, &name, &desc, &cid, &isSys, &isDef, &ca, &ua); err != nil {
			continue
		}
		columns := loadDeliverableColumns(id)
		levelNames := make([]string, len(columns))
		for i, c := range columns {
			levelNames[i] = c["name"].(string)
		}
		systems = append(systems, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": desc,
			"company_id":  cid,
			"is_system":   isSys,
			"is_default":  id == systemID,
			"level_names": levelNames,
			"created_at":  ca.UTC().Format(time.RFC3339Nano),
			"updated_at":  ua.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":  "success",
		"message": "已设置为默认交付物体系",
		"systems": systems,
	})
}

// ============================================================================
// Tenant default deliverable system lookup
// ============================================================================

// tenantDeliverableDefaultID returns the tenant's default deliverable system id,
// or "" when the tenant has none set.
func tenantDeliverableDefaultID(tenantID string) string {
	var id string
	db.QueryRow(
		"SELECT deliverable_id FROM project_deliverable_systems_default_tenant WHERE tenant_id=?",
		tenantID).Scan(&id)
	return id
}

// handleGetDefaultDeliverableSystem serves GET /api/projects/default-deliverable-system/tenant_id/{tid}/
// — 前端 DeliverableSystemList 依赖该端点标记 is_default（曾因未注册而 404，
// 导致页面恒显示「暂无默认交付物体系」，OPT-20260808-004）。
func handleGetDefaultDeliverableSystem(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	deliverableID := tenantDeliverableDefaultID(tenantID)
	if deliverableID == "" {
		writeJSON(w, 200, map[string]interface{}{
			"status":                        "success",
			"default_deliverable_system_id": nil,
		})
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":                        "success",
		"default_deliverable_system_id": deliverableID,
	})
}

// ============================================================================
// Helpers
// ============================================================================

// writeError 输出带 trace_id 的错误响应体：即使网关/代理剥离 X-Trace-Id 响应头，
// 前端仍可从响应体提取 traceId（apiUtils resolveRequestTraceId 的 body 兜底）。
// OPT-20260807-053 加固：缺失 data-traceId 的根因之一即错误响应无 trace 信息。
// OPT-20260807-059：同时输出 error 与 message 两个键——前端有的调用点直接读
// `data.error`（字符串语义），有的读 `data.message`（apiUtils 优先 message），
// 双键并存保证 writeError 可无差别替换纯 `{"error": ...}` 响应而不破坏任一读取方。
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]interface{}{
		"status":  "error",
		"error":   message,
		"message": message,
	}
	if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorMap 与 writeError 同语义但保留调用方附加的响应字段（如 git_repos、
// oauth_bound 等前端表单校验/流程状态需要的数据键），同时注入 trace_id。
// OPT-20260807-059：纯 `{"error": ...}` 之外的复合错误体统一走此入口，
// 保证 body 兜底契约（trace_id）对全部错误分支成立。
func writeErrorMap(w http.ResponseWriter, r *http.Request, status int, body map[string]interface{}) {
	if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
		body["trace_id"] = tid
	}
	if _, has := body["status"]; !has {
		body["status"] = "error"
	}
	if _, has := body["message"]; !has {
		if err, ok := body["error"]; ok {
			if s, ok := err.(string); ok {
				body["message"] = s
			}
		}
	}
	writeJSON(w, status, body)
}

func loadDeliverableColumns(systemID string) []map[string]interface{} {
	rows, err := db.Query(
		"SELECT id,name,order_num FROM project_deliverable_columns WHERE system_id=? ORDER BY order_num", systemID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, name string
		var order int
		if err := rows.Scan(&id, &name, &order); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": id, "name": name, "order": order,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out
}

// extractLevelNames extracts column names from request body,
// supporting both "level_names" (array of strings) and "columns" (array of objects with "name").
func extractLevelNames(body map[string]interface{}) []string {
	// try "level_names" first
	if raw, ok := body["level_names"].([]interface{}); ok {
		out := make([]string, 0, len(raw))
		for _, r := range raw {
			s := strings.TrimSpace(stringField(r))
			if s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// try "columns" (array of {name: string})
	if raw, ok := body["columns"].([]interface{}); ok {
		out := make([]string, 0, len(raw))
		for _, r := range raw {
			if m, ok := r.(map[string]interface{}); ok {
				s := strings.TrimSpace(strField(m, "name"))
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	}
	return nil
}

func stringField(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

// ============================================================================
// Internal deliverable system lookup (used by taskTaskService)
// ============================================================================

func handleInternalDeliverableSystemLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	systemID := strings.TrimSpace(r.URL.Query().Get("id"))
	if systemID == "" {
		writeError(w, r, 400, "id is required")
		return
	}
	var id, name string
	var isSys, isDef bool
	err := db.QueryRow(
		"SELECT id, name, is_system, is_default FROM project_deliverable_systems WHERE id=?",
		systemID).Scan(&id, &name, &isSys, &isDef)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"id":         id,
		"name":       name,
		"is_system":  isSys,
		"is_default": isDef,
	})
}

// ============================================================================
// Legacy manage-deliverable-system (OPT-049: replaces Django manage-deliverable-system)
// ============================================================================

func handleLegacyManageDeliverableSystem(w http.ResponseWriter, r *http.Request, tenantID string) {
	// path: /api/tenant/{tenantID}/manage-deliverable-system/ → /api/projects/manage-deliverable-system/tenant_id/{tid}
	// 工作空间查询一律按租户作用域（company_id=tenantID），防跨租户数据泄漏（OPT-20260808-003）。
	switch r.Method {
	case http.MethodGet:
		workspaceID := r.URL.Query().Get("workspace_id")
		var targetID string
		if workspaceID != "" {
			db.QueryRow("SELECT deliverable_system_id FROM project_workspace_entries WHERE id=? AND company_id=?",
				workspaceID, tenantID).Scan(&targetID)
		}
		objs := []map[string]interface{}{}
		if targetID != "" {
			rows, err := db.Query("SELECT id, name, order_num FROM project_deliverable_columns WHERE system_id=? ORDER BY order_num, id", targetID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var id, name string
					var orderNum int
					if rows.Scan(&id, &name, &orderNum) == nil {
						objs = append(objs, map[string]interface{}{
							"id":    id,
							"name":  name,
							"order": orderNum,
							"color": "",
						})
					}
				}
			}
		}
		writeJSON(w, 200, map[string]interface{}{
			"status":                        "success",
			"current_deliverable_system_id": targetID,
			"current_deliverable_objs":      objs,
		})
	case http.MethodPost:
		r.ParseForm()
		action := strings.TrimSpace(r.PostForm.Get("action"))
		workspaceID := strings.TrimSpace(r.PostForm.Get("workspace_id"))
		systemID := strings.TrimSpace(r.PostForm.Get("deliverable_system_id"))
		if action != "update_deliverable_system" || workspaceID == "" {
			writeError(w, r, 400, "参数错误")
			return
		}
		res, err := db.Exec("UPDATE project_workspace_entries SET deliverable_system_id=?, deliverable_system_from=? WHERE id=? AND company_id=?",
			systemID, systemID, workspaceID, tenantID)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			writeError(w, r, 404, "工作空间不存在")
			return
		}
		writeJSON(w, 200, map[string]interface{}{"status": "success"})
	default:
		writeError(w, r, 405, "method not allowed")
	}
}
