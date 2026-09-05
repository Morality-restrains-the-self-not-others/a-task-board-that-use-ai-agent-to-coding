package main

import (
	"fmt"
	"net/http"
	"strings"
)

type cloudDeletionBlocker struct {
	Code       string `json:"code"`
	Blocking   bool   `json:"blocking"`
	Message    string `json:"message"`
	ActionURL  string `json:"action_url,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	TenantName string `json:"tenant_name,omitempty"`
}

func handleInternalCloudUsersRouter(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/cloud/users/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || (parts[1] != "account-deletion-blockers" && parts[1] != "personal-data") || r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
		return
	}
	userID := strings.TrimSpace(parts[0])
	if userID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	if parts[1] == "personal-data" {
		handleInternalCloudPersonalData(w, r, userID)
		return
	}
	blockers, err := collectCloudDeletionBlockers(userID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"blockers": blockers})
}

func collectCloudDeletionBlockers(userID string) ([]cloudDeletionBlocker, error) {
	rows, err := db.Query(`
		SELECT id, company_id, workspace_id, task_id, COALESCE(comment_id,''),
		       COALESCE(instance_id,''), COALESCE(last_runtime_status,''), COALESCE(server_url,'')
		FROM cloud_server_configs
		WHERE image_invoker_user_id = ? AND TRIM(COALESCE(comment_id,'')) != ''`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var blockers []cloudDeletionBlocker
	for rows.Next() {
		var id, tenantID, workspaceID, taskID, commentID, instanceID, runtimeStatus, serverURL string
		if err := rows.Scan(&id, &tenantID, &workspaceID, &taskID, &commentID, &instanceID, &runtimeStatus, &serverURL); err != nil {
			continue
		}
		machineRunning := machineRuntimeCountsAsStarted(instanceID, runtimeStatus)
		containerRunning := strings.TrimSpace(serverURL) != ""
		if !machineRunning && !containerRunning {
			continue
		}
		actionURL := cloudResourceActionURL(tenantID, workspaceID, taskID)
		code := "CLOUD_RESOURCE_RUNNING"
		msg := "仍有运行中的云资源，请先停止并释放"
		if machineRunning && containerRunning {
			msg = fmt.Sprintf("任务 %s 评论 %s 仍有运行中的机器与容器", taskID, commentID)
		} else if machineRunning {
			code = "CLOUD_MACHINE_RUNNING"
			msg = fmt.Sprintf("任务 %s 评论 %s 仍有运行中的云主机", taskID, commentID)
		} else {
			code = "CLOUD_CONTAINER_RUNNING"
			msg = fmt.Sprintf("任务 %s 评论 %s 仍有运行中的容器", taskID, commentID)
		}
		blockers = append(blockers, cloudDeletionBlocker{
			Code: code, Blocking: true, Message: msg, ActionURL: actionURL, TenantID: tenantID,
		})
	}
	return blockers, nil
}
