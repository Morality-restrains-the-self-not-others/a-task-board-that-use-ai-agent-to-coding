package main

import (
	"context"
	"fmt"
)

// startVmExecuteAsync: production true — HTTP 先 ACK，RunInstances 在单次 goroutine
// 里跑（元规则 46：禁止 ticker）。setupCloudTestDB 置 false，保持存量测例同步等到云调用结束。
var startVmExecuteAsync = true

var executeStartVmNativeFn = executeStartVmNative
var executeStartVmAutoNativeFn = executeStartVmAutoNative

func scheduleStartVmExecute(fn func()) {
	if startVmExecuteAsync {
		go fn()
		return
	}
	fn()
}

func startVmAcceptedBody(message, eventID, traceID string) map[string]interface{} {
	return map[string]interface{}{
		"status":   "success",
		"message":  message,
		"event_id": eventID,
		"trace_id": traceID,
	}
}

func runStartVmNativeAfterAck(ctx context.Context, tenantID, workspaceID, taskID, commentID, platformLabel, regionID string, cloudAuth *cloudAuthRecord, eventData map[string]interface{}) {
	_ = publishTaskSSE(ctx, taskID, commentID, map[string]interface{}{
		"status": "processing", "message": fmt.Sprintf("正在调用%sAPI启动服务器...", platformLabel),
		"progress": 70, "event_name": "server_status_update",
	})
	vmResult, vmErr := executeStartVmNativeFn(cloudAuth.SecretID, cloudAuth.SecretKey, regionID, eventData)
	if vmErr != nil {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, vmErr.Error()))
		return
	}
	_ = publishTaskSSE(ctx, taskID, commentID, map[string]interface{}{
		"status": "success", "message": fmt.Sprintf("%s服务器启动成功！", platformLabel),
		"progress": 100, "vm_info": vmResult, "runtime_status": "Running", "event_name": "server_status_update",
	})
}

func runStartVmAutoAfterAck(ctx context.Context, tenantID, workspaceID, taskID, commentID, regionID, zoneID string, cloudAuth *cloudAuthRecord, eventData map[string]interface{}, body map[string]interface{}) {
	_ = publishTaskSSE(ctx, taskID, commentID, withStartupImageLogScope(body, map[string]interface{}{
		"status": "processing", "message": "正在自动创建前置资源并启动服务器...",
		"progress": 45, "event_name": "server_status_update",
	}))
	vmResult, vmErr := executeStartVmAutoNativeFn(cloudAuth.SecretID, cloudAuth.SecretKey, regionID, zoneID, eventData)
	if vmErr != nil {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, vmErr.Error()))
		return
	}
	_ = publishTaskSSE(ctx, taskID, commentID, map[string]interface{}{
		"status": "success", "message": "服务器启动成功！",
		"progress": 100, "vm_info": vmResult, "runtime_status": "Running", "event_name": "server_status_update",
	})
}
