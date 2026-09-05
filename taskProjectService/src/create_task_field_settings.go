package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// knownCreateTaskFieldKeys lists optional create-task form fields that can be toggled in workspace settings.
var knownCreateTaskFieldKeys = []string{
	"description",
	"task_kind",
	"code_lang",
	"structured_fields",
	"project_branch",
	"feature_params",
	"priority",
	"due_date",
	"auto_run",
	"operator",
	"owner",
	"assignees",
}

// defaultDisabledCreateTaskFieldKeys: 无工作区配置时默认关闭（主要编程语言、结构化任务说明）。
var defaultDisabledCreateTaskFieldKeys = map[string]bool{
	"code_lang":         true,
	"structured_fields": true,
}

func defaultCreateTaskFieldSettings() map[string]bool {
	out := make(map[string]bool, len(knownCreateTaskFieldKeys))
	for _, k := range knownCreateTaskFieldKeys {
		out[k] = !defaultDisabledCreateTaskFieldKeys[k]
	}
	return out
}

func normalizeCreateTaskFieldSettings(raw map[string]interface{}) map[string]bool {
	out := defaultCreateTaskFieldSettings()
	if raw == nil {
		return out
	}
	for _, k := range knownCreateTaskFieldKeys {
		v, ok := raw[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case bool:
			out[k] = t
		case string:
			s := strings.TrimSpace(strings.ToLower(t))
			if s == "false" || s == "0" || s == "no" || s == "off" {
				out[k] = false
			} else {
				out[k] = true
			}
		case float64:
			out[k] = t != 0
		case int:
			out[k] = t != 0
		case int64:
			out[k] = t != 0
		default:
			out[k] = true
		}
	}
	return out
}

func loadCreateTaskFieldSettings(workspaceID string) (map[string]bool, string, error) {
	var raw, updated string
	err := db.QueryRow(
		`SELECT fields_json, updated_at FROM project_workspace_create_task_field_settings WHERE workspace_id=?`,
		workspaceID,
	).Scan(&raw, &updated)
	if err == sql.ErrNoRows {
		return defaultCreateTaskFieldSettings(), "", nil
	}
	if err != nil {
		return nil, "", err
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return defaultCreateTaskFieldSettings(), updated, nil
	}
	return normalizeCreateTaskFieldSettings(parsed), updated, nil
}

func upsertCreateTaskFieldSettings(workspaceID string, fields map[string]bool) (string, error) {
	now := time.Now().UTC()
	b, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	_, err = db.Exec(
		`INSERT INTO project_workspace_create_task_field_settings (workspace_id, fields_json, updated_at)
		 VALUES (?,?,?)
		 ON DUPLICATE KEY UPDATE
			fields_json=VALUES(fields_json),
			updated_at=VALUES(updated_at)`,
		workspaceID, string(b), now,
	)
	if err != nil {
		return "", err
	}
	return now.Format(time.RFC3339), nil
}

func createTaskFieldSettingsResponse(fields map[string]bool, updatedAt string) map[string]interface{} {
	out := map[string]interface{}{
		"status": "success",
		"fields": fields,
	}
	if updatedAt != "" {
		out["updated_at"] = updatedAt
	}
	return out
}

func handleCreateTaskFieldSettings(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	traceID := r.Header.Get("X-Trace-Id")
	userID := strings.TrimSpace(getAuthUser(r))
	if userID == "" {
		logWarn(fmt.Sprintf("create-task-field-settings unauthorized tenant=%s workspace=%s", tenantID, workspaceID), traceID)
		writeError(w, r, http.StatusUnauthorized, "missing user")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	tenantID = strings.TrimSpace(tenantID)
	if workspaceID == "" || tenantID == "" {
		writeError(w, r, http.StatusBadRequest, "tenant_id and workspace_id required")
		return
	}
	if err := verifyWorkspaceInTenant(workspaceID, tenantID); err != nil {
		logWarn(fmt.Sprintf("create-task-field-settings workspace missing tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
		writeError(w, r, http.StatusNotFound, "workspace not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		fields, updated, err := loadCreateTaskFieldSettings(workspaceID)
		if err != nil {
			logError(fmt.Sprintf("create-task-field-settings GET error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "load failed")
			return
		}
		logInfo(fmt.Sprintf("create-task-field-settings GET ok tenant=%s workspace=%s user=%s", tenantID, workspaceID, userID), traceID)
		writeJSON(w, http.StatusOK, createTaskFieldSettingsResponse(fields, updated))
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid JSON")
			return
		}
		rawFields, ok := body["fields"].(map[string]interface{})
		if !ok {
			writeError(w, r, http.StatusBadRequest, "fields must be an object of booleans")
			return
		}
		norm := normalizeCreateTaskFieldSettings(rawFields)
		updated, err := upsertCreateTaskFieldSettings(workspaceID, norm)
		if err != nil {
			logError(fmt.Sprintf("create-task-field-settings PUT error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "save failed")
			return
		}
		enabled := 0
		for _, v := range norm {
			if v {
				enabled++
			}
		}
		logInfo(fmt.Sprintf("create-task-field-settings PUT ok tenant=%s workspace=%s user=%s enabled=%d/%d", tenantID, workspaceID, userID, enabled, len(norm)), traceID)
		writeJSON(w, http.StatusOK, createTaskFieldSettingsResponse(norm, updated))
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
