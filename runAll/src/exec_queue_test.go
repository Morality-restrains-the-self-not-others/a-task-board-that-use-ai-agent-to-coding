package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIStatus_ExecutionQueueIdle(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw, ok := payload["execution_queue"].(map[string]any)
	if !ok {
		t.Fatalf("execution_queue missing: %#v", payload)
	}
	if raw["current"] != nil {
		t.Fatalf("idle current = %#v, want nil", raw["current"])
	}
	pending, ok := raw["pending"].([]any)
	if !ok {
		t.Fatalf("pending missing: %#v", raw)
	}
	if len(pending) != 0 {
		t.Fatalf("idle pending = %#v, want empty", pending)
	}
}

func TestAPIStatus_ExecutionQueueIncludesActiveAndPending(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name:     "g1",
			Services: []Service{{Name: "svc", Command: "true"}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := runner.SetActiveBuildAllRunID()
	runner.PublishProgress(runID, StartAllProgressEvent{
		Total: 2, Started: 1, Remaining: 1, Phase: "progress", Operation: "build", Current: "svc",
	})
	runner.SetExecQueuePending([]ExecQueuePendingItem{{
		ID: "q-1", Type: "start-all", Label: "全部启动",
	}})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	q, ok := payload["execution_queue"].(map[string]any)
	if !ok {
		t.Fatalf("execution_queue missing: %#v", payload)
	}
	current, ok := q["current"].(map[string]any)
	if !ok {
		t.Fatalf("current missing: %#v", q)
	}
	if current["kind"] != "build-all" {
		t.Fatalf("kind = %v, want build-all", current["kind"])
	}
	if current["run_id"] != runID {
		t.Fatalf("run_id = %v, want %s", current["run_id"], runID)
	}
	if current["label"] != "全部重新编译" {
		t.Fatalf("label = %v", current["label"])
	}
	pending, ok := q["pending"].([]any)
	if !ok || len(pending) != 1 {
		t.Fatalf("pending = %#v", q["pending"])
	}
	item, _ := pending[0].(map[string]any)
	if item["type"] != "start-all" || item["label"] != "全部启动" {
		t.Fatalf("pending item = %#v", item)
	}
}

func TestAPIExecQueue_ReplacePending(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	body := []byte(`{"pending":[{"id":"q-2","type":"build-all","label":"全部重新编译"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/exec-queue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body=%s", rec.Code, rec.Body.String())
	}

	got := runner.ExecQueuePending()
	if len(got) != 1 || got[0].Type != "build-all" || got[0].ID != "q-2" {
		t.Fatalf("pending = %#v", got)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/exec-queue", bytes.NewReader([]byte(`{"pending":[]}`)))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("clear status = %d, body=%s", rec2.Code, rec2.Body.String())
	}
	if got := runner.ExecQueuePending(); len(got) != 0 {
		t.Fatalf("cleared pending = %#v", got)
	}
}

func TestAPIExecQueue_DeduplicatesSameTypePending(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	// OPT-20260819-040 回归：确认连点 / 多标签同一 tick 提交两条 build-all 时，
	// /api/exec-queue 必须折叠为一条，避免 67 服务编译收尾又从头再来一轮。
	body := []byte(`{"pending":[{"id":"q-a","type":"build-all","label":"全部重新编译"},{"id":"q-b","type":"build-all","label":"全部重新编译"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/exec-queue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body=%s", rec.Code, rec.Body.String())
	}
	got := runner.ExecQueuePending()
	if len(got) != 1 || got[0].Type != "build-all" {
		t.Fatalf("pending after dedup = %#v, want single build-all", got)
	}
}

func TestBuildAllAction_PendingDoesNotBlockStart(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name:     "g1",
			Services: []Service{{Name: "svc", Command: "true"}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetExecQueuePending([]ExecQueuePendingItem{{
		ID: "q-pending", Type: "build-all", Label: "全部重新编译",
	}})
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	// 队列执行器 POST /api/build-all 时 pending 里仍可能有自己刚同步上去的
	// 同类型项；不得 409，否则横幅「已有重新编译进行中」但进度条永不出现。
	req := httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	if runner.HasPendingExecQueueType("build-all") {
		t.Fatal("pending build-all should be consumed after start")
	}
}

func TestAPIExecQueue_RejectsUnknownType(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	body := []byte(`{"pending":[{"id":"q-x","type":"rm-rf","label":"nope"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/exec-queue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestActiveBulkProgress_IncludesDevDB(t *testing.T) {
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := "single-init-db-dev-db-1"
	runner.SetActiveDevDBRun("init-db", runID)
	runner.PublishProgress(runID, StartAllProgressEvent{
		Total: 1, Started: 0, Remaining: 1, Phase: "starting", Operation: "init-db", Current: "开始执行...",
	})
	snap := runner.ActiveBulkProgress()
	if snap == nil {
		t.Fatal("expected init-db snapshot")
	}
	if snap.Kind != "init-db" || snap.RunID != runID {
		t.Fatalf("snap = %#v", snap)
	}
	if snap.Event == nil || snap.Event.Current != "开始执行..." {
		t.Fatalf("event = %#v", snap.Event)
	}
	runner.ClearActiveDevDBRun()
	if got := runner.ActiveBulkProgress(); got != nil {
		t.Fatalf("after clear, snap = %#v", got)
	}
}

func TestExecQueue_PersistsPendingAndReloadsAfterHotReplace(t *testing.T) {
	regDir := t.TempDir()
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(regDir, "reg.txt"))

	store := NewStatusStore()
	store.Init([]string{"svc"})
	cfg := &Config{
		Version: "1",
		Groups: []Group{{
			Name:     "g1",
			Services: []Service{{Name: "svc", Command: "true"}},
		}},
	}
	runner1, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := runner1.SetActiveBuildAllRunID()
	runner1.PublishProgress(runID, StartAllProgressEvent{
		Total: 2, Started: 1, Remaining: 1, Phase: "progress", Operation: "build", Current: "svc",
	})
	runner1.SetExecQueuePending([]ExecQueuePendingItem{{
		ID: "q-hot", Type: "start-all", Label: "全部启动",
	}})

	queuePath := filepath.Join(regDir, "exec_queue.json")
	if _, err := os.Stat(queuePath); err != nil {
		t.Fatalf("expected %s after SetExecQueuePending: %v", queuePath, err)
	}

	runner2, err := NewRunner(cfg, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner reload: %v", err)
	}
	runner2.LoadExecQueueFromDisk()

	// Hot-replace kills the worker goroutine: orphaned "current" must become
	// pending (for UI replay) and must NOT look like a live ActiveBulkProgress,
	// otherwise the progress panel stays stuck and cancel returns 404 forever.
	pending := runner2.ExecQueuePending()
	if len(pending) != 2 {
		t.Fatalf("reloaded pending = %#v, want orphaned build-all + start-all", pending)
	}
	if pending[0].Type != "build-all" || pending[1].Type != "start-all" {
		t.Fatalf("reloaded pending order = %#v", pending)
	}
	if snap := runner2.ActiveBulkProgress(); snap != nil {
		t.Fatalf("orphaned current must not stay active, got %#v", snap)
	}
	if ev, ok := runner2.progressBroadcaster.Latest(runID); !ok || !ev.Done || ev.Phase != "interrupted" {
		t.Fatalf("expected interrupted terminal event for %s, got ok=%v ev=%#v", runID, ok, ev)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, NewStatusStore(), runner2, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	q, _ := payload["execution_queue"].(map[string]any)
	if q["current"] != nil {
		t.Fatalf("status current after hot-replace reload = %#v, want nil", q["current"])
	}
	pendingRaw, _ := q["pending"].([]any)
	if len(pendingRaw) != 2 {
		t.Fatalf("status pending after hot-replace reload = %#v", q)
	}
	if !runner2.TryBeginBulk("start-all") {
		t.Fatal("orphaned current must not hold bulk mutex, else pending replay 409s")
	}
	runner2.EndBulk("start-all")
}

func TestExecQueue_OrphanedPreciseRestartBecomesPending(t *testing.T) {
	regDir := t.TempDir()
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(regDir, "reg.txt"))

	cfg := &Config{
		Version: "1",
		Groups: []Group{{
			Name:     "g1",
			Services: []Service{{Name: "svc", Command: "true"}},
		}},
	}
	runner1, err := NewRunner(cfg, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := "precise-restart-orphan-test"
	if !runner1.TryBeginPreciseRestart(runID) {
		t.Fatal("TryBeginPreciseRestart")
	}
	runner1.PublishProgress(runID, StartAllProgressEvent{
		Total: 3, Started: 1, Remaining: 2, Phase: "progress", Operation: "restart",
		Current: "svc", Done: false,
	})
	runner1.persistExecQueueFile()
	runner1.endPreciseRestart() // simulate process death without clearing file

	// Re-write file as if crash left current mid-flight (endPreciseRestart clears in-memory only).
	queuePath := filepath.Join(regDir, "exec_queue.json")
	data, _ := json.Marshal(ExecutionQueueSnapshot{
		Current: &ActiveBulkProgressSnapshot{
			Kind: "precise-restart", RunID: runID, Operation: "restart", Label: "精准编译重启",
			Event: &StartAllProgressEvent{
				Total: 3, Started: 1, Remaining: 2, Phase: "progress", Operation: "restart",
				Current: "svc", Done: false,
			},
		},
		Pending: []ExecQueuePendingItem{},
	})
	if err := os.WriteFile(queuePath, data, 0o644); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	runner2, err := NewRunner(cfg, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner reload: %v", err)
	}
	runner2.LoadExecQueueFromDisk()
	if runner2.IsPreciseRestartActive() {
		t.Fatal("precise-restart must not stay active after orphan reload")
	}
	if snap := runner2.ActiveBulkProgress(); snap != nil {
		t.Fatalf("ActiveBulkProgress = %#v, want nil", snap)
	}
	pending := runner2.ExecQueuePending()
	if len(pending) != 1 || pending[0].Type != "precise-restart" {
		t.Fatalf("pending = %#v", pending)
	}
	// New precise-restart must be allowed (no ghost lock).
	if !runner2.TryBeginPreciseRestart("precise-restart-new") {
		t.Fatal("TryBeginPreciseRestart should succeed after orphan cleared")
	}
	runner2.endPreciseRestart()
}

func TestPreciseRestartCancel_ClearsOrphanWithoutLifecycleHandle(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name:     "g1",
			Services: []Service{{Name: "svc", Command: "true"}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := "precise-restart-ghost"
	runner.preciseRestartActive.mu.Lock()
	runner.preciseRestartActive.active = true
	runner.preciseRestartActive.runID = runID
	runner.preciseRestartActive.mu.Unlock()
	runner.PublishProgress(runID, StartAllProgressEvent{
		Total: 2, Started: 1, Remaining: 1, Phase: "progress", Operation: "restart", Current: "svc",
	})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/precise-restart/cancel", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "cleared" && body["status"] != "cancelled" {
		t.Fatalf("body = %#v", body)
	}
	if runner.IsPreciseRestartActive() {
		t.Fatal("orphan precise-restart still active after cancel")
	}
	if snap := runner.ActiveBulkProgress(); snap != nil {
		t.Fatalf("ActiveBulkProgress after cancel = %#v", snap)
	}
}

func TestExecQueue_IdlePersistRemovesFile(t *testing.T) {
	regDir := t.TempDir()
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(regDir, "reg.txt"))
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetExecQueuePending([]ExecQueuePendingItem{{
		ID: "q-tmp", Type: "stop-all", Label: "全部关闭",
	}})
	queuePath := filepath.Join(regDir, "exec_queue.json")
	if _, err := os.Stat(queuePath); err != nil {
		t.Fatalf("expected file: %v", err)
	}
	runner.SetExecQueuePending(nil)
	if _, err := os.Stat(queuePath); !os.IsNotExist(err) {
		t.Fatalf("idle persist should remove %s, err=%v", queuePath, err)
	}
}

func TestExecQueue_SkipPersistWhenPathNotAbsolute(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetExecQueuePending([]ExecQueuePendingItem{{
		ID: "q-rel", Type: "start-all", Label: "全部启动",
	}})
	rel := filepath.Join(cwd, ".runall", "exec_queue.json")
	if _, err := os.Stat(rel); !os.IsNotExist(err) {
		t.Fatalf("relative persist wrote %s, err=%v", rel, err)
	}
}
