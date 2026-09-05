package main

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"tracelog"
)

// RegisteredTask represents a task registered for status push.
type RegisteredTask struct {
	TenantID      string
	WorkspaceID   string
	TaskID        string
	TaskAPIOrigin string
	AccessToken   string
	CommentID     string
	LastPushAt    float64
	LogCursor     int
}

// relayState holds the current runtime state of the onlineServiceJS subprocess
// or a selected-image Docker container.
type relayState struct {
	Running              bool
	PID                  int
	Port                 int
	UIURL                string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt time.Time // UTC; zero means unknown → skip proactive refresh
	Logs                 []string
	Error                string
	StartedAt            float64
	ActiveTaskID         string
	ActiveTraceID        string
	ActiveSpanID         string
	ContainerID          string
	ContainerName        string
	Image                string
	// TokenSyncPending is true while selected_image/child waits for post-exchange tokens.
	// Status push must not use bootstrap ACCESS_TOKEN during this window.
	TokenSyncPending bool
}

// Global state, protected by stateMu.
var (
	stateMu sync.Mutex
	state   = relayState{
		Logs: make([]string, 0),
	}

	// lifecycleMu serializes /v1/start and /v1/stop so stop cannot kill a newly started onlineServiceJS.
	// Single shared onlineServiceJS process → must stay process-global (not per-task).
	lifecycleMu sync.Mutex

	registeredTasks = make(map[string]*RegisteredTask)
	taskSeq         = make(map[string]int)
	taskLogs        = make(map[string][]string)
)

// Constants matching Python version.
const (
	tokenExchangeTimeout = 15 * time.Second
	tokenExchangeRetries = 2
	maxLogLines          = 4000
	// Align with taskCredentialService tokenReuseMinTTL: refresh before the last 5 minutes.
	tokenProactiveRefreshSkew = 5 * time.Minute
)

var requiredEnvKeys = []string{
	"TASK_API_ENDPOINT_ORIGIN",
	"BUSINESS_API_ENDPOINT_ORIGIN",
	"ACCESS_TOKEN",
}

func trimTaskID(taskID string) string {
	return strings.TrimSpace(taskID)
}

// lifecycleLockWait is how long /v1/start|/v1/stop wait for lifecycleMu before 503.
// Override with RELAY_LIFECYCLE_LOCK_WAIT_SEC (seconds). Default 30.
func lifecycleLockWait() time.Duration {
	raw := strings.TrimSpace(os.Getenv("RELAY_LIFECYCLE_LOCK_WAIT_SEC"))
	if raw == "" {
		return 30 * time.Second
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec < 0 {
		return 30 * time.Second
	}
	if sec == 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// tryAcquireLifecycleLock waits up to lifecycleLockWait for the global start/stop mutex.
// On failure the caller must not Unlock. Returns false when timed out.
func tryAcquireLifecycleLock() bool {
	wait := lifecycleLockWait()
	if wait == 0 {
		lifecycleMu.Lock()
		return true
	}
	deadline := time.Now().Add(wait)
	for {
		if lifecycleMu.TryLock() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// resetTaskLogsLocked clears persisted startup logs for one task. Caller must hold stateMu.
func resetTaskLogsLocked(taskID string) {
	task := trimTaskID(taskID)
	if task == "" {
		return
	}
	taskLogs[task] = make([]string, 0)
}

// clearLogsForScopeLocked clears startup logs for a scoped task and resets push cursors.
// If registeredTasks[task] exists, tenant/workspace must match or ok=false (scope mismatch).
// If the task is the active live task, also clears state.Logs. Caller must hold stateMu.
func clearLogsForScopeLocked(tenantID, workspaceID, taskID string) (clearedTask string, clearedLive bool, ok bool, mismatch bool) {
	task := trimTaskID(taskID)
	tenant := strings.TrimSpace(tenantID)
	workspace := strings.TrimSpace(workspaceID)
	if task == "" || tenant == "" || workspace == "" {
		return "", false, false, false
	}
	if reg, exists := registeredTasks[task]; exists && reg != nil {
		if strings.TrimSpace(reg.TenantID) != tenant || strings.TrimSpace(reg.WorkspaceID) != workspace {
			return task, false, false, true
		}
	}
	active := trimTaskID(state.ActiveTaskID)
	resetTaskLogsLocked(task)
	if active != "" && task == active {
		state.Logs = make([]string, 0)
		clearedLive = true
	}
	if reg, exists := registeredTasks[task]; exists && reg != nil {
		reg.LogCursor = 0
	}
	return task, clearedLive, true, false
}

func appendTaskLogLocked(taskID, line string) {
	task := trimTaskID(taskID)
	if task == "" || line == "" {
		return
	}
	rows := append(taskLogs[task], line)
	if len(rows) > maxLogLines {
		rows = rows[len(rows)-maxLogLines:]
	}
	taskLogs[task] = rows
}

// logsForViewerLocked returns live logs for the active task or persisted logs for other tasks.
func logsForViewerLocked(viewerTaskID string) []string {
	logs := state.Logs
	if logs == nil {
		logs = []string{}
	}
	active := trimTaskID(state.ActiveTaskID)
	viewer := trimTaskID(viewerTaskID)
	if viewer == "" {
		return logs
	}
	if active != "" && viewer == active {
		return logs
	}
	stored := taskLogs[viewer]
	if stored == nil {
		return []string{}
	}
	return stored
}

func logTaskIDForViewerLocked(viewerTaskID string) string {
	viewer := trimTaskID(viewerTaskID)
	if viewer != "" {
		return viewer
	}
	return trimTaskID(state.ActiveTaskID)
}

// appendLogLocked appends a line to the log buffer. Caller must hold stateMu.
func appendLogLocked(line string) {
	if line == "" {
		return
	}
	state.Logs = append(state.Logs, line)
	if len(state.Logs) > maxLogLines {
		state.Logs = state.Logs[len(state.Logs)-maxLogLines:]
	}
	appendTaskLogLocked(state.ActiveTaskID, line)
}

func activeCorrelationLocked() tracelog.Correlation {
	return tracelog.Correlation{
		TraceID: strings.TrimSpace(state.ActiveTraceID),
		SpanID:  strings.TrimSpace(state.ActiveSpanID),
	}
}

func setActiveCorrelationLocked(c tracelog.Correlation) {
	state.ActiveTraceID = strings.TrimSpace(c.TraceID)
	state.ActiveSpanID = strings.TrimSpace(c.SpanID)
}

func activeCorrelationCtx() context.Context {
	stateMu.Lock()
	defer stateMu.Unlock()
	corr := activeCorrelationLocked()
	if corr.TraceID == "" {
		return context.Background()
	}
	return tracelog.ContextWithCorrelation(context.Background(), corr)
}

func activeTraceID() string {
	return tracelog.TraceIDFromContext(activeCorrelationCtx())
}

// appendLog acquires the lock and appends a line.
func appendLog(line string) {
	if line == "" {
		return
	}
	stateMu.Lock()
	appendLogLocked(line)
	stateMu.Unlock()
	tracelog.EmitWithTrace(activeTraceID(), "info", line, "relay", nil)
}

// appendSubprocessLog stores child output for /v1/status and forwards it for Loki/Grafana.
func appendSubprocessLog(line string) {
	if line == "" {
		return
	}
	stateMu.Lock()
	appendLogLocked(line)
	stateMu.Unlock()
	tracelog.ForwardChildLine(line, "onlineServiceJS")
}
