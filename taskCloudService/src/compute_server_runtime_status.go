package main

import (
	"net/http"
	"strings"
	"time"
)

const (
	runtimeMsgMissingCSC             = "未找到服务器配置记录"
	runtimeMsgMissingCSCAfterFail    = "启动失败且未创建评论级服务器配置。请用启动 TraceId 排查后重试。"
	runtimeMsgMissingCSCAfterStart   = "启动已发起，但尚未创建评论级服务器配置。请用启动 TraceId 排查。"
	runtimeMsgEmptyInstanceAfterFail = "云实例启动失败，尚未分配实例。请用启动 TraceId 排查后重试。"
)

func missingCommentCSCRuntime(tenantID, workspaceID, taskID, commentID string) (msg string, startFailed bool) {
	if last := latestCommentBindingServerFailedMessage(workspaceID, tenantID, taskID, commentID); last != "" {
		return humanizeStartVmError(last), true
	}
	b, err := loadCommentContainerBinding(tenantID, taskID, commentID)
	if err != nil || b == nil {
		return runtimeMsgMissingCSC, false
	}
	switch strings.TrimSpace(b.Status) {
	case ccbStatusFailed:
		return runtimeMsgMissingCSCAfterFail, true
	case ccbStatusStarting, ccbStatusWaitingPrevious:
		return runtimeMsgMissingCSCAfterStart, false
	}
	if strings.TrimSpace(b.StartTraceID) != "" {
		return runtimeMsgMissingCSCAfterStart, false
	}
	return runtimeMsgMissingCSC, false
}

func missingCommentCSCRuntimeMessage(tenantID, workspaceID, taskID, commentID string) string {
	msg, _ := missingCommentCSCRuntime(tenantID, workspaceID, taskID, commentID)
	return msg
}

func emptyInstanceRuntimeDecision(cfg *CloudServerConfig, tenantID, workspaceID, taskID string) (runtimeStatus interface{}, message string, startFailed bool) {
	if cfg == nil {
		return nil, "该任务尚未创建云实例", false
	}
	switch strings.ToLower(normalizeMachineRuntimeStatus(cfg.LastRuntimeStatus)) {
	case "released", "terminated":
		return machineRuntimeReleased, "服务器已停止并释放", false
	case "stopped":
		return machineRuntimeStopped, "服务器已停止", false
	case "stopping", "shutting-down", "shuttingdown":
		return machineRuntimeStopping, "服务器停止中", false
	case "failed":
		msg := strings.TrimSpace(cfg.ErrorReason)
		if msg == "" {
			msg = runtimeMsgEmptyInstanceAfterFail
		}
		return nil, humanizeStartVmError(msg), true
	}
	commentID := strings.TrimSpace(cfg.CommentID)
	if commentID != "" {
		ws := strings.TrimSpace(workspaceID)
		if ws == "" {
			ws = strings.TrimSpace(cfg.WorkspaceID)
		}
		if last := latestCommentBindingServerFailedMessage(ws, tenantID, taskID, commentID); last != "" {
			return nil, humanizeStartVmError(last), true
		}
		b, err := loadCommentContainerBinding(tenantID, taskID, commentID)
		if err == nil && b != nil && strings.TrimSpace(b.Status) == ccbStatusFailed {
			msg := strings.TrimSpace(cfg.ErrorReason)
			if msg == "" {
				msg = runtimeMsgEmptyInstanceAfterFail
			}
			return nil, humanizeStartVmError(msg), true
		}
	}
	// 评论级 CSC / 启动中绑定：与「服务器启动状态=启动中」对齐，勿报「未找到」或空「未创建」
	provisioning := commentID != "" || commentBindingIsProvisioning(tenantID, taskID)
	if provisioning {
		return machineRuntimeStarting, "云实例创建中，等待分配", false
	}
	return nil, "该任务尚未创建云实例", false
}

func attachServerStartedAt(resp map[string]interface{}, tenantID, workspaceID, taskID string) {
	if resp == nil {
		return
	}
	open, err := loadOpenCloudServerConfigHistory(tenantID, workspaceID, taskID)
	if err != nil || open == nil || open.StartedAt.IsZero() {
		return
	}
	resp["server_started_at"] = open.StartedAt.UTC().Format(time.RFC3339)
}

