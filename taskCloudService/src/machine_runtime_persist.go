package main

import (
	"strings"
	"time"
)

func setCloudServerLastRuntimeStatus(companyID, workspaceID, taskID, commentID, instanceID, status string) error {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	instanceID = trim(instanceID)
	status = normalizeMachineRuntimeStatus(status)
	if companyID == "" || taskID == "" {
		return nil
	}
	if commentID == "" && instanceID == "" {
		logWarn("event=last_runtime_status_unscoped_skipped task_id="+taskID, taskID)
		return nil
	}
	now := time.Now().UTC()
	q := `UPDATE cloud_server_configs SET last_runtime_status=?, updated_at=? WHERE company_id=? AND task_id=? AND TRIM(COALESCE(comment_id,''))!=''`
	args := []interface{}{status, now, companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	switch {
	case instanceID != "":
		q += ` AND instance_id=?`
		args = append(args, instanceID)
	default:
		q += ` AND comment_id=?`
		args = append(args, commentID)
	}
	_, err := db.Exec(q, args...)
	if err == nil {
		recomputeTaskRunningCounts(companyID, workspaceID, taskID)
	}
	return err
}

// clearReleasedMachineBinding clears stale local machine binding after cloud reports Released / not found.
func clearReleasedMachineBinding(tenantID, workspaceID, taskID, instanceID string) {
	tenantID = trim(tenantID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if tenantID == "" || taskID == "" {
		return
	}
	_, _ = clearContainerReachabilityNative(tenantID, workspaceID, taskID, "runtime_released", instanceID)
	_ = setCloudServerLastRuntimeStatus(tenantID, workspaceID, taskID, "", instanceID, machineRuntimeReleased)
}

// applyObservedMachineRuntimeStatus persists polled cloud status and clears binding when released.
func applyObservedMachineRuntimeStatus(tenantID, workspaceID, taskID, instanceID, runtimeStatus string, found bool) {
	tenantID = trim(tenantID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if tenantID == "" || taskID == "" {
		return
	}
	if isMockMachineInstanceID(instanceID) {
		_ = setCloudServerLastRuntimeStatus(tenantID, workspaceID, taskID, "", instanceID, machineRuntimeRunning)
		return
	}
	if machineRuntimeIsReleasedOrMissing(runtimeStatus, found) {
		clearReleasedMachineBinding(tenantID, workspaceID, taskID, instanceID)
		return
	}
	status := normalizeMachineRuntimeStatus(runtimeStatus)
	if status == "" {
		return
	}
	_ = setCloudServerLastRuntimeStatus(tenantID, workspaceID, taskID, "", instanceID, status)
}

func effectiveMachineRuntimeStatus(instanceID, lastStatus string) string {
	instanceID = strings.TrimSpace(instanceID)
	lastStatus = normalizeMachineRuntimeStatus(lastStatus)
	if isMockMachineInstanceID(instanceID) {
		return machineRuntimeRunning
	}
	return lastStatus
}
