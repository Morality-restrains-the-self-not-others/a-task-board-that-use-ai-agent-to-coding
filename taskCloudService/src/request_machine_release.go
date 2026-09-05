package main

import (
	"context"
	"fmt"
	"net/http"
	"snowflake"
	"strings"
)

// busyBindingCount counts CSC rows on the same instance that still look busy
// (non-empty server_url and not already terminal_released).
func busyBindingCount(bindings []instanceBindingRow, excludeReleased bool) int {
	n := 0
	for _, b := range bindings {
		if excludeReleased && b.TerminalReleased != 0 {
			continue
		}
		if strings.TrimSpace(b.ServerURL) != "" {
			n++
		}
	}
	return n
}

// soleOrEmptyBusy returns true when the instance has at most this task's busy
// container (or no busy containers at all).
func soleOrEmptyBusy(bindings []instanceBindingRow, selfTaskID string) bool {
	selfTaskID = strings.TrimSpace(selfTaskID)
	busyOthers := 0
	for _, b := range bindings {
		if b.TerminalReleased != 0 {
			continue
		}
		if strings.TrimSpace(b.ServerURL) == "" {
			continue
		}
		if strings.TrimSpace(b.TaskID) == selfTaskID {
			continue
		}
		busyOthers++
	}
	return busyOthers == 0
}

type machineReleaseResult struct {
	Released      bool
	AlreadyDone   bool
	NoConfig      bool
	DestroyedNode bool
	BusyCount     int
	TerminalKind  string
	Reason        string
}

