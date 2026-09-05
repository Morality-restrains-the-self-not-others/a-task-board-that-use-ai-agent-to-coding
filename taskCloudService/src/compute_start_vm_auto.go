package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tracelog"
)

// Phase 3i slice 3: Go-native finalize (userdata/event build + taskBill); Django only persists ORM records.

func handleStartVmAutoNative(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
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

	if clientIP := resolveStartVmClientPublicIP(r, body); clientIP != "" {
		body["client_public_ip"] = clientIP
	}

	if msg := validateStartVmAutoPayload(body); msg != "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}
	enrichStartVmPayloadFromInstalledImage(tenantID, body)

	taskID := strField(body, "task_id")
	commentID := strField(body, "comment_id")
	if blocked, reason := commentBindingBlocksStartVM(tenantID, taskID, commentID); blocked {
		writeErrorMapJSON(w, r, http.StatusConflict, map[string]interface{}{
			"status": "error", "message": reason,
		})
		return
	}

	// 评论级 CSC 的真实创建由 bootstrapStartVmTokens 承担（模板落真实 platform 后硬失败，
	// OPT-20260815-015）；凭证解析前的 soft-fail ensure 在模板仍为 mock 时必然失败，
	// 只会产出误导日志（ensure_csc_failed 与 start-vm 并行时像被忽略的失败）。

	unlock := lockCommentStartVM(tenantID, taskID, commentID)
	asyncStarted := false
	defer func() {
		if !asyncStarted {
			unlock()
		}
	}()
	ctx, runTraceID := bindStartVmTraceContext(r.Context(), r, taskID)
	if skipDuplicateCommentStartVM(w, r, ctx, tenantID, workspaceID, taskID, commentID) {
		return
	}
	persistCommentBindingStartTraceID(taskID, commentID, runTraceID)
	tracelog.LogForwardStage(ctx, "start_vm_auto_begin", map[string]any{
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
	})
	userID := getAuthUser(r)
	if userID == "" {
		userID = strings.TrimSpace(r.Header.Get("X-User-Id"))
	}
	if userID == "" {
		writeErrorMapJSON(w, r, http.StatusUnauthorized, map[string]interface{}{"status": "error", "message": "未认证用户"})
		return
	}

	cloudAuth, msg := resolveCloudAuthForStartVm(tenantID, body)
	if msg != "" {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, msg))
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}

	if inv := resolveImageInvokerUserID(r, body); inv != "" {
		body["image_invoker_user_id"] = inv
	}
	if handled, gateErr := applyStartVmPolicyGate(w, ctx, tenantID, workspaceID, taskID, cloudAuth, strField(body, "comment_id"), strField(body, "container_image_id")); gateErr != nil {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, gateErr.Error()))
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": gateErr.Error()})
		return
	} else if handled {
		return
	}

	regionID := strField(body, "region_id")
	instanceType := strField(body, "selected_instance")
	if instanceType == "" {
		if hw, ok := body["hardware_config"].(map[string]interface{}); ok {
			instanceType = strField(hw, "instance_type")
		}
	}
	desiredArch := inferInstanceArchitecture(instanceType)
	resolvedImageID, resolvedRegion, imgErr := resolveCloudServerImageID(
		tenantID,
		strField(body, "cloud_server_image_id"),
		strField(body, "container_image_id"),
		cloudAuth.PlatformType,
		regionID,
		desiredArch,
	)
	if imgErr != "" {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, imgErr))
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": imgErr})
		return
	}
	if resolvedImageID == "" {
		msg := "未找到可用的云服务器镜像。请确保该镜像在镜像市场声明了运行环境，且租户已安装对应镜像；或直接传入 cloud_server_image_id。"
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, msg))
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}
	if resolvedRegion != "" {
		body["region_id"] = resolvedRegion
	}
	body["container_image_url"] = strField(body, "container_image_url")

	_ = publishTaskSSE(ctx, taskID, commentID, withStartupImageLogScope(body, map[string]interface{}{
		"status": "processing", "message": "检测到自动创建资源，正在准备前置资源创建任务...",
		"progress": 35, "event_name": "server_status_update",
	}))

	result, statusCode, finErr := finalizeStartVmInGo(ctx, startVmFinalizeInput{
		TenantID:                   tenantID,
		WorkspaceID:                workspaceID,
		UserID:                     userID,
		TraceID:                    runTraceID,
		Body:                       body,
		CloudAuth:                  cloudAuth,
		ResolvedCloudServerImageID: resolvedImageID,
		ResolvedRegionID:           resolvedRegion,
		ContainerImageURL:          strField(body, "container_image_url"),
	})
	if finErr != nil {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, finErr.Error()))
		code := statusCode
		if code < 400 {
			code = http.StatusBadGateway
		}
		writeErrorMapJSON(w, r, code, map[string]interface{}{"status": "error", "message": finErr.Error()})
		return
	}

	via := resolveStartedVia(r, body)
	persistStartedVia(tenantID, taskID, via)
	if via != "queued_schedule" {
		notifyTaskServiceDequeueQueued(tenantID, taskID, via)
	}

	// OPT-20260827-031: HTTP 先 ACK；自动建网 + RunInstances 放到单次 goroutine。
	regionID = strField(result.EventData, "region_id")
	zoneID := strField(result.EventData, "zone_id")
	workCtx := context.WithoutCancel(ctx)
	eventData := result.EventData
	bodyCopy := body
	asyncStarted = true
	scheduleStartVmExecute(func() {
		defer unlock()
		runStartVmAutoAfterAck(workCtx, tenantID, workspaceID, taskID, commentID, regionID, zoneID, cloudAuth, eventData, bodyCopy)
	})

	writeJSON(w, http.StatusOK, startVmAcceptedBody("自动创建资源并启动虚拟机请求已提交", result.EventID, runTraceID))
}
