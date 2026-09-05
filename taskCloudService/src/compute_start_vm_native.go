package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Phase 3i slice 4: Go-native start-vm (existing VPC/VSwitch/SG) — mirrors start-vm-auto but publishes CLOUD_SERVER_STARTED.

func handleStartVmNative(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
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

	if msg := validateStartVmDirectPayload(body); msg != "" {
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

	requestedRegion := strField(body, "region_id")
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
		requestedRegion,
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
	if requestedRegion != "" && resolvedRegion != "" && requestedRegion != resolvedRegion {
		msg := fmt.Sprintf(
			"请求的地域(%s)与镜像可用地域(%s)不匹配，请选择正确的地域或镜像。",
			requestedRegion, resolvedRegion,
		)
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, msg))
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": msg})
		return
	}
	if resolvedRegion != "" {
		body["region_id"] = resolvedRegion
	}
	body["container_image_url"] = strField(body, "container_image_url")

	platformLabel := strings.TrimSpace(cloudAuth.PlatformType)
	if platformLabel == "" {
		platformLabel = "云"
	}
	_ = publishTaskSSE(ctx, taskID, commentID, map[string]interface{}{
		"status": "processing", "message": fmt.Sprintf("正在准备启动%s服务器...", platformLabel),
		"progress": 50, "event_name": "server_status_update",
	})

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

	// OPT-20260827-031: HTTP 先 ACK；RunInstances 在 goroutine（或测试同步路径）完成。
	regionID := strField(result.EventData, "region_id")
	workCtx := context.WithoutCancel(ctx)
	eventData := result.EventData
	asyncStarted = true
	scheduleStartVmExecute(func() {
		defer unlock()
		runStartVmNativeAfterAck(workCtx, tenantID, workspaceID, taskID, commentID, platformLabel, regionID, cloudAuth, eventData)
	})

	writeJSON(w, http.StatusOK, startVmAcceptedBody("启动虚拟机请求已提交", result.EventID, runTraceID))
}
