package main

import (
	"fmt"
	"net/http"
	"strings"
)

const cloudServerConfigSelectCols = `id, company_id, workspace_id, task_id, COALESCE(comment_id,''), platform, instance_id,
		COALESCE(security_group_id,''), COALESCE(vswitch_id,''), region, zone_id,
		authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(container_vscode_url,''), COALESCE(error_reason,''), COALESCE(launch_request_id,''), client_token,
		COALESCE(last_runtime_status,''), COALESCE(image_invoker_user_id,''),
		COALESCE(verification_secret,''), COALESCE(userdata_run_verified,''),
		created_at, updated_at`

func listCommentCloudServerConfigsForTask(companyID, workspaceID, taskID string) ([]CloudServerConfig, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if taskID == "" || db == nil {
		return nil, nil
	}
	q := `SELECT ` + cloudServerConfigSelectCols + `
		FROM cloud_server_configs
		WHERE task_id=? AND TRIM(COALESCE(comment_id,'')) != ''`
	args := []interface{}{taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CloudServerConfig, 0)
	for rows.Next() {
		cfg, scanErr := scanCloudServerConfigFields(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *cfg)
	}
	return out, rows.Err()
}

func handleInternalListByTask(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
	if taskID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "task_id required")
		return
	}
	comments, err := listCommentCloudServerConfigsForTask(tenantID, workspaceID, taskID)
	if err != nil {
		logError("event=list_by_task_query_failed task_id="+taskID+" err="+err.Error(), traceID)
		writeErrorJSON(w, r, http.StatusInternalServerError, "list comment configs failed")
		return
	}
	machines, containers := countsFromCommentConfigs(comments)
	items := make([]map[string]interface{}, 0, len(comments))
	for i := range comments {
		items = append(items, cloudServerConfigToJSON(&comments[i]))
	}
	logInfo(fmt.Sprintf("event=list_by_task task_id=%s comments=%d machines=%d containers=%d",
		taskID, len(items), machines, containers), traceID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id":                 taskID,
		"running_machine_count":   machines,
		"running_container_count": containers,
		"comments":                items,
	})
}

func countsFromCommentConfigs(comments []CloudServerConfig) (machines, containers int) {
	for i := range comments {
		c := &comments[i]
		if machineRuntimeCountsAsStarted(c.InstanceID, c.LastRuntimeStatus) {
			machines++
		}
		if strings.TrimSpace(c.ServerURL) != "" {
			containers++
		}
	}
	return
}
