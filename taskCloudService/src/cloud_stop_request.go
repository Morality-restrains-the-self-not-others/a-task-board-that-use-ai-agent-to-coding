package main

import (
	"net/http"
	"strings"
)

// cloud_stop_request 幂等表（dataMigrate/taskCloudService/028）：
// CLOUD_SERVER_STOPPED 的 stop_request_id 落库为唯一键，作为 owner 侧最终去重。
// 消费者在处理 DeleteInstance 前先查 processed 以跳过重放；clear-after-stop 成功后 best-effort 记录。

func recordCloudStopRequest(stopRequestID, tenantID, workspaceID, taskID, instanceID string) error {
	stopRequestID = trim(stopRequestID)
	if stopRequestID == "" {
		return nil
	}
	_, err := db.Exec(
		`INSERT IGNORE INTO cloud_stop_request(stop_request_id, tenant_id, workspace_id, task_id, instance_id)
		 VALUES (?, ?, ?, ?, ?)`,
		stopRequestID, trim(tenantID), trim(workspaceID), trim(taskID), trim(instanceID),
	)
	return err
}

func cloudStopRequestProcessed(stopRequestID string) (bool, error) {
	stopRequestID = trim(stopRequestID)
	if stopRequestID == "" {
		return false, nil
	}
	var one int
	err := db.QueryRow(`SELECT 1 FROM cloud_stop_request WHERE stop_request_id = ? LIMIT 1`, stopRequestID).Scan(&one)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func handleInternalStopRequestRoutes(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/cloud/stops/")
	path = strings.Trim(path, "/")
	switch {
	case path == "processed" && r.Method == http.MethodGet:
		handleInternalStopRequestProcessed(w, r)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func handleInternalStopRequestProcessed(w http.ResponseWriter, r *http.Request) {
	stopRequestID := strings.TrimSpace(r.URL.Query().Get("stop_request_id"))
	if stopRequestID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "stop_request_id required"})
		return
	}
	processed, err := cloudStopRequestProcessed(stopRequestID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "processed": processed})
}
