package main

import (
	"net/http"
)

func handleInternalClearAfterStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	instanceID := strField(body, "instance_id")
	stopRequestID := strField(body, "stop_request_id")
	reason := strField(body, "stop_reason")
	if reason == "" {
		reason = strField(body, "reason")
	}
	if tenantID == "" || taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "tenant_id and task_id required"})
		return
	}
	ok, err := clearContainerReachabilityNative(tenantID, workspaceID, taskID, reason, instanceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	// OPT-20260818-015：CLOUD_SERVER_STOPPED 成功后把 stop_request_id 落库为唯一键，
	// 重放事件可据此跳过（消费者侧在 DeleteInstance 前先查 processed）。best-effort。
	if stopRequestID != "" {
		_ = recordCloudStopRequest(stopRequestID, tenantID, workspaceID, taskID, instanceID)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "cleared": ok})
}
