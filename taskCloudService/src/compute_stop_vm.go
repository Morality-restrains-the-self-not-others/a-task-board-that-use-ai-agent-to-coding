package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Phase 3i: Go-native stop-vm — release ECS via Aliyun SDK, clear reachability, no Django proxy.

func handleStopVmNative(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if tenantID == "" || workspaceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "缺少租户或工作区上下文",
		})
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "无法读取请求体"})
		return
	}
	var body map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "JSON 格式错误"})
			return
		}
	}
	if body == nil {
		body = map[string]interface{}{}
	}

	taskID := strField(body, "task_id")
	if taskID == "" {
		taskID = strings.TrimSpace(r.Header.Get("X-Task-Id"))
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少任务ID"})
		return
	}

	commentID := commentIDFromComputeRequest(r, body)
	if rejectMissingComputeCommentID(w, r, commentID) {
		return
	}

	stopReason := stopReasonFromComputeBody(body)

	publishStopSSE := func(statusData map[string]interface{}) {
		if statusData["event_name"] == "" {
			statusData["event_name"] = "server_status_update"
		}
		if msg := strField(statusData, "message"); isCloudServerStopSchedulingMessage(msg) {
			statusData["message"] = annotateStopServerMessage(msg, stopReason)
		}
		statusData["stop_reason"] = stopReason
		statusData["stop_reason_label"] = stopReasonTriggerLabel(stopReason)
		_ = publishTaskSSE(r.Context(), taskID, commentID, statusData)
	}

	publishStopSSE(map[string]interface{}{
		"status": "initializing", "message": "正在初始化服务器停止...", "progress": 0,
	})

	cfg, err := resolveScopedCloudServerConfig(tenantID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		msg := "未找到任务对应的服务器配置"
		publishStopSSE(map[string]interface{}{"status": "error", "message": msg, "progress": 0})
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		msg := "未提供实例ID"
		publishStopSSE(map[string]interface{}{"status": "error", "message": msg, "progress": 0})
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}

	wsID := strings.TrimSpace(workspaceID)
	if wsID == "" {
		wsID = strings.TrimSpace(cfg.WorkspaceID)
	}
	if isLocalSkipCloudPlatform(cfg.Platform) && !strings.HasPrefix(strings.TrimSpace(cfg.InstanceID), "i-") {
		eventData := buildStopVmEventData(tenantID, wsID, taskID, cfg, stopReason)
		if err := publishDomainEvent(r.Context(), "CLOUD_SERVER_STOPPED", eventData, taskID); err != nil {
			logInfo("kafka CLOUD_SERVER_STOPPED publish failed: "+err.Error(), r.Header.Get("X-Trace-Id"))
		}
		publishStopSSE(map[string]interface{}{
			"status": "processing", "message": "正在停止 Mock 实例...", "progress": 50,
			"runtime_status": "Stopping",
		})
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "虚拟机停止请求已提交",
			"vm_info": map[string]interface{}{
				"instance_id": cfg.InstanceID,
				"region":      strings.TrimSpace(cfg.Region),
			},
		})
		return
	}

	authBody := map[string]interface{}{
		"authorization_id":  cfg.AuthorizationID,
		"cloud_platform_id": "1",
	}
	if strings.TrimSpace(cfg.Platform) != "" && !strings.EqualFold(cfg.Platform, "aliyun") {
		authBody["cloud_platform_id"] = ""
	}
	cloudAuth, msg := resolveCloudAuthForStartVm(tenantID, authBody)
	if msg != "" {
		publishStopSSE(map[string]interface{}{"status": "error", "message": msg, "progress": 0})
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}
	if cloudAuth == nil {
		msg := "未配置云平台授权"
		publishStopSSE(map[string]interface{}{"status": "error", "message": msg, "progress": 0})
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}

	platformLabel := strings.TrimSpace(cloudAuth.PlatformType)
	if platformLabel == "" {
		platformLabel = "云"
	}
	if !strings.EqualFold(cloudAuth.PlatformType, "aliyun") {
		msg := fmt.Sprintf("当前平台 %s 的 stop-vm 尚未在 Go 服务中实现", cloudAuth.PlatformType)
		publishStopSSE(map[string]interface{}{"status": "error", "message": msg, "progress": 0})
		writeErrorMapJSON(w, r, http.StatusNotImplemented, map[string]interface{}{"status": "error", "message": msg})
		return
	}

	publishStopSSE(map[string]interface{}{
		"status": "processing", "message": fmt.Sprintf("正在准备停止%s服务器...", platformLabel), "progress": 30,
	})

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "cn-hongkong"
	}

	eventData := buildStopVmEventData(tenantID, wsID, taskID, cfg, stopReason)
	if plat := strings.TrimSpace(cloudAuth.PlatformType); plat != "" {
		eventData["cloud_platform_type"] = resolveStopEventPlatform(
			fmt.Sprint(eventData["cloud_platform_type"]), cfg.InstanceID, plat)
	}
	if err := publishDomainEvent(r.Context(), "CLOUD_SERVER_STOPPED", eventData, taskID); err != nil {
		logInfo("kafka CLOUD_SERVER_STOPPED publish failed: "+err.Error(), r.Header.Get("X-Trace-Id"))
	}

	publishStopSSE(map[string]interface{}{
		"status": "processing", "message": fmt.Sprintf("正在调用%sAPI停止服务器...", platformLabel), "progress": 50,
		"runtime_status": "Stopping",
	})

	resp := map[string]interface{}{
		"status":  "success",
		"message": "虚拟机停止请求已提交",
		"vm_info": map[string]interface{}{
			"instance_id": cfg.InstanceID,
			"region":      region,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func clearContainerReachabilityInGo(ctx context.Context, tenantID, workspaceID, taskID, reason, traceID string) error {
	_ = ctx
	_ = traceID
	_, err := clearContainerReachabilityNative(tenantID, workspaceID, taskID, reason, "")
	return err
}

func patchCloudServerConfigAfterStop(tenantID, workspaceID, taskID string) error {
	_, err := clearContainerReachabilityNative(tenantID, workspaceID, taskID, "stop_vm", "")
	return err
}