func handleRequestMachineRelease(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	terminalKind := strings.TrimSpace(fmt.Sprintf("%v", body["terminal_kind"]))
	reason := strings.TrimSpace(fmt.Sprintf("%v", body["reason"]))
	res, err := releaseMachineForTerminal(ctx, cfgRow, tenantID, workspaceID, taskID, terminalKind, reason)
	if err != nil {
		writeErrorJSON(w, nil, http.StatusInternalServerError, err.Error())
		return
	}
	if res.NoConfig {
		writeErrorMapJSON(w, nil, http.StatusOK, map[string]interface{}{
			"ok": true, "status": "ok", "released": false, "detail": "no_config",
		})
		return
	}
	if res.AlreadyDone {
		writeErrorMapJSON(w, nil, http.StatusOK, map[string]interface{}{
			"ok": true, "status": "ok", "released": true, "detail": "already_released",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":             true,
		"status":         "ok",
		"released":       res.Released,
		"destroyed_node": res.DestroyedNode,
		"busy_bindings":  res.BusyCount,
		"terminal_kind":  res.TerminalKind,
	})
}

// releaseMachineForTerminal publishes CLOUD_SERVER_STOPPED (when this task is the
// sole busy binding) then marks CSC terminal_released. Used by inbound 410
// compensation when TASK_STATUS_CHANGED was DLT'd while cloud was down.
func releaseMachineForTerminal(ctx context.Context, cfgRow *CloudServerConfig, tenantID, workspaceID, taskID, terminalKind, reason string) (machineReleaseResult, error) {
	res := machineReleaseResult{}
	terminalKind = strings.TrimSpace(strings.ToLower(terminalKind))
	if terminalKind == "" || terminalKind == "<nil>" {
		terminalKind = "cancelled"
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || reason == "<nil>" {
		reason = "task_status_" + terminalKind
	}
	res.TerminalKind = terminalKind
	res.Reason = reason

	if cfgRow == nil {
		res.NoConfig = true
		return res, nil
	}

	if cfgRow.TerminalReleasedFlag() == 1 && strings.TrimSpace(cfgRow.InstanceID) == "" && strings.TrimSpace(cfgRow.ServerURL) == "" {
		res.AlreadyDone = true
		res.Released = true
		return res, nil
	}

	instanceID := strings.TrimSpace(cfgRow.InstanceID)
	companyID := strings.TrimSpace(cfgRow.CompanyID)
	if companyID == "" {
		companyID = strings.TrimSpace(tenantID)
	}
	wsID := strings.TrimSpace(cfgRow.WorkspaceID)
	if wsID == "" {
		wsID = strings.TrimSpace(workspaceID)
	}

	shouldDestroyNode := true
	busyCount := 0
	if instanceID != "" {
		bindings, err := listInstanceBindings(companyID, wsID, instanceID)
		if err != nil {
			return res, err
		}
		busyCount = busyBindingCount(bindings, true)
		shouldDestroyNode = soleOrEmptyBusy(bindings, taskID)
	}
	res.BusyCount = busyCount
	res.DestroyedNode = shouldDestroyNode

	logInfo(fmt.Sprintf(
		"event=request_machine_release task_id=%s terminal_kind=%s instance_id=%s busy=%d destroy=%v reason=%s",
		taskID, terminalKind, instanceID, busyCount, shouldDestroyNode, reason,
	), taskID)

	if shouldDestroyNode && instanceID != "" {
		eventData := map[string]interface{}{
			"task_id":             taskID,
			"tenant_id":           tenantID,
			"company_id":          companyID,
			"workspace_id":        wsID,
			"instance_id":         instanceID,
			"region_id":           strings.TrimSpace(cfgRow.Region),
			"authorization_id":    strings.TrimSpace(cfgRow.AuthorizationID),
			"cloud_platform_type": resolveStopEventPlatform(cfgRow.Platform, instanceID, ""),
			"stop_reason":         reason,
			"stop_reason_label":   stopReasonTriggerLabel(reason),
			"stop_request_id":     snowflake.GenerateIDString(),
		}
		if cid := strings.TrimSpace(cfgRow.CommentID); cid != "" {
			eventData["comment_id"] = cid
		}
		if err := publishDomainEvent(ctx, "CLOUD_SERVER_STOPPED", eventData, taskID); err != nil {
			return res, fmt.Errorf("publish CLOUD_SERVER_STOPPED failed: %w", err)
		}
	}

	if err := markTerminalReleased(companyID, wsID, taskID); err != nil {
		return res, err
	}

	platformLabel := strings.TrimSpace(cfgRow.Platform)
	if platformLabel == "" || (isLocalSkipCloudPlatform(platformLabel) && strings.HasPrefix(instanceID, "i-")) {
		platformLabel = "aliyun"
	}
	if platformLabel == "" {
		platformLabel = "云"
	}
	stopLine := annotateStopServerMessage(
		fmt.Sprintf("正在调用%sAPI停止服务器...", platformLabel), reason)
	commentID := strings.TrimSpace(cfgRow.CommentID)
	msg := annotateStopServerMessage("容器请求释放机器节点", reason)
	status := "released"
	progress := 100
	runtimeStatus := ""
	if shouldDestroyNode && instanceID != "" {
		msg = stopLine
		status = "processing"
		progress = 50
		runtimeStatus = "Stopping"
	}
	_ = publishTaskSSE(ctx, taskID, commentID, map[string]interface{}{
		"status":            status,
		"event_name":        "server_status_update",
		"message":           msg,
		"progress":          progress,
		"stop_reason":       reason,
		"stop_reason_label": stopReasonTriggerLabel(reason),
		"destroyed":         shouldDestroyNode,
		"terminal_kind":     terminalKind,
		"runtime_status":    runtimeStatus,
	})
	res.Released = true
	return res, nil
}

// TerminalReleasedFlag reads this CSC row's terminal_released.
// Must be comment-scoped (id or task+comment). A sibling/task-level Released 行
// 不得把「进行中」任务的新实例入站判成 cancelled。
func (c *CloudServerConfig) TerminalReleasedFlag() int {
	if c == nil || db == nil {
		return 0
	}
	var flag int
	var err error
	if id := strings.TrimSpace(c.ID); id != "" {
		err = db.QueryRow(`SELECT COALESCE(terminal_released,0) FROM cloud_server_configs WHERE id=?`, id).Scan(&flag)
	} else {
		companyID := strings.TrimSpace(c.CompanyID)
		workspaceID := strings.TrimSpace(c.WorkspaceID)
		taskID := strings.TrimSpace(c.TaskID)
		commentID := strings.TrimSpace(c.CommentID)
		if companyID == "" || workspaceID == "" || taskID == "" || commentID == "" {
			return 0
		}
		err = db.QueryRow(
			`SELECT COALESCE(terminal_released,0) FROM cloud_server_configs WHERE company_id=? AND workspace_id=? AND task_id=? AND comment_id=?`,
			companyID, workspaceID, taskID, commentID,
		).Scan(&flag)
	}
	if err != nil {
		return 0
	}
	return flag
}
