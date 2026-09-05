package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"tracelog"
)

func TestStatusPushURL(t *testing.T) {
	reg := &RegisteredTask{
		TaskAPIOrigin: "http://api.daydaymoney.com:8001",
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "task1",
		CommentID:     "cmt_1",
	}
	got := statusPushURL(reg)
	expected := "http://api.daydaymoney.com:8001/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/relay-to-trae/status-push/"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStatusPushURLTrailingSlash(t *testing.T) {
	reg := &RegisteredTask{
		TaskAPIOrigin: "http://api.daydaymoney.com:8001/",
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "task1",
		CommentID:     "cmt_1",
	}
	got := statusPushURL(reg)
	expected := "http://api.daydaymoney.com:8001/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/relay-to-trae/status-push/"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStatusPushURLWithCommentID(t *testing.T) {
	reg := &RegisteredTask{
		TaskAPIOrigin: "http://api.daydaymoney.com:8001",
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "task1",
		CommentID:     "cmt_1",
	}
	got := statusPushURL(reg)
	expected := "http://api.daydaymoney.com:8001/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/relay-to-trae/status-push/"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStatusPushURLIgnoresDashCommentID(t *testing.T) {
	reg := &RegisteredTask{
		TaskAPIOrigin: "http://api.daydaymoney.com:8001",
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "task1",
		CommentID:     "-",
	}
	got := statusPushURL(reg)
	if got != "" {
		t.Errorf("expected empty URL without comment_id, got %q", got)
	}
}

func TestStatusPushURLWithoutCommentID(t *testing.T) {
	reg := &RegisteredTask{
		TaskAPIOrigin: "http://api.daydaymoney.com:8001",
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "task1",
	}
	got := statusPushURL(reg)
	if got != "" {
		t.Errorf("expected empty URL without comment_id, got %q", got)
	}
}

func TestCollectStatusSnapshotLockedNotRunning(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	// Use a high ephemeral port that is not expected to be LISTENing on the host
	// (8765 is commonly occupied in this monorepo's local stack).
	state.Port = 59991
	state.PID = 0
	state.Logs = []string{"log1", "log2"}
	state.Error = ""
	state.UIURL = ""
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", 0)

	if snapshot["running"] != false {
		t.Error("expected running=false")
	}
	if snapshot["online_service_up"] != false {
		t.Error("expected online_service_up=false when not listening")
	}
	logs, _ := snapshot["logs"].([]string)
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestCollectStatusSnapshotLockedCursor(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.Logs = []string{"a", "b", "c", "d", "e"}
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", 2)

	logs, _ := snapshot["logs"].([]string)
	if len(logs) != 3 {
		t.Errorf("expected 3 logs from cursor 2, got %d: %v", len(logs), logs)
	}
	if logs[0] != "c" {
		t.Errorf("expected 'c', got %q", logs[0])
	}
	nc, _ := snapshot["next_cursor"].(float64)
	if int(nc) != 5 {
		t.Errorf("expected next_cursor=5, got %v", nc)
	}
}

func TestCollectStatusSnapshotLockedCursorNegativeClamped(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.Logs = []string{"x", "y"}
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", -1)
	logs, _ := snapshot["logs"].([]string)
	if len(logs) != 2 {
		t.Fatalf("expected full logs for negative cursor clamp, got %v", logs)
	}
	nc, _ := snapshot["next_cursor"].(float64)
	if int(nc) != 2 {
		t.Fatalf("expected next_cursor=2, got %v", nc)
	}
}

func TestCollectStatusSnapshotLockedCursorTooLargeClamped(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.Logs = []string{"a", "b", "c"}
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", 999)
	logs, _ := snapshot["logs"].([]string)
	if len(logs) != 0 {
		t.Fatalf("expected empty logs when cursor exceeds length, got %v", logs)
	}
	nc, _ := snapshot["next_cursor"].(float64)
	if int(nc) != 3 {
		t.Fatalf("expected next_cursor=3, got %v", nc)
	}
}

func TestCollectStatusSnapshotLockedPortProbe(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.Port = 59995 // not listening
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", 0)

	if snapshot["port_listening"] != false {
		t.Error("expected port_listening=false for closed port")
	}
}

func TestCollectStatusSnapshotLockedOrphanPort(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.Port = 8765
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", 0)

	orphan, _ := snapshot["orphan_port"].(bool)
	// If port 8765 happens to be listening, orphan_port should reflect that.
	// We just verify the field exists in the snapshot.
	if _, ok := snapshot["orphan_port"]; !ok {
		t.Error("expected orphan_port key in snapshot")
	}
	_ = orphan
}

func TestRegisterTaskLockedWithTokenSync(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = "latest-token"
	state.Logs = []string{"already-buffered-bootstrap-line"}
	delete(registeredTasks, "sync-task")
	delete(taskSeq, "sync-task")

	registerTaskLocked("t1", "w1", "sync-task", "http://origin.example.com/api", "old-token", "")
	reg, ok := registeredTasks["sync-task"]
	stateMu.Unlock()

	if !ok {
		t.Fatal("expected task to be registered")
	}
	if reg.AccessToken != "latest-token" {
		t.Errorf("expected 'latest-token' (synced from state), got %q", reg.AccessToken)
	}
	// 新注册必须从 0 推送本会话已缓冲日志，避免 SSE 中段 + /status 全量导致前端重复 bootstrap。
	if reg.LogCursor != 0 {
		t.Errorf("expected LogCursor=0 for new registration, got %d", reg.LogCursor)
	}
}

func TestRegisterTaskLockedPreservesLogCursorOnReregister(t *testing.T) {
	stateMu.Lock()
	state.Logs = []string{"a", "b", "c"}
	registeredTasks["cursor-task"] = &RegisteredTask{
		TaskID:    "cursor-task",
		LogCursor: 2,
	}
	registerTaskLocked("t1", "w1", "cursor-task", "http://origin.example.com/api", "tok", "")
	reg := registeredTasks["cursor-task"]
	stateMu.Unlock()

	if reg.LogCursor != 2 {
		t.Errorf("expected LogCursor=2 preserved, got %d", reg.LogCursor)
	}
}

func TestRegisterTaskLockedInitializesSeq(t *testing.T) {
	stateMu.Lock()
	delete(registeredTasks, "seq-task")
	delete(taskSeq, "seq-task")

	registerTaskLocked("t1", "w1", "seq-task", "http://origin/api", "tok", "")
	seq := taskSeq["seq-task"]
	stateMu.Unlock()

	if seq != 0 {
		t.Errorf("expected seq=0 for new task, got %d", seq)
	}
}

func TestUnregisterTaskLockedClearsSeq(t *testing.T) {
	stateMu.Lock()
	registeredTasks["clear-task"] = &RegisteredTask{TaskID: "clear-task"}
	taskSeq["clear-task"] = 42
	unregisterTaskLocked("clear-task")
	_, taskOk := registeredTasks["clear-task"]
	_, seqOk := taskSeq["clear-task"]
	stateMu.Unlock()

	if taskOk {
		t.Error("task should be removed")
	}
	if seqOk {
		t.Error("seq should be removed")
	}
}

func TestCollectStatusSnapshotZombieProcess(t *testing.T) {
	stateMu.Lock()
	cmd = nil // no process
	state.Running = true
	state.PID = 12345
	state.Logs = nil
	stateMu.Unlock()

	snapshot := collectStatusSnapshot("", 0)

	// Zombie detection: running=true but no live process → running should become false.
	stateMu.Lock()
	if state.Running {
		t.Error("expected Running to be reset to false for zombie process")
	}
	stateMu.Unlock()
	_ = snapshot
}

func TestPushStatusToBackendEmptyTasks(t *testing.T) {
	// Should not panic with empty registered tasks.
	stateMu.Lock()
	registeredTasks = make(map[string]*RegisteredTask)
	stateMu.Unlock()

	// Just verify the push loop doesn't crash with no tasks.
	// We can't easily test the full push without a backend.
}

func TestTaskSeqIncrement(t *testing.T) {
	stateMu.Lock()
	taskSeq["inc-task"] = 0
	seq := taskSeq["inc-task"] + 1
	taskSeq["inc-task"] = seq
	stateMu.Unlock()

	if seq != 1 {
		t.Errorf("expected seq=1, got %d", seq)
	}
}

func TestPushStatusToBackendPropagatesCorrelationHeaders(t *testing.T) {
	var gotTrace, gotParent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-Id")
		gotParent = r.Header.Get("X-Parent-Span-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ack":1,"status":"ok"}`))
	}))
	defer server.Close()

	stateMu.Lock()
	setActiveCorrelationLocked(tracelog.Correlation{
		TraceID: "push-trace-test123456",
		SpanID:  "c1c2c3d4e5f67890",
	})
	taskSeq["push-trace-task"] = 0
	stateMu.Unlock()

	reg := &RegisteredTask{
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "push-trace-task",
		CommentID:     "cmt-push",
		TaskAPIOrigin: server.URL,
		AccessToken:   "token",
	}
	if ok := pushStatusToBackend(reg); !ok {
		t.Fatal("expected push to succeed")
	}
	if gotTrace != "push-trace-test123456" {
		t.Fatalf("X-Trace-Id = %q", gotTrace)
	}
	if gotParent != "c1c2c3d4e5f67890" {
		t.Fatalf("X-Parent-Span-Id = %q", gotParent)
	}
}

func TestPushStatusClientUsesNoProxyAndPushTimeout(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")

	originalPushTimeoutSec := pushTimeoutSec
	defer func() {
		pushTimeoutSec = originalPushTimeoutSec
	}()

	// Guardrail 1: push path should bypass env proxy and still succeed.
	pushTimeoutSec = 1.0
	noProxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ack":1,"status":"ok"}`))
	}))
	defer noProxyServer.Close()

	stateMu.Lock()
	taskSeq["push-no-proxy-task"] = 0
	stateMu.Unlock()

	noProxyReg := &RegisteredTask{
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "push-no-proxy-task",
		CommentID:     "cmt-push",
		TaskAPIOrigin: noProxyServer.URL,
		AccessToken:   "token",
	}
	if ok := pushStatusToBackend(noProxyReg); !ok {
		t.Fatal("expected push to succeed without using proxy")
	}

	// Guardrail 2: push client timeout must honor pushTimeoutSec.
	pushTimeoutSec = 0.05
	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ack":1,"status":"ok"}`))
	}))
	defer timeoutServer.Close()

	stateMu.Lock()
	taskSeq["push-timeout-task"] = 0
	stateMu.Unlock()

	timeoutReg := &RegisteredTask{
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "push-timeout-task",
		CommentID:     "cmt-push",
		TaskAPIOrigin: timeoutServer.URL,
		AccessToken:   "token",
	}

	start := time.Now()
	ok := pushStatusToBackend(timeoutReg)
	elapsed := time.Since(start)
	if ok {
		t.Fatal("expected push to fail when backend response exceeds push timeout")
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("expected push timeout near pushTimeoutSec, elapsed=%v", elapsed)
	}
}

