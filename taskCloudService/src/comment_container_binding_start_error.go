package main

import (
	"log"
	"strings"
	"sync"
)

// start-vm 错误常早于 binding INSERT。旁路暂存错误 SSE，INSERT 后 drain 收口 failed，
// 避免卡片卡在 starting、runtime-status 误报「启动已发起」。
var pendingBindingStartError sync.Map

type pendingStartError struct {
	StatusData map[string]interface{}
}

func stashPendingBindingStartError(taskID, commentID string, statusData map[string]interface{}) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" || statusData == nil {
		return
	}
	if strings.ToLower(strField(statusData, "status")) != "error" {
		return
	}
	if strings.TrimSpace(strField(statusData, "message")) == "" {
		return
	}
	copied := copyStatusDataMap(statusData)
	pendingBindingStartError.Store(pendingBindingStartTraceKey(taskID, commentID), pendingStartError{StatusData: copied})
	logInfo("event=binding_start_error_pending_no_row task_id="+taskID+" comment_id="+commentID,
		strField(copied, "trace_id"))
}

func loadPendingBindingStartError(taskID, commentID string) map[string]interface{} {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return nil
	}
	raw, ok := pendingBindingStartError.Load(pendingBindingStartTraceKey(taskID, commentID))
	if !ok {
		return nil
	}
	pe, ok := raw.(pendingStartError)
	if !ok || pe.StatusData == nil {
		return nil
	}
	return copyStatusDataMap(pe.StatusData)
}

func deletePendingBindingStartError(taskID, commentID string) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return
	}
	pendingBindingStartError.Delete(pendingBindingStartTraceKey(taskID, commentID))
}

func drainPendingBindingStartError(companyID, taskID, commentID string) {
	statusData := loadPendingBindingStartError(taskID, commentID)
	if statusData == nil {
		return
	}
	companyID = trim(companyID)
	if companyID == "" {
		companyID = strField(statusData, "tenant_id")
		if companyID == "" {
			companyID = strField(statusData, "company_id")
		}
	}
	if companyID == "" {
		return
	}
	b, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || b == nil {
		return
	}
	persistServerSchedulingLogForBinding(b, statusData)
	markCommentBindingFailedAfterStartError(b)
	deletePendingBindingStartError(taskID, commentID)
	log.Printf("[taskCloudService] event=binding_start_error_drained task_id=%s comment_id=%s binding_id=%s",
		taskID, commentID, b.ID)
}

func copyStatusDataMap(src map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func startVmErrorStatusData(tenantID, workspaceID, msg string) map[string]interface{} {
	if h := humanizeStartVmError(msg); h != "" {
		msg = h
	}
	return map[string]interface{}{
		"status":       "error",
		"message":      msg,
		"progress":     0,
		"event_name":   "server_status_update",
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
	}
}

func failBindingFromStartVmErrorSSE(taskID, commentID string, statusData map[string]interface{}) {
	companyID := strField(statusData, "tenant_id")
	if companyID == "" {
		companyID = strField(statusData, "company_id")
	}
	if companyID == "" {
		stashPendingBindingStartError(taskID, commentID, statusData)
		return
	}
	b, _, err := ensureCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", strField(statusData, "workspace_id"))
	if err != nil || b == nil {
		log.Printf("[taskCloudService] event=binding_start_error_ensure_failed task_id=%s comment_id=%s err=%v",
			taskID, commentID, err)
		stashPendingBindingStartError(taskID, commentID, statusData)
		return
	}
	persistServerSchedulingLogForBinding(b, statusData)
	markCommentBindingFailedAfterStartError(b)
	deletePendingBindingStartError(taskID, commentID)
}

func latestCommentBindingServerFailedMessage(workspaceID, companyID, taskID, commentID string) string {
	commentID = trim(commentID)
	if commentID == "" {
		return ""
	}
	logs, err := listCommentContainerBindingLogsIn(workspaceID, companyID, taskID)
	if err != nil {
		return ""
	}
	msg := ""
	for i := range logs {
		if logs[i].CommentID != commentID || logs[i].Stage != ccbStageServerFailed {
			continue
		}
		if m := strings.TrimSpace(logs[i].Message); m != "" {
			msg = m
		}
	}
	return stripTraceIDLogSuffix(msg)
}

func stripTraceIDLogSuffix(msg string) string {
	msg = strings.TrimSpace(msg)
	if i := strings.LastIndex(msg, " trace_id="); i >= 0 {
		return strings.TrimSpace(msg[:i])
	}
	return msg
}
