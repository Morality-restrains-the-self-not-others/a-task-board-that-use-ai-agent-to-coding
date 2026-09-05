package main

import (
	"context"
	"log"
	"strings"
	"time"
)

// clearContainerReachabilityNative closes runtime sessions and clears reachability in Go SQLite (SSOT).
// When instanceID is non-empty, only histories for that instance are closed, and CSC binding is
// cleared only if it currently points at the same instance (avoids wiping a newer binding).
func clearContainerReachabilityNative(tenantID, workspaceID, taskID, reason, instanceID string) (bool, error) {
	tenantID = strings.TrimSpace(tenantID)
	taskID = strings.TrimSpace(taskID)
	workspaceID = strings.TrimSpace(workspaceID)
	instanceID = strings.TrimSpace(instanceID)
	if tenantID == "" || taskID == "" {
		return false, nil
	}

	var cfg *CloudServerConfig
	if instanceID != "" {
		cfg, _ = loadCloudServerConfigByInstanceForTask(tenantID, workspaceID, taskID, instanceID)
	} else {
		cfg, _ = loadNewestCloudServerConfigForTask(tenantID, workspaceID, taskID)
	}

	wsID := workspaceID
	if wsID == "" && cfg != nil {
		wsID = strings.TrimSpace(cfg.WorkspaceID)
	}
	stopReason := strings.TrimSpace(reason)
	if stopReason == "" {
		stopReason = "relay_stop"
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	patch := map[string]interface{}{
		"stopped_at":  now,
		"stop_reason": stopReason,
	}
	if cfg != nil {
		patch["server_url"] = cfg.ServerURL
		patch["business_api_endpoint"] = cfg.BusinessAPIEndpoint
		patch["public_ip"] = cfg.PublicIP
	}
	if instanceID != "" {
		patch["public_ip"] = ""
		patch["server_url"] = ""
		patch["business_api_endpoint"] = ""
	}
	_, _ = closeOpenCloudServerConfigHistories(tenantID, wsID, taskID, patch, instanceID)

	if cfg == nil {
		if instanceID != "" {
			_ = publishContainerTaskUIContextSSE(tenantID, wsID, taskID)
			return true, nil
		}
		return false, nil
	}

	cfgInstance := strings.TrimSpace(cfg.InstanceID)
	shouldClearCSC := instanceID == "" || cfgInstance == "" || cfgInstance == instanceID
	if !shouldClearCSC {
		_ = publishContainerTaskUIContextSSE(tenantID, wsID, taskID)
		return true, nil
	}

	return clearContainerReachabilityOnConfig(cfg, tenantID, wsID, taskID)
}

func clearContainerReachabilityOnConfig(cfg *CloudServerConfig, tenantID, workspaceID, taskID string) (bool, error) {
	if cfg == nil {
		return false, nil
	}
	cfg.InstanceID = ""
	cfg.PublicIP = ""
	cfg.ServerURL = ""
	cfg.BusinessAPIEndpoint = ""
	cfg.ContainerVscodeURL = ""
	cfg.ErrorReason = ""
	cfg.LastRuntimeStatus = machineRuntimeReleased
	cfg.UpdatedAt = time.Now().UTC()
	if err := upsertCloudServerConfig(*cfg); err != nil {
		return false, err
	}
	releaseCommentContainerBindingOnConfig(cfg, tenantID, taskID)
	_ = clearConfigIdleSince(tenantID, workspaceID, taskID)
	_ = setCloudServerLastRuntimeStatus(tenantID, workspaceID, taskID, cfg.CommentID, cfg.InstanceID, machineRuntimeReleased)
	_ = publishContainerTaskUIContextSSE(tenantID, workspaceID, taskID)
	return true, nil
}

func publishContainerTaskUIContextSSE(tenantID, workspaceID, taskID string) error {
	ctx := buildContainerTaskUIContext(tenantID, workspaceID, taskID)
	statusData := map[string]interface{}{
		"status":     "container_task_ui_context",
		"event_name": "server_status_update",
	}
	if ctx != nil {
		if v, ok := ctx["has_server_config"]; ok {
			statusData["has_server_config"] = v
		}
		if v, ok := ctx["container_endpoint_registered"]; ok {
			statusData["container_endpoint_registered"] = v
		}
		if v, ok := ctx["container_page_url"]; ok {
			statusData["container_page_url"] = v
		}
		if v, ok := ctx["container_vscode_url"]; ok {
			statusData["container_vscode_url"] = v
		}
	}
	return publishSSEMessage(context.Background(), taskID, statusData)
}

// releaseCommentContainerBindingOnConfig marks the CSC comment's starting/running
// binding as released. Parallel comments on the same task are left untouched.
func releaseCommentContainerBindingOnConfig(cfg *CloudServerConfig, tenantID, taskID string) {
	if cfg == nil {
		return
	}
	commentID := strings.TrimSpace(cfg.CommentID)
	if commentID == "" {
		return
	}
	b, err := loadCommentContainerBinding(tenantID, taskID, commentID)
	if err != nil || b == nil {
		return
	}
	switch b.Status {
	case ccbStatusStarting, ccbStatusRunning:
	default:
		return
	}
	if err := updateCommentContainerBindingStatus(b.ID, ccbStatusReleased); err != nil {
		log.Printf("[taskCloudService] event=comment_container_binding_release_failed binding_id=%s comment_id=%s err=%v",
			b.ID, commentID, err)
		return
	}
	b.Status = ccbStatusReleased
	log.Printf("[taskCloudService] event=comment_container_binding_released binding_id=%s comment_id=%s task_id=%s",
		b.ID, commentID, taskID)
	logCommentContainerBindingStageBestEffort(b, ccbStatusReleased)
}