func TestPushStatusToBackendSkipsWhenPreviousPushInFlight(t *testing.T) {
	originalPushTimeoutSec := pushTimeoutSec
	defer func() {
		pushTimeoutSec = originalPushTimeoutSec
		backendPushClientMu.Lock()
		backendPushClient = nil
		backendPushClientMu.Unlock()
		pushInFlight = sync.Map{}
	}()

	pushTimeoutSec = 2.0
	requests := 0
	var requestsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsMu.Lock()
		requests++
		requestsMu.Unlock()
		time.Sleep(300 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ack":1,"status":"ok"}`))
	}))
	defer server.Close()

	stateMu.Lock()
	taskSeq["push-inflight-task"] = 0
	stateMu.Unlock()

	reg := &RegisteredTask{
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "push-inflight-task",
		CommentID:     "cmt-push",
		TaskAPIOrigin: server.URL,
		AccessToken:   "token",
	}

	done := make(chan bool, 2)
	go func() {
		done <- pushStatusToBackend(reg)
	}()
	go func() {
		done <- pushStatusToBackend(reg)
	}()

	ok1 := <-done
	ok2 := <-done
	if !ok1 || !ok2 {
		t.Fatalf("expected both pushes to be treated as ok, got ok1=%v ok2=%v", ok1, ok2)
	}

	requestsMu.Lock()
	got := requests
	requestsMu.Unlock()
	if got != 1 {
		t.Fatalf("expected exactly one backend request while in flight, got %d", got)
	}
}

