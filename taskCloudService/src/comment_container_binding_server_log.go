package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// appendCommentContainerBindingLogMessage 追加自由文案启动日志（服务器调度进度等）。
// message 截断至 512（列宽）；写入失败由调用方 best-effort 处理。
func appendCommentContainerBindingLogMessage(b *CommentContainerBinding, stage, message string) error {
	if b == nil {
		return nil
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		return nil
	}
	if len(msg) > 512 {
		msg = msg[:509] + "…"
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = ccbStageServerScheduling
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	ws := bindingLogWorkspaceID(b)
	if ws == "" {
		ws = resolveLogWorkspaceID("", b.TaskID, nil)
	}
	return insertCCBLogRow(ws, b.CompanyID, b.TaskID, b.CommentID, b.ID, stage, msg, now)
}

func loadCommentContainerBindingByTaskComment(taskID, commentID string) (*CommentContainerBinding, error) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return nil, fmt.Errorf("task_id, comment_id required")
	}
	row := db.QueryRow(
		`SELECT `+commentContainerBindingSelectColumns+`
		FROM cloud_comment_container_bindings
		WHERE task_id=? AND comment_id=?
		ORDER BY updated_at DESC, id DESC
		LIMIT 1`,
		taskID, commentID,
	)
	return scanCommentContainerBinding(row)
}

func isCloudServerStopSchedulingMessage(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	if strings.Contains(msg, "停止虚拟机成功") || strings.Contains(msg, "Mock 实例已停止") {
		return true
	}
	if strings.Contains(msg, "正在停止 Mock 实例") {
		return true
	}
	if strings.Contains(msg, "正在初始化服务器停止") {
		return true
	}
	if strings.Contains(msg, "正在准备停止") && strings.Contains(msg, "服务器") {
		return true
	}
	if strings.Contains(msg, "正在调用") && strings.Contains(msg, "停止服务器") {
		return true
	}
	return false
}

func shouldPersistServerSchedulingLog(statusData map[string]interface{}) bool {
	if strField(statusData, "message") == "" {
		return false
	}
	if strField(statusData, "event_name") == "server_status_update" {
		return true
	}
	switch strings.ToLower(strField(statusData, "status")) {
	case "processing", "initializing", "starting", "success", "error", "sdk_call", "released":
		return true
	default:
		return false
	}
}

func serverSchedulingLogMessage(statusData map[string]interface{}) string {
	msg := strings.TrimSpace(strField(statusData, "message"))
	if msg == "" {
		return ""
	}
	// 旧容器镜像的克隆失败文案不内嵌 URL，但 body 携带 repo_url；
	// 在此把 URL 插到「失败 <name>」之后（冒号前），与 onlineServiceJS
	// formatBootstrapCloneRepoFailureMessage 的 `<name> <url>:` 格式对齐，
	// 冷打开从 binding 日志即可还原「手动重试」身份（512 截断前完成）。
	msg = insertRepoURLAfterFailedRepo(msg, strField(statusData, "repo_url"))
	tid := strings.TrimSpace(strField(statusData, "trace_id"))
	if tid == "" {
		return msg
	}
	// 冷打开还原启动 TraceId：日志正文携带可解析后缀
	if strings.Contains(msg, "trace_id=") {
		return msg
	}
	return msg + " trace_id=" + tid
}

func isGitCloneRepoURL(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "git@")
}

// insertRepoURLAfterFailedRepo 把 git URL 插到「失败 <name>」之后（冒号前）。
// 消息已含该 URL（新容器文案已内嵌）或非「失败 <name>:」形态时不改写，避免重复/误插。
func insertRepoURLAfterFailedRepo(msg, repoURL string) string {
	msg = strings.TrimSpace(msg)
	repoURL = strings.TrimSpace(repoURL)
	if !isGitCloneRepoURL(repoURL) || msg == "" {
		return msg
	}
	if strings.Contains(msg, repoURL) {
		return msg
	}
	const prefix = "失败 "
	idx := strings.Index(msg, prefix)
	if idx < 0 {
		return msg
	}
	after := msg[idx+len(prefix):]
	colon := strings.IndexByte(after, ':')
	if colon <= 0 {
		return msg
	}
	insertAt := idx + len(prefix) + colon
	return msg[:insertAt] + " " + repoURL + msg[insertAt:]
}

