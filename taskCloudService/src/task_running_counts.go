package main

import (
	"fmt"
	"time"

	"taskCloudService/domain"
)

func classifyCommentRuntimes(companyID, workspaceID, taskID string) []domain.CommentRuntime {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" || db == nil {
		return nil
	}
	q := `SELECT COALESCE(comment_id,''), COALESCE(instance_id,''), COALESCE(server_url,''), COALESCE(last_runtime_status,'')
		 FROM cloud_server_configs WHERE company_id=? AND task_id=?`
	args := []interface{}{companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		logWarn("event=task_running_counts_query_failed task_id="+taskID+" err="+err.Error(), taskID)
		return nil
	}
	defer rows.Close()
	var out []domain.CommentRuntime
	for rows.Next() {
		var commentID, instanceID, serverURL, lastStatus string
		if err := rows.Scan(&commentID, &instanceID, &serverURL, &lastStatus); err != nil {
			logWarn("event=task_running_counts_scan_failed task_id="+taskID+" err="+err.Error(), taskID)
			return nil
		}
		status := effectiveMachineRuntimeStatus(instanceID, lastStatus)
		out = append(out, domain.CommentRuntime{
			CommentID:          commentID,
			MachineStarted:     machineRuntimeCountsAsStarted(instanceID, status),
			ContainerReachable: containerReachabilityCountsAsRunning(instanceID, status, serverURL),
		})
	}
	if err := rows.Err(); err != nil {
		logWarn("event=task_running_counts_rows_failed task_id="+taskID+" err="+err.Error(), taskID)
		return nil
	}
	return out
}

func loadTaskRunningCounts(companyID, workspaceID, taskID string) domain.RunningCounts {
	return domain.ComputeRunningCounts(classifyCommentRuntimes(companyID, workspaceID, taskID))
}

func recomputeTaskRunningCounts(companyID, workspaceID, taskID string) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" || db == nil {
		return
	}
	counts := loadTaskRunningCounts(companyID, workspaceID, taskID)
	q := `UPDATE cloud_server_configs SET running_machine_count=?, running_container_count=?, updated_at=?
		 WHERE company_id=? AND task_id=? AND TRIM(COALESCE(comment_id,''))=''`
	args := []interface{}{counts.Machines, counts.Containers, time.Now().UTC(), companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if _, err := db.Exec(q, args...); err != nil {
		logWarn("event=task_running_counts_write_failed task_id="+taskID+" err="+err.Error(), taskID)
		return
	}
	logInfo(fmt.Sprintf("event=task_running_counts_updated task_id=%s machines=%d containers=%d",
		taskID, counts.Machines, counts.Containers), taskID)
}

func recomputeTaskRunningCountsForCSC(cscID, taskID, commentID string) {
	if db == nil {
		return
	}
	var companyID, workspaceID, resolvedTask string
	q := `SELECT company_id, workspace_id, task_id FROM cloud_server_configs WHERE id=?`
	args := []interface{}{trim(cscID)}
	if trim(cscID) == "" {
		q = `SELECT company_id, workspace_id, task_id FROM cloud_server_configs WHERE task_id=? AND comment_id=? LIMIT 1`
		args = []interface{}{trim(taskID), trim(commentID)}
	}
	if err := db.QueryRow(q, args...).Scan(&companyID, &workspaceID, &resolvedTask); err != nil {
		if trim(taskID) != "" {
			logWarn("event=task_running_counts_lookup_failed task_id="+taskID+" err="+err.Error(), taskID)
		}
		return
	}
	recomputeTaskRunningCounts(companyID, workspaceID, resolvedTask)
}
