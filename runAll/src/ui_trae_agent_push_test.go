package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ---------- 工具函数 ----------

func testTraeAgentPushRunner(t *testing.T) *Runner {
	t.Helper()
	store := NewStatusStore()
	runner, err := NewRunner(&Config{}, store)
	if err != nil {
		t.Fatal(err)
	}
	stubNoPortListenersForTest(runner)
	return runner
}

// writeTraeAgentPushState 在临时 stateDir 中写入推送状态文件，返回 stateDir。
func writeTraeAgentPushState(t *testing.T, pending, running bool, sha, logBody string) string {
	t.Helper()
	dir := t.TempDir()
	if pending {
		if err := os.WriteFile(filepath.Join(dir, "trae_agent_docker_push_pending"), []byte("stub\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if running {
		// 写一个不可能存活的 PID（如 999999），锁存在但进程已死 → Running 应报 false（陈旧锁）
		if err := os.WriteFile(filepath.Join(dir, "trae_agent_docker_push.lock"), []byte("999999\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if sha != "" {
		if err := os.WriteFile(filepath.Join(dir, "trae_agent_docker_push_sha"), []byte(sha+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if logBody != "" {
		if err := os.WriteFile(filepath.Join(dir, "trae_agent_docker_push.log"), []byte(logBody), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// ---------- GET /api/trae-agent-push/status ----------

func TestTraeAgentPushStatus_PendingAndNoRunning(t *testing.T) {
	dir := writeTraeAgentPushState(t, true, false, "abc123", "line1\nline2\n")
	t.Setenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR", dir)
	t.Setenv("RUNALL_TRAE_AGENT_PUSH_SCRIPT", "/tmp/stub-trae-push.sh")
	runner := testTraeAgentPushRunner(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trae-agent-push/status", nil)
	rec := httptest.NewRecorder()
	handleTraeAgentPushStatus(rec, req, runner)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var view traeAgentPushStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if !view.Pending {
		t.Fatal("Pending should be true when pending file exists")
	}
	if view.Running {
		t.Fatal("Running should be false with stale lock (dead pid 999999)")
	}
	if view.LastSHA != "abc123" {
		t.Fatalf("LastSHA=%q want abc123", view.LastSHA)
	}
	if len(view.LogTail) != 2 || view.LogTail[0] != "line1" || view.LogTail[1] != "line2" {
		t.Fatalf("LogTail=%v", view.LogTail)
	}
}

func TestTraeAgentPushStatus_NoPendingAndRunningLockAlive(t *testing.T) {
	dir := writeTraeAgentPushState(t, false, true, "", "")
	// 用当前进程自身 PID 作为「存活锁」→ Running 应为 true
	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(dir, "trae_agent_docker_push.lock"), []byte(itoa(pid)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR", dir)
	runner := testTraeAgentPushRunner(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trae-agent-push/status", nil)
	rec := httptest.NewRecorder()
	handleTraeAgentPushStatus(rec, req, runner)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var view traeAgentPushStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Pending {
		t.Fatal("Pending should be false")
	}
	if !view.Running {
		t.Fatal("Running should be true with alive pid lock")
	}
}

func TestTraeAgentPushStatus_MethodNotAllowed(t *testing.T) {
	dir := writeTraeAgentPushState(t, false, false, "", "")
	t.Setenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR", dir)
	runner := testTraeAgentPushRunner(t)

	req := httptest.NewRequest(http.MethodPost, "/api/trae-agent-push/status", nil)
	rec := httptest.NewRecorder()
	handleTraeAgentPushStatus(rec, req, runner)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rec.Code)
	}
}

// ---------- POST /api/trae-agent-push ----------

// TestTraeAgentPushAction_ConflictWhenRunning 运行锁存活时触发 → 409。
func TestTraeAgentPushAction_ConflictWhenRunning(t *testing.T) {
	dir := writeTraeAgentPushState(t, true, true, "", "")
	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(dir, "trae_agent_docker_push.lock"), []byte(itoa(pid)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR", dir)
	runner := testTraeAgentPushRunner(t)

	req := httptest.NewRequest(http.MethodPost, "/api/trae-agent-push", nil)
	rec := httptest.NewRecorder()
	handleTraeAgentPushAction(rec, req, runner)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d want 409 body=%s", rec.Code, rec.Body.String())
	}
}

// TestTraeAgentPushAction_NoPending 无 pending 登记时触发 → 400。
func TestTraeAgentPushAction_NoPending(t *testing.T) {
	dir := writeTraeAgentPushState(t, false, false, "", "")
	t.Setenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR", dir)
	runner := testTraeAgentPushRunner(t)

	req := httptest.NewRequest(http.MethodPost, "/api/trae-agent-push", nil)
	rec := httptest.NewRecorder()
	handleTraeAgentPushAction(rec, req, runner)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

// TestTraeAgentPushAction_Accepted 有 pending + 脚本 stub 可执行 → 202 accepted。
func TestTraeAgentPushAction_Accepted(t *testing.T) {
	dir := writeTraeAgentPushState(t, true, false, "", "")
	t.Setenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR", dir)
	stub := filepath.Join(t.TempDir(), "stub-push.sh")
	if err := os.WriteFile(stub, []byte("#!/usr/bin/env bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUNALL_TRAE_AGENT_PUSH_SCRIPT", stub)
	runner := testTraeAgentPushRunner(t)

	req := httptest.NewRequest(http.MethodPost, "/api/trae-agent-push", nil)
	rec := httptest.NewRecorder()
	handleTraeAgentPushAction(rec, req, runner)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202 body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "accepted" {
		t.Fatalf("resp=%v", resp)
	}
}

// ---------- 工具函数 ----------

func TestReadFirstLineAndLogTail(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("  abc  \ndef\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := readFirstLine(p); got != "abc" {
		t.Fatalf("readFirstLine=%q want abc", got)
	}
	// 40 行内全量
	var lines []string
	for i := 1; i <= 5; i++ {
		lines = append(lines, "line"+itoa(i))
	}
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	if got := readLogTail(p, 40); len(got) != 5 || got[4] != "line5" {
		t.Fatalf("readLogTail=%v", got)
	}
	// 截断到 maxLines
	big := make([]string, 10)
	for i := range big {
		big[i] = "row" + itoa(i)
	}
	if err := os.WriteFile(p, []byte(strings.Join(big, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	tail := readLogTail(p, 3)
	if len(tail) != 3 || tail[0] != "row7" || tail[2] != "row9" {
		t.Fatalf("truncated tail=%v", tail)
	}
}

func TestReadLogFromOffset_Incremental(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(p, []byte("a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	chunk, off, err := readLogFromOffset(p, 0)
	if err != nil {
		t.Fatal(err)
	}
	if chunk != "a\n" || off != 2 {
		t.Fatalf("first chunk=%q off=%d", chunk, off)
	}
	// 追加后从 off 继续读
	if err := os.WriteFile(p, []byte("a\nb\n"), 0644); err != nil {
		t.Fatal(err)
	}
	chunk2, off2, err := readLogFromOffset(p, off)
	if err != nil {
		t.Fatal(err)
	}
	if chunk2 != "b\n" || off2 != 4 {
		t.Fatalf("second chunk=%q off=%d", chunk2, off2)
	}
	// 无新增 → 空
	chunk3, off3, err := readLogFromOffset(p, off2)
	if err != nil {
		t.Fatal(err)
	}
	if chunk3 != "" || off3 != 4 {
		t.Fatalf("idle chunk=%q off=%d", chunk3, off3)
	}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

