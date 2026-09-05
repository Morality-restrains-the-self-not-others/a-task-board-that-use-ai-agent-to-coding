package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestUIHomePage_BuildConflictAttachesProgress 回归：409「已有全部/分组重新编译进行中」
// 必须接上进度条/SSE，且不得把队列项标失败（否则横幅有、进度条无）。
func TestUIHomePage_BuildConflictAttachesProgress(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, snippet := range []string{
		`async function adoptInProgressBulkOp(`,
		`await adoptInProgressBulkOp('build'`,
		`connectBuildAllSSE()`,
		`id="start-all-progress-panel"`,
		`await this.syncPendingToServer()`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status page missing snippet %q", snippet)
		}
	}
	if strings.Contains(body, "if (_bulkProgressResumed) return") {
		t.Fatal("resumeActiveBulkProgress still one-shot; SSE 断开后无法再挂上进度条")
	}
	// 409 路径不得在 adopt 之前把队列标失败。
	idx := strings.Index(body, "已有全部/分组重新编译进行中")
	if idx < 0 {
		t.Fatal("missing rebuild-in-progress banner copy")
	}
	window := body[idx:]
	if len(window) > 800 {
		window = window[:800]
	}
	if strings.Contains(window, "_onBulkProgressDone = null") &&
		!strings.Contains(window, "adoptInProgressBulkOp") {
		t.Fatal("409 rebuild path still fails the exec queue without attaching progress")
	}
}

func TestAPIBuildAll_ConflictIncludesActiveProgress(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-build"})
	store.Update("svc-build", StatusStopped, "")
	dir := t.TempDir()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:         "svc-build",
				Command:      "true",
				BuildCommand: "sleep 2",
				WorkingDir:   dir,
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`)))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want 202, body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	mux.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`)))
	if second.Code != http.StatusConflict {
		t.Fatalf("second status = %d, want 409, body=%s", second.Code, second.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode 409: %v body=%s", err, second.Body.String())
	}
	raw, ok := payload["active_bulk_progress"].(map[string]any)
	if !ok {
		t.Fatalf("409 missing active_bulk_progress: %#v", payload)
	}
	if raw["kind"] != "build-all" {
		t.Fatalf("kind = %v, want build-all", raw["kind"])
	}
	if strings.TrimSpace(strVal(raw["run_id"])) == "" {
		t.Fatalf("run_id empty: %#v", raw)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runner.GetActiveBuildAllRunID() == "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("build-all did not finish clearing active run id")
}

func TestConsumePendingExecQueueType(t *testing.T) {
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetExecQueuePending([]ExecQueuePendingItem{
		{ID: "q1", Type: "build-all", Label: "全部重新编译"},
		{ID: "q2", Type: "start-all", Label: "全部启动"},
	})
	if !runner.ConsumePendingExecQueueType("build-all") {
		t.Fatal("expected consume to remove build-all")
	}
	left := runner.ExecQueuePending()
	if len(left) != 1 || left[0].Type != "start-all" {
		t.Fatalf("pending after consume = %#v", left)
	}
	if runner.ConsumePendingExecQueueType("build-all") {
		t.Fatal("second consume should be false")
	}
}

func strVal(v any) string {
	s, _ := v.(string)
	return s
}
