package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"runAll/src/domain"
)

// newTestRunnerForDevDB builds a minimal Runner usable by the dev database
// handlers, with MONOREPO_ROOT pinned to a temp dir so the .runall/ status
// marker never touches the real repo.
func newTestRunnerForDevDB(t *testing.T) (*Runner, string) {
	t.Helper()
	t.Setenv("RUNALL_ALLOW_DEV_DB_RESET", "1")
	monoRoot := t.TempDir()
	t.Setenv("MONOREPO_ROOT", monoRoot)
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "svc-db-ctx", Command: "echo ok", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	return runner, monoRoot
}

// waitDevDBDone subscribes to the run's progress and returns the final event
// (or fails the test on timeout). The channel is closed by CloseProgressRun
// 30s later, so we break on the Done event instead of ranging to EOF.
func waitDevDBDone(t *testing.T, runner *Runner, runID string) StartAllProgressEvent {
	t.Helper()
	ch := runner.SubscribeProgress(runID)
	var last StartAllProgressEvent
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return last
			}
			last = ev
			if ev.Done {
				return last
			}
		case <-timeout:
			t.Fatalf("timeout waiting for progress done (run_id=%s)", runID)
		}
	}
}

// Regression (OPT-20260812-049): clear/init now returns HTTP 202 accepted with a
// run_id immediately and executes in the background; the full result is delivered
// via /api/progress SSE Detail. The dev database reset lock is released after the
// background operation completes, and .runall/db-<op>.last_status records the outcome.
func TestDevDatabaseHandler_AsyncAcceptedAndProgressDetail(t *testing.T) {
	runner, monoRoot := newTestRunnerForDevDB(t)

	started := make(chan struct{})
	release := make(chan struct{})
	mux := http.NewServeMux()
	registerDevDatabaseHandler(mux, runner, "/api/dev/init-databases", "init-db", domain.ToolDbInit, "INIT_ALL", func(ctx context.Context) any {
		// Deterministically hold the dev db reset lock until the test releases it,
		// so the concurrent 409 assertion is race-free.
		close(started)
		<-release
		return domain.DatabasePlatformInitResult{
			Status: "ok",
			Migrations: []domain.DatabaseStepResult{
				{Database: "task_auth", Status: "ok"},
			},
			Inits: []domain.DatabaseStepResult{
				{Database: "task_auth", Status: "ok"},
			},
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/dev/init-databases?confirm=INIT_ALL", nil)
	req.Host = "127.0.0.1:9999"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202, body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode accept: %v", err)
	}
	if resp["status"] != "accepted" || resp["run_id"] == "" {
		t.Fatalf("bad accept response: %v", resp)
	}
	runID := resp["run_id"]

	// Handler must have returned while the operation is still running:
	// a second POST while the first is in flight should 409.
	<-started
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/dev/init-databases?confirm=INIT_ALL", nil)
	req2.Host = "127.0.0.1:9999"
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("concurrent status=%d want 409, body=%s", rec2.Code, rec2.Body.String())
	}
	close(release) // let the background init finish

	last := waitDevDBDone(t, runner, runID)
	if last.Phase != "done" || last.Done != true {
		t.Fatalf("final event phase=%q done=%v want phase=done", last.Phase, last.Done)
	}
	detail, ok := last.Detail.(domain.DatabasePlatformInitResult)
	if !ok {
		t.Fatalf("Detail type=%T want DatabasePlatformInitResult", last.Detail)
	}
	if detail.Status != "ok" || len(detail.Migrations) != 1 {
		t.Fatalf("Detail mismatch: %+v", detail)
	}

	// Lock released after background completion → a fresh request can start.
	if !runner.TryAcquireDevDatabaseReset() {
		t.Fatalf("dev db reset lock not released after background completion")
	}
	runner.ReleaseDevDatabaseReset()

	// .runall/db-init.last_status must be written with the terminal status.
	raw, err := os.ReadFile(filepath.Join(monoRoot, ".runall", "db-init.last_status"))
	if err != nil {
		t.Fatalf("read last_status: %v", err)
	}
	if got := string(raw); !containsLine(got, "status=ok") {
		t.Fatalf("last_status missing status=ok:\n%s", got)
	}
}

// Regression (nightly 2026-09-05): the dev db reset lock must already be released
// — not merely "a few instructions later" — before the terminal progress event is
// published. The lock used to be released by a defer at goroutine exit, i.e. after
// PublishProgress(done): a subscriber that observed Done could still see
// TryAcquireDevDatabaseReset() == false and a fresh clear/init would spuriously 409
// (the panic path released before publishing due to defer LIFO, but the
// normal/blocked path did not). devDBBeforeTerminalEventHook fires inside the
// producer goroutine at exactly the moment the terminal event is about to be
// published, making this ordering assertion deterministic rather than a scheduler race.
func TestDevDatabaseHandler_LockReleasedBeforeTerminalEvent(t *testing.T) {
	runner, _ := newTestRunnerForDevDB(t)

	var (
		mu                 sync.Mutex
		hookRan            bool
		lockFreeAtTerminal bool
	)
	prevHook := devDBBeforeTerminalEventHook
	devDBBeforeTerminalEventHook = func(r *Runner) {
		mu.Lock()
		defer mu.Unlock()
		hookRan = true
		// If the lock is still held here (pre-fix ordering released it only at
		// goroutine exit), the terminal Done event is about to be published while a
		// fresh clear/init would still 409 — the bug under test.
		lockFreeAtTerminal = r.TryAcquireDevDatabaseReset()
		if lockFreeAtTerminal {
			r.ReleaseDevDatabaseReset()
		}
	}
	t.Cleanup(func() { devDBBeforeTerminalEventHook = prevHook })

	mux := http.NewServeMux()
	registerDevDatabaseHandler(mux, runner, "/api/dev/init-databases", "init-db", domain.ToolDbInit, "INIT_ALL", func(ctx context.Context) any {
		return domain.DatabasePlatformInitResult{
			Status: "ok",
			Migrations: []domain.DatabaseStepResult{
				{Database: "task_auth", Status: "ok"},
			},
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/dev/init-databases?confirm=INIT_ALL", nil)
	req.Host = "127.0.0.1:9999"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202, body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode accept: %v", err)
	}
	last := waitDevDBDone(t, runner, resp["run_id"])
	if last.Phase != "done" || !last.Done {
		t.Fatalf("final event phase=%q done=%v want phase=done", last.Phase, last.Done)
	}

	mu.Lock()
	ran, free := hookRan, lockFreeAtTerminal
	mu.Unlock()
	if !ran {
		t.Fatalf("devDBBeforeTerminalEventHook did not fire before terminal publish")
	}
	if !free {
		t.Fatalf("dev db reset lock still held when terminal progress event is about to be published")
	}
}

// Regression (OPT-20260813-002 #3): a panic inside the clear/init path must be
// recovered — it must not crash runAll — and must surface as an error progress
// event plus a status=panic marker in .runall/.
func TestDevDatabaseHandler_PanicRecovered(t *testing.T) {
	runner, monoRoot := newTestRunnerForDevDB(t)

	mux := http.NewServeMux()
	registerDevDatabaseHandler(mux, runner, "/api/dev/clear-databases", "clear-db", domain.ToolDbClear, "CLEAR_ALL", func(ctx context.Context) any {
		panic("boom: simulated init panic")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/dev/clear-databases?confirm=CLEAR_ALL", nil)
	req.Host = "127.0.0.1:9999"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202, body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	last := waitDevDBDone(t, runner, resp["run_id"])
	if last.Phase != "error" || !last.Done {
		t.Fatalf("panic final event phase=%q done=%v want phase=error", last.Phase, last.Done)
	}
	if last.Error == "" {
		t.Fatalf("panic event must carry error detail")
	}

	// Lock released even after panic.
	if !runner.TryAcquireDevDatabaseReset() {
		t.Fatalf("dev db reset lock not released after panic")
	}
	runner.ReleaseDevDatabaseReset()

	raw, err := os.ReadFile(filepath.Join(monoRoot, ".runall", "db-clear.last_status"))
	if err != nil {
		t.Fatalf("read last_status: %v", err)
	}
	if got := string(raw); !containsLine(got, "status=panic") {
		t.Fatalf("last_status missing status=panic:\n%s", got)
	}
}

// Regression: init blocked by running services surfaces as an error progress
// event carrying the blocked-services message (was previously an inline 409).
func TestDevDatabaseHandler_BlockedByRunningServices(t *testing.T) {
	runner, _ := newTestRunnerForDevDB(t)

	mux := http.NewServeMux()
	registerDevDatabaseHandler(mux, runner, "/api/dev/init-databases", "init-db", domain.ToolDbInit, "INIT_ALL", func(ctx context.Context) any {
		return domain.DatabasePlatformInitResult{
			Status:          "blocked",
			BlockedServices: []string{"task-auth", "taskFE"},
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/dev/init-databases?confirm=INIT_ALL", nil)
	req.Host = "127.0.0.1:9999"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202, body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	last := waitDevDBDone(t, runner, resp["run_id"])
	if last.Phase != "error" || !last.Done {
		t.Fatalf("blocked final event phase=%q done=%v want phase=error", last.Phase, last.Done)
	}
	if last.Error == "" || !strings.Contains(last.Error, "task-auth") {
		t.Fatalf("blocked event error=%q must mention task-auth", last.Error)
	}
	if detail, ok := last.Detail.(domain.DatabasePlatformInitResult); !ok || detail.Status != "blocked" {
		t.Fatalf("Detail=%v want blocked init result", last.Detail)
	}
}

// Regression: client/proxy disconnect must not cancel clear/init mid-flight.
// The handler now returns 202 immediately and runs detached, so cancellation
// after the response can no longer reach the background operation.
func TestDevDatabaseHandler_IgnoresRequestCancel(t *testing.T) {
	runner, _ := newTestRunnerForDevDB(t)

	var (
		mu       sync.Mutex
		sawErr   error
		finished bool
	)
	mux := http.NewServeMux()
	registerDevDatabaseHandler(mux, runner, "/api/dev/clear-databases", "clear-db", domain.ToolDbClear, "CLEAR_ALL", func(ctx context.Context) any {
		// Simulate a slow wipe; request cancel happens while we wait.
		select {
		case <-time.After(150 * time.Millisecond):
		case <-ctx.Done():
			mu.Lock()
			sawErr = ctx.Err()
			mu.Unlock()
			return map[string]any{"status": "partial"}
		}
		mu.Lock()
		sawErr = ctx.Err()
		finished = true
		mu.Unlock()
		return map[string]any{"status": "ok"}
	})

	reqCtx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(reqCtx, http.MethodPost, "/api/dev/clear-databases?confirm=CLEAR_ALL", nil)
	req.Host = "127.0.0.1:9999"
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		mux.ServeHTTP(rec, req)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel() // client disconnect
	<-done

	// The handler must already have returned 202 (it never awaits the op).
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202, body=%s", rec.Code, rec.Body.String())
	}

	// The background operation still finishes with a detached (non-cancelled) ctx.
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		fin := finished
		saw := sawErr
		mu.Unlock()
		if fin {
			if saw != nil {
				t.Fatalf("detached ctx should not be canceled, got %v", saw)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background operation did not finish; sawErr=%v", saw)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func containsLine(haystack, needle string) bool {
	for _, line := range splitLines(haystack) {
		if line == needle {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}
