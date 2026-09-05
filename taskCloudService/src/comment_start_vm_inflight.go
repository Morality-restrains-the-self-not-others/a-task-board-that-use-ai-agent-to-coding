package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"tracelog"
)

// commentStartVMMutex 串行同一评论的 start-vm / start-vm-auto，避免 @镜像 mention
// 与 binding bootstrap 两条 HTTP 并发 RunInstances（ClientToken 含 userdata digest，
// 阿里云不视作同一请求 → 同名双实例）。
type commentStartVMMutex struct {
	mu sync.Mutex
}

var commentStartVMGates sync.Map // string → *commentStartVMMutex

func commentStartVMGateKey(companyID, taskID, commentID string) string {
	return trim(companyID) + "|" + trim(taskID) + "|" + trim(commentID)
}

func commentStartVMGate(companyID, taskID, commentID string) *commentStartVMMutex {
	key := commentStartVMGateKey(companyID, taskID, commentID)
	actual, _ := commentStartVMGates.LoadOrStore(key, &commentStartVMMutex{})
	return actual.(*commentStartVMMutex)
}

// lockCommentStartVM 在 finalize / RunInstances 全程持锁。comment_id 为空不加锁。
// 禁止在 bootstrap 出站 HTTP 上持这把锁（会死锁：bootstrap 持锁 → POST start-vm → handler 再抢同一把锁）。
func lockCommentStartVM(companyID, taskID, commentID string) func() {
	if trim(commentID) == "" {
		return func() {}
	}
	g := commentStartVMGate(companyID, taskID, commentID)
	g.mu.Lock()
	return func() { g.mu.Unlock() }
}

// commentStartVMBusy 窥探闸门是否被占用。TryLock 成功后立即 Unlock，不持有。
func commentStartVMBusy(companyID, taskID, commentID string) bool {
	if trim(commentID) == "" {
		return false
	}
	g := commentStartVMGate(companyID, taskID, commentID)
	if !g.mu.TryLock() {
		return true
	}
	g.mu.Unlock()
	return false
}

// skipDuplicateCommentStartVM 同一评论已有非 mock 且 Starting/Running 的实例时幂等 200。
// 调用方须已持 lockCommentStartVM。不写 start_trace_id（禁止覆盖 mention 已持久化的 TraceId）。
func skipDuplicateCommentStartVM(w http.ResponseWriter, r *http.Request, ctx context.Context, tenantID, workspaceID, taskID, commentID string) bool {
	commentID = trim(commentID)
	if commentID == "" {
		return false
	}
	attached, err := tryAttachSameTaskInFlightMachine(tenantID, workspaceID, taskID, commentID, "")
	if err != nil {
		logWarn(fmt.Sprintf("event=start_vm_duplicate_check_failed task_id=%s comment_id=%s err=%v",
			taskID, commentID, err), tracelog.TraceIDFromContext(ctx))
		return false
	}
	if !attached.Reused {
		return false
	}
	writeCommentStartVMDuplicateSkipped(w, r, ctx, taskID, commentID, attached)
	return true
}

func writeCommentStartVMDuplicateSkipped(w http.ResponseWriter, r *http.Request, ctx context.Context, taskID, commentID string, attached idleReuseResult) {
	traceID := tracelog.TraceIDFromContext(ctx)
	if traceID == "" && r != nil {
		traceID = stringsTrimTraceHeader(r)
	}
	logInfo(fmt.Sprintf("event=start_vm_duplicate_skipped task_id=%s comment_id=%s instance_id=%s",
		taskID, commentID, attached.InstanceID), traceID)
	statusData := map[string]interface{}{
		"status": "success", "message": "评论级云主机已在启动或运行中，跳过重复建机",
		"progress": 100, "event_name": "server_status_update",
		"reuse": true, "duplicate_skipped": true, "inflight_attach": true,
		"instance_id": attached.InstanceID,
	}
	if trim(commentID) != "" {
		statusData["comment_id"] = commentID
	}
	if traceID != "" {
		statusData["trace_id"] = traceID
	}
	// 走 publishSSEMessage 而非 publishTaskSSE：后者会 persistCommentBindingStartTraceID 覆盖已有 TraceId。
	_ = publishSSEMessage(ctx, taskID, statusData)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":            "success",
		"reuse":             true,
		"duplicate_skipped": true,
		"inflight_attach":   true,
		"instance_id":       attached.InstanceID,
		"public_ip":         attached.PublicIP,
		"message":           "评论级云主机已在启动或运行中，跳过重复建机",
		"trace_id":          traceID,
	})
}

func stringsTrimTraceHeader(r *http.Request) string {
	if r == nil {
		return ""
	}
	return trim(r.Header.Get(tracelog.Header))
}

// commentCSCBootstrapShouldSkipStart 在 mention 正在 start-vm 或实例已绑定时跳过自调。
// 不因「任意 pending start 事件」跳过：mention 失败可能留下 pending，bootstrap 仍须能建机。
func commentCSCBootstrapShouldSkipStart(csc *CloudServerConfig) (reason, instanceID string, skip bool) {
	if csc == nil {
		return "", "", false
	}
	commentID := trim(csc.CommentID)
	if commentID == "" {
		return "", "", false
	}
	if commentStartVMBusy(csc.CompanyID, csc.TaskID, commentID) {
		return "start_inflight", "", true
	}
	attached, err := tryAttachSameTaskInFlightMachine(csc.CompanyID, csc.WorkspaceID, csc.TaskID, commentID, "")
	if err != nil {
		logWarn(fmt.Sprintf("event=comment_csc_bootstrap_duplicate_check_failed task_id=%s comment_id=%s err=%v",
			csc.TaskID, commentID, err), "")
		return "", "", false
	}
	if attached.Reused {
		return "instance_bound", attached.InstanceID, true
	}
	return "", "", false
}
