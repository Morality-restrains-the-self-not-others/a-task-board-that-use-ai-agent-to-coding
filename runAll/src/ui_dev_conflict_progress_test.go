package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"runAll/src/domain"
)

// OPT-20260820-007: 清库/初始化 409 必须带回 active_bulk_progress（含 kind/run_id），
// 前端才能 adoptInProgressBulkOp 挂上进度条/SSE，而不是只显示「已在进行中」横幅。

func TestDevDatabaseHandler_ConflictIncludesActiveProgress(t *testing.T) {
	runner, _ := newTestRunnerForDevDB(t)

	started := make(chan struct{})
	release := make(chan struct{})
	mux := http.NewServeMux()
	registerDevDatabaseHandler(mux, runner, "/api/dev/init-databases", "init-db", domain.ToolDbInit, "INIT_ALL", func(ctx context.Context) any {
		close(started)
		<-release
		return domain.DatabasePlatformInitResult{Status: "ok"}
	})

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/dev/init-databases?confirm=INIT_ALL", nil))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status=%d want 202, body=%s", first.Code, first.Body.String())
	}
	var accepted map[string]string
	_ = json.NewDecoder(first.Body).Decode(&accepted)
	runID := accepted["run_id"]
	if runID == "" {
		t.Fatal("first accept missing run_id")
	}

	<-started
	second := httptest.NewRecorder()
	mux.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/dev/init-databases?confirm=INIT_ALL", nil))
	if second.Code != http.StatusConflict {
		t.Fatalf("second status=%d want 409, body=%s", second.Code, second.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode 409: %v body=%s", err, second.Body.String())
	}
	raw, ok := payload["active_bulk_progress"].(map[string]any)
	if !ok {
		t.Fatalf("409 missing active_bulk_progress: %#v", payload)
	}
	if raw["kind"] != "init-db" {
		t.Fatalf("kind=%v want init-db", raw["kind"])
	}
	if rid, _ := raw["run_id"].(string); rid != runID {
		t.Fatalf("run_id=%q want %q", rid, runID)
	}

	close(release)
	// 等后台完成并释放 dev-db 锁，避免污染后续测试。
	last := waitDevDBDone(t, runner, runID)
	if last.Phase != "done" || !last.Done {
		t.Fatalf("final event phase=%q done=%v want phase=done", last.Phase, last.Done)
	}
}

func TestUIHomePage_DevDBConflictAttachesProgress(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, snippet := range []string{
		`await adoptInProgressBulkOp('init-db'`,
		`await adoptInProgressBulkOp('clear-db'`,
		`_openDevDBProgressSSE(devSnap.run_id`,
		`初始化数据库已在进行中，请等待完成后再试`,
		`清库已在进行中，请等待完成后再试`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status page missing snippet %q", snippet)
		}
	}
	// 409 路径必须在 adopt 之后、finalizeDevDBProgress 之前 return（否则进度面板被标失败）。
	for _, pair := range [][2]string{
		{"await adoptInProgressBulkOp('init-db'", "finalizeDevDBProgress('init-db'"},
		{"await adoptInProgressBulkOp('clear-db'", "finalizeDevDBProgress('clear-db'"},
	} {
		adoptIdx := strings.Index(body, pair[0])
		finIdx := strings.Index(body, pair[1])
		if adoptIdx < 0 {
			t.Fatalf("status page missing adopt snippet %q", pair[0])
		}
		if finIdx < 0 {
			t.Fatalf("status page missing finalize snippet %q", pair[1])
		}
		if adoptIdx > finIdx {
			t.Fatalf("%s coded after %s — 409 path would finalize before adopting", pair[0], pair[1])
		}
		if !strings.Contains(body[adoptIdx:finIdx], "return") {
			t.Fatalf("%s branch must return before %s", pair[0], pair[1])
		}
	}
}