func persistServerSchedulingLogForBinding(b *CommentContainerBinding, statusData map[string]interface{}) {
	if b == nil || !shouldPersistServerSchedulingLog(statusData) {
		return
	}
	if ws := strField(statusData, "workspace_id"); ws != "" {
		b.WorkspaceID = ws
	}
	stage := ccbStageServerScheduling
	msg := serverSchedulingLogMessage(statusData)
	stopMsg := isCloudServerStopSchedulingMessage(msg)
	switch strings.ToLower(strField(statusData, "status")) {
	case "success":
		if !stopMsg {
			stage = ccbStageServerStarted
		}
	case "error":
		stage = ccbStageServerFailed
		msg = humanizeStartVmError(msg)
	}
	if err := appendCommentContainerBindingLogMessage(b, stage, msg); err != nil {
		log.Printf("[taskCloudService] event=comment_container_binding_server_log_failed binding_id=%s stage=%s err=%v",
			b.ID, stage, err)
	}
	if stage == ccbStageServerFailed {
		persistCommentCSCStartFailure(b, msg)
	}
	if stage == ccbStageServerStarted {
		recoverFailedCommentBindingToStarting(b, "comment_scoped_start_success")
	}
}

// logServerSchedulingToBindingBestEffort 将服务器调度 SSE 文案写入评论 binding 启动日志，
// 供冷打开还原；Kafka/DB 失败均不阻断 SSE 主路径。
func logServerSchedulingToBindingBestEffort(taskID, commentID string, statusData map[string]interface{}) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" || !shouldPersistServerSchedulingLog(statusData) {
		return
	}
	b, err := loadCommentContainerBindingByTaskComment(taskID, commentID)
	if err != nil || b == nil {
		if strings.ToLower(strField(statusData, "status")) == "error" {
			failBindingFromStartVmErrorSSE(taskID, commentID, statusData)
		}
		return
	}
	persistServerSchedulingLogForBinding(b, statusData)
	if strings.ToLower(strField(statusData, "status")) == "error" {
		markCommentBindingFailedAfterStartError(b)
	}
}

func isCommentIDMissingSchedulingMessage(statusData map[string]interface{}) bool {
	return strings.TrimSpace(strField(statusData, "message")) == "缺少评论ID"
}

// logServerSchedulingToActiveBindingsBestEffort：任务级 start-vm（无 comment_id）时，
// 将调度错误/进度写入本任务全部非终态 binding，便于面板还原真实失败原因与 startTraceId。
// 「缺少评论ID」是校验错误而非调度进度，禁止扇出到评论启动日志。
func logServerSchedulingToActiveBindingsBestEffort(taskID string, statusData map[string]interface{}) {
	taskID = trim(taskID)
	if taskID == "" || isCommentIDMissingSchedulingMessage(statusData) {
		return
	}
	if !shouldPersistServerSchedulingLog(statusData) {
		return
	}
	statusFilter := `'pending','waiting_previous','starting','failed'`
	if strings.ToLower(strField(statusData, "status")) == "success" {
		// 任务级成功不得涂到 failed 卡片：否则日志「启动成功」与红条「启动失败」并存
		statusFilter = `'pending','waiting_previous','starting'`
	}
	rows, err := db.Query(
		`SELECT `+commentContainerBindingSelectColumns+`
		FROM cloud_comment_container_bindings
		WHERE task_id=? AND status IN (`+statusFilter+`)
		ORDER BY updated_at DESC`,
		taskID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	list, err := scanCommentContainerBindingRows(rows)
	if err != nil {
		return
	}
	for i := range list {
		persistServerSchedulingLogForBinding(&list[i], copyStatusDataWithoutTraceID(statusData))
	}
}

func copyStatusDataWithoutTraceID(src map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		if k == "trace_id" {
			continue
		}
		out[k] = v
	}
	return out
}
