package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultIdleRecycleMinutes = 5
)

type WorkspaceMachinePolicy struct {
	CompanyID               string
	WorkspaceID             string
	IdleRecycleMinutes      int
	EnabledAuthorizationIDs []string
	UpdatedAt               time.Time
}

func defaultWorkspaceMachinePolicy(companyID, workspaceID string) WorkspaceMachinePolicy {
	return WorkspaceMachinePolicy{
		CompanyID:               companyID,
		WorkspaceID:             workspaceID,
		IdleRecycleMinutes:      defaultIdleRecycleMinutes,
		EnabledAuthorizationIDs: []string{},
	}
}

func loadWorkspaceMachinePolicy(companyID, workspaceID string) (WorkspaceMachinePolicy, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	if companyID == "" || workspaceID == "" {
		return defaultWorkspaceMachinePolicy(companyID, workspaceID), nil
	}
	row := db.QueryRow(
		`SELECT idle_recycle_minutes, enabled_authorization_ids, updated_at
		 FROM cloud_workspace_machine_policies WHERE company_id=? AND workspace_id=?`,
		companyID, workspaceID,
	)
	var minutes int
	var authJSON, updated string
	err := row.Scan(&minutes, &authJSON, &updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return defaultWorkspaceMachinePolicy(companyID, workspaceID), nil
		}
		return WorkspaceMachinePolicy{}, err
	}
	p := defaultWorkspaceMachinePolicy(companyID, workspaceID)
	p.IdleRecycleMinutes = minutes
	p.EnabledAuthorizationIDs = parseAuthorizationIDsJSON(authJSON)
	if updated != "" {
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	}
	return p, nil
}

func parseAuthorizationIDsJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []string{}
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []string{}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func upsertWorkspaceMachinePolicy(p WorkspaceMachinePolicy) error {
	now := time.Now().UTC()
	authJSON, err := json.Marshal(p.EnabledAuthorizationIDs)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO cloud_workspace_machine_policies
			(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids, updated_at)
		 VALUES (?,?,?,?,?)
		 ON DUPLICATE KEY UPDATE
			idle_recycle_minutes=VALUES(idle_recycle_minutes),
			enabled_authorization_ids=VALUES(enabled_authorization_ids),
			updated_at=VALUES(updated_at)`,
		p.CompanyID, p.WorkspaceID, p.IdleRecycleMinutes, string(authJSON), now,
	)
	return err
}

func workspaceMachinePolicyToJSON(p WorkspaceMachinePolicy) map[string]interface{} {
	return map[string]interface{}{
		"status":                    "success",
		"idle_recycle_minutes":      p.IdleRecycleMinutes,
		"enabled_authorization_ids": p.EnabledAuthorizationIDs,
	}
}

func handleWorkspaceMachinePolicy(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	switch r.Method {
	case http.MethodGet:
		p, err := loadWorkspaceMachinePolicy(tenantID, workspaceID)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
				"status": "error", "message": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, workspaceMachinePolicyToJSON(p))
	case http.MethodPut:
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
				"status": "error", "message": "无法读取请求体",
			})
			return
		}
		var body map[string]interface{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
					"status": "error", "message": "JSON 格式错误",
				})
				return
			}
		}
		if body == nil {
			body = map[string]interface{}{}
		}
		cur, err := loadWorkspaceMachinePolicy(tenantID, workspaceID)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
				"status": "error", "message": err.Error(),
			})
			return
		}
		if _, ok := body["idle_recycle_minutes"]; ok {
			minutes := intField(body, "idle_recycle_minutes", cur.IdleRecycleMinutes)
			if minutes < 0 {
				writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
					"status": "error", "message": "idle_recycle_minutes 不能为负数",
				})
				return
			}
			cur.IdleRecycleMinutes = minutes
		}
		if v, ok := body["enabled_authorization_ids"]; ok {
			cur.EnabledAuthorizationIDs = parseAuthorizationIDsFromBody(v)
		}
		cur.CompanyID = tenantID
		cur.WorkspaceID = workspaceID
		if err := upsertWorkspaceMachinePolicy(cur); err != nil {
			writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
				"status": "error", "message": err.Error(),
			})
			return
		}
		logInfo("workspace machine policy updated tenant="+tenantID+" workspace="+workspaceID, r.Header.Get("X-Trace-Id"))
		writeJSON(w, http.StatusOK, workspaceMachinePolicyToJSON(cur))
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func parseAuthorizationIDsFromBody(v interface{}) []string {
	switch arr := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			id := strings.TrimSpace(fmt.Sprintf("%v", item))
			if id != "" && id != "<nil>" {
				out = append(out, id)
			}
		}
		return out
	case []string:
		return arr
	default:
		return []string{}
	}
}

func authorizationAllowedByPolicy(policy WorkspaceMachinePolicy, authorizationID string) (bool, string) {
	authorizationID = strings.TrimSpace(authorizationID)
	if len(policy.EnabledAuthorizationIDs) == 0 {
		return true, ""
	}
	for _, id := range policy.EnabledAuthorizationIDs {
		if id == authorizationID {
			return true, ""
		}
	}
	return false, "该云平台授权未在工作空间的启用机器节点列表中，无法启动虚拟机"
}