func writeServerRuntimeStatusJSON(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string, cfg *CloudServerConfig, resp map[string]interface{}) {
	if resp == nil {
		resp = map[string]interface{}{}
	}
	attachServerStartedAt(resp, tenantID, workspaceID, taskID)
	attachMachineOwnerHint(resp, tenantID, workspaceID, taskID, cfg)
	// 成功/降级响应也注入 trace_id，供 FE data-traceId 与 Loki 排障（含 auth_missing 文案）
	if r != nil {
		if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
			resp["trace_id"] = tid
			w.Header().Set("X-Trace-Id", tid)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleServerRuntimeStatus(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "缺少任务ID",
		})
		return
	}
	if tenantID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "无法获取租户ID",
		})
		return
	}
	commentID := commentIDFromComputeRequest(r, nil)
	if rejectMissingComputeCommentID(w, r, commentID) {
		return
	}
	cfg, err := resolveScopedCloudServerConfig(tenantID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		msg, startFailed := missingCommentCSCRuntime(tenantID, workspaceID, taskID, commentID)
		if msg != runtimeMsgMissingCSC {
			logWarn("event=server_runtime_missing_comment_csc task_id="+taskID+" comment_id="+commentID+" message="+msg, taskID)
		}
		resp := map[string]interface{}{
			"status": "success", "runtime_status": nil, "message": msg,
		}
		if startFailed {
			resp["start_failed"] = true
		}
		writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, nil, resp)
		return
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		cfg = healCommentCSCInstanceFromCloud(cfg, tenantID, workspaceID, taskID)
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		runtimeStatus, message, startFailed := emptyInstanceRuntimeDecision(cfg, tenantID, workspaceID, taskID)
		resp := map[string]interface{}{
			"status":         "success",
			"runtime_status": runtimeStatus,
			"message":        message,
			"instance_id":    nil,
			"csc_id":         cfg.ID,
			"comment_id":     cfg.CommentID,
			"platform":       cfg.Platform,
			"region":         cfg.Region,
		}
		if runtimeStatus == machineRuntimeReleased {
			resp["released"] = true
		}
		if startFailed {
			resp["start_failed"] = true
			logWarn("event=server_runtime_empty_instance_start_failed task_id="+taskID+" comment_id="+strings.TrimSpace(cfg.CommentID)+" csc_id="+cfg.ID+" message="+message, taskID)
		}
		writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, cfg, resp)
		return
	}
	instanceID := strings.TrimSpace(cfg.InstanceID)
	if strings.HasPrefix(instanceID, "mock-") {
		applyObservedMachineRuntimeStatus(tenantID, workspaceID, taskID, instanceID, machineRuntimeRunning, true)
		writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, cfg, map[string]interface{}{
			"status": "success", "runtime_status": "Running", "instance_id": instanceID,
			"platform": cfg.Platform, "region": cfg.Region, "mock": true,
			"message": "Mock 实例（本地 Docker 容器），运行状态为模拟值",
		})
		return
	}
	auth, err := resolveCloudAuthForRuntimeDescribe(tenantID, cfg)
	if err != nil || auth == nil {
		writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, cfg, buildAuthMissingRuntimeFallback(cfg, instanceID))
		return
	}
	attr, found, requestID, err := describeInstanceForRuntime(auth.SecretID, auth.SecretKey, cfg.Region, instanceID)
	if err != nil {
		if code := aliyunErrorCode(err); code == "InvalidInstanceId.NotFound" || strings.Contains(code, "InvalidInstanceId") {
			wsID := strings.TrimSpace(workspaceID)
			if wsID == "" {
				wsID = strings.TrimSpace(cfg.WorkspaceID)
			}
			applyObservedMachineRuntimeStatus(tenantID, wsID, taskID, instanceID, machineRuntimeReleased, false)
			_ = publishContainerTaskUIContextSSE(tenantID, wsID, taskID)
			writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, cfg, map[string]interface{}{
				"status": "success", "runtime_status": "Released", "released": true,
				"message":     "该实例已被释放（云平台返回实例不存在）",
				"instance_id": instanceID, "platform": cfg.Platform, "region": cfg.Region,
				"error_code": code,
			})
			return
		}
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	if !found {
		wsID := strings.TrimSpace(workspaceID)
		if wsID == "" {
			wsID = strings.TrimSpace(cfg.WorkspaceID)
		}
		applyObservedMachineRuntimeStatus(tenantID, wsID, taskID, instanceID, machineRuntimeReleased, false)
		_ = publishContainerTaskUIContextSSE(tenantID, wsID, taskID)
		if requestID != "" {
			w.Header().Set("X-Cloud-Request-Id-DescribeInstances", requestID)
		}
		writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, cfg, map[string]interface{}{
			"status": "success", "runtime_status": "Released", "released": true,
			"message":     "该实例已被释放（云平台返回实例不存在）",
			"instance_id": instanceID, "platform": cfg.Platform, "region": cfg.Region,
			"error_code": "InvalidInstanceId.NotFound",
		})
		return
	}
	runtimeStatus, _ := attr["Status"].(string)
	applyObservedMachineRuntimeStatus(tenantID, workspaceID, taskID, instanceID, runtimeStatus, true)
	// 缓存本次 Describe 的带宽字段，供 auth_missing 回退继续展示流量带宽。
	_ = persistInstanceBandwidth(instanceID, bandwidthChargeTypeFromAttr(attr), bandwidthOutFromAttr(attr))
	resp := map[string]interface{}{
		"status": "success", "runtime_status": runtimeStatus,
		"instance_id": instanceID, "platform": cfg.Platform, "region": cfg.Region,
		"instance_attribute": map[string]interface{}{"body": attr},
	}
	if autoRelease, ok := attr["AutoReleaseTime"].(string); ok && strings.TrimSpace(autoRelease) != "" {
		resp["auto_release_time"] = strings.TrimSpace(autoRelease)
	}
	if requestID != "" {
		w.Header().Set("X-Cloud-Request-Id-DescribeInstances", requestID)
	}
	writeServerRuntimeStatusJSON(w, r, tenantID, workspaceID, taskID, cfg, resp)
}
