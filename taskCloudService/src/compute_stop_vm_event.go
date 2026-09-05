package main

import (
	"snowflake"
	"strings"
)

func buildStopVmEventData(tenantID, workspaceID, taskID string, cfg *CloudServerConfig, stopReason string) map[string]interface{} {
	wsID := strings.TrimSpace(workspaceID)
	if wsID == "" && cfg != nil {
		wsID = strings.TrimSpace(cfg.WorkspaceID)
	}
	region := ""
	authID := ""
	platform := "aliyun"
	instanceID := ""
	commentID := ""
	if cfg != nil {
		region = strings.TrimSpace(cfg.Region)
		authID = strings.TrimSpace(cfg.AuthorizationID)
		platform = strings.TrimSpace(cfg.Platform)
		instanceID = strings.TrimSpace(cfg.InstanceID)
		commentID = strings.TrimSpace(cfg.CommentID)
		platform = resolveStopEventPlatform(platform, instanceID, "")
	}
	reason := strings.TrimSpace(stopReason)
	if reason == "" {
		reason = "user_stop"
	}
	return map[string]interface{}{
		"task_id":             taskID,
		"tenant_id":           tenantID,
		"company_id":          tenantID,
		"workspace_id":        wsID,
		"comment_id":          commentID,
		"instance_id":         instanceID,
		"region_id":           region,
		"authorization_id":    authID,
		"cloud_platform_type": platform,
		"stop_reason":         reason,
		"stop_reason_label":   stopReasonTriggerLabel(reason),
		// Unique per publish so consumer idempotency does not collapse stops by company_id/task_id.
		"stop_request_id": snowflake.GenerateIDString(),
	}
}

func resolveStopEventPlatform(cfgPlatform, instanceID, authPlatform string) string {
	if p := strings.TrimSpace(authPlatform); p != "" && !isLocalSkipCloudPlatform(p) {
		return p
	}
	p := strings.TrimSpace(cfgPlatform)
	if p == "" {
		return "aliyun"
	}
	if isLocalSkipCloudPlatform(p) && strings.HasPrefix(strings.TrimSpace(instanceID), "i-") {
		return "aliyun"
	}
	return p
}

func isLocalSkipCloudPlatform(platform string) bool {
	p := strings.ToLower(strings.TrimSpace(platform))
	return p == "mock" || p == "relay-local" || strings.HasPrefix(p, "relay")
}
