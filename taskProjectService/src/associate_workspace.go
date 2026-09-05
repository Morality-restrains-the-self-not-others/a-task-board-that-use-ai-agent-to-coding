package main

import (
	"database/sql"
	"net/http"
	"strings"
)

// handleAssociateWorkspace — POST /api/tenant/{tid}/projects/{pid}/associateWorkspace/
// body: {"workspace_id":"..."} 添加关联；{"workspace_id":"...","action":"remove"} 移除关联。
func handleAssociateWorkspace(w http.ResponseWriter, r *http.Request, tenantID, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON")
		return
	}
	wsID := strField(body, "workspace_id")
	if wsID == "" {
		writeError(w, r, http.StatusBadRequest, "workspace_id 不能为空")
		return
	}
	var projCompany string
	err = db.QueryRow("SELECT company_id FROM project_entries WHERE id=? AND company_id=?", projectID, tenantID).Scan(&projCompany)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "项目不存在或无权访问")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	var wsCompany string
	err = db.QueryRow("SELECT company_id FROM project_workspace_entries WHERE id=? AND company_id=?", wsID, tenantID).Scan(&wsCompany)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "工作空间不存在或无权访问")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	action := strings.ToLower(strField(body, "action"))
	if action == "remove" {
		if _, err := db.Exec("DELETE FROM project_workspaces WHERE project_id=? AND workspace_id=?", projectID, wsID); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		if _, err := db.Exec("INSERT IGNORE INTO project_workspaces(project_id, workspace_id) VALUES(?,?)", projectID, wsID); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	detail, err := loadProjectDetail(projectID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func writeNotImplemented(w http.ResponseWriter, r *http.Request) {
	payload := map[string]string{"error": "endpoint not yet implemented in taskProjectService"}
	if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
		payload["trace_id"] = tid
	}
	writeJSON(w, http.StatusNotImplemented, payload)
}