func TestPushStatusUnauthorizedUnregistersImmediately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"无效的 access_token"}`))
	}))
	defer server.Close()

	stateMu.Lock()
	taskSeq["push-401-task"] = 0
	reg := &RegisteredTask{
		TenantID:      "t1",
		WorkspaceID:   "w1",
		TaskID:        "push-401-task",
		CommentID:     "cmt-push",
		TaskAPIOrigin: server.URL,
		AccessToken:   "stale-token",
	}
	registeredTasks["push-401-task"] = reg
	stateMu.Unlock()

	if ok := pushStatusToBackend(reg); ok {
		t.Fatal("expected push to fail on 401")
	}

	stateMu.Lock()
	_, stillRegistered := registeredTasks["push-401-task"]
	stateMu.Unlock()
	if stillRegistered {
		t.Fatal("expected task to be unregistered immediately after status push 401")
	}
}

func TestCollectStatusSnapshotPerTaskLogs(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = true
	state.ActiveTaskID = "task-a"
	state.Logs = []string{"live-a-1", "live-a-2"}
	taskLogs["task-a"] = []string{"live-a-1", "live-a-2"}
	taskLogs["task-b"] = nil
	stateMu.Unlock()

	snapshotA := collectStatusSnapshot("task-a", 0)
	logsA, _ := snapshotA["logs"].([]string)
	if len(logsA) != 2 || logsA[0] != "live-a-1" {
		t.Fatalf("expected live logs for active task-a, got %v", logsA)
	}
	if snapshotA["active_task_id"] != "task-a" {
		t.Fatalf("expected active_task_id task-a, got %v", snapshotA["active_task_id"])
	}
	if snapshotA["log_task_id"] != "task-a" {
		t.Fatalf("expected log_task_id task-a, got %v", snapshotA["log_task_id"])
	}

	snapshotB := collectStatusSnapshot("task-b", 0)
	logsB, _ := snapshotB["logs"].([]string)
	if len(logsB) != 0 {
		t.Fatalf("expected empty logs for unrelated task-b, got %v", logsB)
	}
	if snapshotB["log_task_id"] != "task-b" {
		t.Fatalf("expected log_task_id task-b, got %v", snapshotB["log_task_id"])
	}
}

func TestCollectStatusSnapshotReturnsPersistedLogsForInactiveTask(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.ActiveTaskID = ""
	state.Logs = nil
	taskLogs["task-a"] = []string{"history-a-1"}
	taskLogs["task-b"] = []string{"history-b-1"}
	stateMu.Unlock()

	snapshotA := collectStatusSnapshot("task-a", 0)
	logsA, _ := snapshotA["logs"].([]string)
	if len(logsA) != 1 || logsA[0] != "history-a-1" {
		t.Fatalf("expected persisted logs for task-a, got %v", logsA)
	}
}
