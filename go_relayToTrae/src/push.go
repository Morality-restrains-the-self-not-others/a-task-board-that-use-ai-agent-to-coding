package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"tracelog"
)

func statusPushURL(reg *RegisteredTask) string {
	origin := strings.TrimRight(strings.TrimSpace(reg.TaskAPIOrigin), "/")
	tid := strings.TrimSpace(reg.TenantID)
	wid := strings.TrimSpace(reg.WorkspaceID)
	task := strings.TrimSpace(reg.TaskID)
	cid := strings.TrimSpace(reg.CommentID)
	if origin == "" || tid == "" || wid == "" || task == "" || cid == "" || cid == "-" {
		return ""
	}
	return fmt.Sprintf("%s/api/tenant/%s/workspace/%s/task/%s/comment/%s/cloud/relay-to-trae/status-push/",
		origin, tid, wid, task, cid)
}

func collectStatusSnapshotLocked(cursor int, portListening bool, viewerTaskID string) map[string]interface{} {
	// Use Signal(0) for immediate zombie detection (like Python's poll()).
	// ProcessState==nil only becomes non-nil after Wait(), which may lag
	// behind actual process death while the stdout scanner drains.
	containerRef := strings.TrimSpace(state.ContainerName)
	if containerRef == "" {
		containerRef = strings.TrimSpace(state.ContainerID)
	}
	containerMode := containerRef != ""

	procAlive := false
	if !containerMode {
		procAlive = cmd != nil && cmd.Process != nil && cmd.Process.Signal(syscall.Signal(0)) == nil
		if state.Running && !procAlive {
			state.Running = false
			state.PID = 0
		}
	}

	onlinePort := state.Port
	if onlinePort <= 0 {
		onlinePort = pickPort()
	}
	trackedRunning := state.Running

	logs := logsForViewerLocked(viewerTaskID)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(logs) {
		cursor = len(logs)
	}
	nextCursor := len(logs)

	snap := map[string]interface{}{
		"running":           trackedRunning,
		"online_service_up": trackedRunning || portListening,
		"port_listening":    portListening,
		"orphan_port":       portListening && !trackedRunning,
		"pid":               float64(state.PID),
		"port":              float64(onlinePort),
		"ui_url":            state.UIURL,
		"error":             state.Error,
		"active_task_id":    trimTaskID(state.ActiveTaskID),
		"log_task_id":       logTaskIDForViewerLocked(viewerTaskID),
		"logs":              logs[cursor:],
		"next_cursor":       float64(nextCursor),
	}
	if containerMode {
		snap["container_id"] = state.ContainerID
		snap["container_name"] = state.ContainerName
		snap["image"] = state.Image
		snap["mode"] = "selected_image"
	}
	return snap
}

func collectStatusSnapshot(viewerTaskID string, cursor int) map[string]interface{} {
	stateMu.Lock()
	onlinePort := state.Port
	if onlinePort <= 0 {
		onlinePort = pickPort()
	}
	stateMu.Unlock()

	pl := portListening(onlinePort)

	stateMu.Lock()
	defer stateMu.Unlock()
	return collectStatusSnapshotLocked(cursor, pl, viewerTaskID)
}

func resolveAccessTokenForRegisterLocked(accessToken string) string {
	token := strings.TrimSpace(accessToken)
	stateToken := strings.TrimSpace(state.AccessToken)
	if stateToken != "" {
		return stateToken
	}
	return token
}

func syncRegisteredAccessTokenFromStateLocked(reg *RegisteredTask) {
	stateToken := strings.TrimSpace(state.AccessToken)
	if stateToken != "" {
		reg.AccessToken = stateToken
	}
}

func registerTaskLocked(tenantID, workspaceID, taskID, taskAPIOrigin, accessToken, commentID string) {
	task := strings.TrimSpace(taskID)
	if task == "" {
		return
	}
	origin := extractOrigin(taskAPIOrigin)
	if origin == "" {
		origin = strings.TrimRight(strings.TrimSpace(taskAPIOrigin), "/")
	}
	token := resolveAccessTokenForRegisterLocked(accessToken)
	if origin == "" || token == "" {
		return
	}
	// 新注册从 0 推送本会话日志：start 已清空 Logs，若用 len(Logs) 会跳过已缓冲的
	// bootstrap 行，导致前端先收到 SSE 中段、再收到 /status 全量时重复拼接引导日志。
	logCursor := 0
	if existing, ok := registeredTasks[task]; ok {
		logCursor = existing.LogCursor
	}
	registeredTasks[task] = &RegisteredTask{
		TenantID:      strings.TrimSpace(tenantID),
		WorkspaceID:   strings.TrimSpace(workspaceID),
		TaskID:        task,
		TaskAPIOrigin: origin,
		AccessToken:   token,
		CommentID:     strings.TrimSpace(commentID),
		LogCursor:     logCursor,
	}
	if _, ok := taskSeq[task]; !ok {
		taskSeq[task] = 0
	}
}

