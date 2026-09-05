package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"tracelog"
)

// allowedRuntimeEventNames: bootstrap / auto_run 关键阶段，供 Loki 检索。
var allowedRuntimeEventNames = map[string]struct{}{
	"BOOTSTRAP_PHASE":                      {},
	"BOOTSTRAP_COMPLETE":                   {},
	"BOOTSTRAP_FAILED":                     {},
	"AUTO_RUN_FIRST_INSTRUCTION_START":     {},
	"AUTO_RUN_FIRST_INSTRUCTION_STARTED":   {},
	"AUTO_RUN_FIRST_SKIP":                  {},
	"AUTO_RUN_FIRST_INSTRUCTION_FAILED":    {},
	"AT_MENTION_JOB_START":                 {},
	"AT_MENTION_JOB_SKIP":                  {},
	"AT_MENTION_JOB_FAILED":                {},
	"AUTO_RUN_DELIVERY_BEGIN":              {},
	"AUTO_RUN_DELIVERY_COMPLETE":           {},
	"AUTO_RUN_DELIVERY_FAILED":             {},
	"AUTO_RUN_DELIVERY_SKIP":               {},
	"AUTO_RUN_PR_BACKFILL_OK":              {},
	"AUTO_RUN_PR_BACKFILL_FAILED":          {},
	"EDIT_RUN_AGENT_COMMENT_CREATED":       {},
	"EDIT_RUN_AGENT_COMMENT_CREATE_FAILED": {},
}

// handleRuntimeEvent 接收容器侧 BOOTSTRAP_*/AUTO_RUN_* 关键事件，写入结构化日志（进 Loki）。
// 常规 PHASE 不转发 SSE（避免刷屏）；BOOTSTRAP_FAILED / BOOTSTRAP_COMPLETE 推送任务详情，
// 避免「双向已连接」但任务关联区无限「等待可写层」且无失败原因。
func handleRuntimeEvent(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	event := strings.TrimSpace(fmt.Sprintf("%v", body["event"]))
	if event == "" || event == "<nil>" {
		writeErrorJSON(w, nil, http.StatusBadRequest, "event 必填")
		return
	}
	if _, ok := allowedRuntimeEventNames[event]; !ok {
		writeErrorJSON(w, nil, http.StatusBadRequest, "unsupported event: "+event)
		return
	}
	phase := strings.TrimSpace(fmt.Sprintf("%v", body["phase"]))
	if phase == "<nil>" {
		phase = ""
	}
	message := strings.TrimSpace(fmt.Sprintf("%v", body["message"]))
	if message == "<nil>" {
		message = ""
	}
	if len(message) > 800 {
		message = message[:800]
	}
	level := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", body["level"])))
	switch level {
	case "debug", "info", "warn", "error":
	default:
		level = "info"
	}
	attrs := map[string]any{
		"event":        event,
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"level":        level,
	}
	if phase != "" {
		attrs["phase"] = phase
	}
	if message != "" {
		attrs["detail"] = message
	}
	if cfgRow != nil && strings.TrimSpace(cfgRow.TaskID) != "" {
		attrs["task_id"] = strings.TrimSpace(cfgRow.TaskID)
	}
	if extra, ok := body["fields"].(map[string]any); ok && extra != nil {
		for k, v := range extra {
			key := strings.TrimSpace(k)
			if key == "" || key == "event" || key == "access_token" {
				continue
			}
			if len(attrs) >= 24 {
				break
			}
			attrs[key] = v
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	tracelog.LogForwardStage(ctx, "container_runtime_event", attrs)
	if event == "BOOTSTRAP_COMPLETE" && cfgRow != nil {
		if err := markInstructionIdle(ctx, cfgRow); err != nil {
			logInfo("event=instruction_idle_mark status=error source=bootstrap cfg="+cfgRow.ID+" err="+err.Error(), cfgRow.TaskID)
		}
	}
	publishBootstrapRuntimeOutcomeSSE(ctx, cfgRow, taskID, event, phase, message, firstNonEmptyTraceID(ctx, body))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID, "event": event})
}

// bootstrapRuntimeSSEStatus maps runtime-event names to task-detail SSE status.
// Empty means do not publish (avoid PHASE spam). clone_begin / task_detail_begin
// are forwarded so the task-association banner is not stuck on generic「等待可写层」
// while clone has not produced a layer snapshot yet.
func bootstrapRuntimeSSEStatus(event, phase string) string {
	switch strings.TrimSpace(event) {
	case "BOOTSTRAP_FAILED":
		return "container_bootstrap_failed"
	case "BOOTSTRAP_COMPLETE":
		return "container_bootstrap_complete"
	case "BOOTSTRAP_PHASE":
		switch strings.TrimSpace(phase) {
		case "clone_begin", "task_detail_begin", "credentials_recovery_begin":
			return "container_bootstrap_progress"
		default:
			return ""
		}
	default:
		return ""
	}
}

func mapTraceString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", m[key]))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

// firstNonEmptyTraceID prefers body/fields trace from the container, then inbound HTTP ctx.
func firstNonEmptyTraceID(ctx context.Context, body map[string]any) string {
	if s := mapTraceString(body, "trace_id"); s != "" {
		return s
	}
	if s := mapTraceString(body, "traceId"); s != "" {
		return s
	}
	if extra, ok := body["fields"].(map[string]any); ok {
		if s := mapTraceString(extra, "trace_id"); s != "" {
			return s
		}
	}
	return tracelog.TraceIDFromContext(ctx)
}

// publishBootstrapRuntimeOutcomeSSE 将引导终态推给任务详情「任务关联」区。
// 使用 publishSSEMessage（非整条 binding 调度日志），避免无 DB 测试路径 panic，且引导失败属任务级。
func publishBootstrapRuntimeOutcomeSSE(ctx context.Context, cfgRow *CloudServerConfig, taskID, event, phase, message, traceID string) {
	status := bootstrapRuntimeSSEStatus(event, phase)
	if status == "" {
		return
	}
	tid := strings.TrimSpace(taskID)
	if cfgRow != nil {
		if t := strings.TrimSpace(cfgRow.TaskID); t != "" {
			tid = t
		}
	}
	if tid == "" {
		return
	}
	statusData := map[string]interface{}{
		"status":     status,
		"event_name": "server_status_update",
		"phase":      phase,
		"message":    message,
		"event":      event,
	}
	if t := strings.TrimSpace(traceID); t != "" {
		statusData["trace_id"] = t
	}
	if cfgRow != nil {
		if cid := strings.TrimSpace(cfgRow.CommentID); cid != "" {
			statusData["comment_id"] = cid
		}
	}
	_ = publishSSEMessage(ctx, tid, statusData)
}