func unregisterTaskLocked(taskID string) {
	task := strings.TrimSpace(taskID)
	if task == "" {
		return
	}
	delete(registeredTasks, task)
	delete(taskSeq, task)
}

func pushStatusToBackend(reg *RegisteredTask) bool {
	task := reg.TaskID
	if !tryBeginPushInFlight(task) {
		return true
	}
	defer endPushInFlight(task)

	stateMu.Lock()
	pending := state.TokenSyncPending
	stateMu.Unlock()
	if pending {
		return true
	}

	// Proactive refresh before push to avoid long-run 401 when TTL is nearly exhausted.
	if err := ensureFreshAccessToken(time.Now()); err != nil {
		appendLog(fmt.Sprintf("[relayToTrae] token-refresh: proactive refresh failed (will push with current token): %v", err))
	}

	stateMu.Lock()
	syncRegisteredAccessTokenFromStateLocked(reg)
	if reg.LogCursor < 0 {
		reg.LogCursor = 0
	}
	onlinePort := state.Port
	if onlinePort <= 0 {
		onlinePort = pickPort()
	}
	cursor := reg.LogCursor
	stateMu.Unlock()

	pl := portListening(onlinePort)

	stateMu.Lock()
	snapshot := collectStatusSnapshotLocked(cursor, pl, task)
	if current, ok := registeredTasks[task]; ok && current == reg {
		if nc, ok := snapshot["next_cursor"].(float64); ok {
			current.LogCursor = int(nc)
		} else {
			current.LogCursor = cursor
		}
	}
	seq := taskSeq[task] + 1
	taskSeq[task] = seq
	stateMu.Unlock()

	body := map[string]interface{}{
		"seq":          float64(seq),
		"access_token": reg.AccessToken,
		"status":       snapshot,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		appendLog(fmt.Sprintf("[relayToTrae] status push marshal failed: %v", err))
		return false
	}

	urlStr := statusPushURL(reg)
	if strings.TrimSpace(urlStr) == "" {
		appendLog("[relayToTrae] status push skipped: comment_id required for TaskApiEndPoint")
		return false
	}
	ctx := activeCorrelationCtx()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)

	resp, err := backendPushHTTPClient().Do(req)
	if err != nil {
		appendLog(fmt.Sprintf("[relayToTrae] status push failed: %v", err))
		return false
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		detail := string(raw)
		if len(detail) > 200 {
			detail = detail[:200]
		}
		appendLog(fmt.Sprintf("[relayToTrae] status push HTTP %d: %s", resp.StatusCode, detail))
		// 401：token 已失效，继续推只会刷屏；立即注销，与「网络/5xx missed ACK」区分。
		if resp.StatusCode == http.StatusUnauthorized {
			stateMu.Lock()
			appendLogLocked(fmt.Sprintf(
				"[relayToTrae] status push 401 unauthorized, unregistering task %s",
				task,
			))
			unregisterTaskLocked(task)
			stateMu.Unlock()
		}
		return false
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		payload = map[string]interface{}{}
	}

	ack, _ := payload["ack"]
	statusStr, _ := payload["status"].(string)

	var ackInt int
	switch v := ack.(type) {
	case float64:
		ackInt = int(v)
	case int:
		ackInt = v
	case string:
		ackInt, _ = strconv.Atoi(v)
	}

	if ackInt == seq && statusStr == "ok" {
		return true
	}

	appendLog(fmt.Sprintf("[relayToTrae] status push unacknowledged task=%s seq=%d ack=%v", task, seq, ack))
	return false
}

// pushAllRegisteredTasks 事件驱动推送：遍历当前注册任务，逐个 pushStatusToBackend。
// 替代原 1.5s 轮询（OPT-20260816-032），由 /v1/register、/v1/start、/v1/stop
// 与子进程状态变化触发；后端按最后一次 push 超时判定离线，sidecar 不再盲推心跳。
func pushAllRegisteredTasks() {
	stateMu.Lock()
	tasks := make([]*RegisteredTask, 0, len(registeredTasks))
	for _, reg := range registeredTasks {
		tasks = append(tasks, reg)
	}
	stateMu.Unlock()

	for _, reg := range tasks {
		pushStatusToBackend(reg)
	}
}

var (
	pushInFlight        sync.Map
	backendPushClient   *http.Client
	backendPushClientMu sync.Mutex
)

func backendPushHTTPClient() *http.Client {
	backendPushClientMu.Lock()
	defer backendPushClientMu.Unlock()
	timeout := time.Duration(pushTimeoutSec * float64(time.Second))
	if backendPushClient == nil {
		backendPushClient = newBackendHTTPClient(timeout)
		return backendPushClient
	}
	if backendPushClient.Timeout != timeout {
		backendPushClient = newBackendHTTPClient(timeout)
	}
	return backendPushClient
}

func tryBeginPushInFlight(taskID string) bool {
	_, loaded := pushInFlight.LoadOrStore(taskID, struct{}{})
	return !loaded
}

func endPushInFlight(taskID string) {
	pushInFlight.Delete(taskID)
}

