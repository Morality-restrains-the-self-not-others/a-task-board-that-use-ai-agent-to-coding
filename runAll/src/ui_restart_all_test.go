package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"runAll/src/infrastructure"
)

func TestAPIRestartAll_Accepted(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-infra", Services: []Service{{
				Name:        "svc-restart-all-infra",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			}}},
			{Name: "g-app", Services: []Service{{
				Name:        "svc-restart-all-app",
				DependsOn:   []string{"svc-restart-all-infra"},
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("svc-restart-all-infra", StatusStopped, "")
	store.Update("svc-restart-all-app", StatusStopped, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/restart-all", strings.NewReader(`{"session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "accepted" {
		t.Fatalf("status body = %#v, want status=accepted", body)
	}
	if body["run_id"] == "" {
		t.Fatal("expected run_id in accepted response")
	}

	waitForServiceStatus(t, store, "svc-restart-all-infra", StatusHealthy, 8*time.Second)
	waitForServiceStatus(t, store, "svc-restart-all-app", StatusHealthy, 8*time.Second)
}

func TestAPIRestartAll_RequiresSessionID(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/restart-all", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusAccepted {
		t.Fatalf("expected rejection without session_id, got %d", rec.Code)
	}
}

func TestAPIRestartAll_ConflictWhenActive(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if !runner.TryBeginRestartAll() {
		t.Fatal("TryBeginRestartAll")
	}
	t.Cleanup(runner.endRestartAll)

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/restart-all", strings.NewReader(`{"session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// OPT-20260901-001: on a fresh clone-run with an empty MySQL volume the
// registry has unapplied dataMigrate SQL; bulk restart must be refused before
// stopping anything so the gap surfaces as「初始化全部数据库」, not as per-service
// LAUNCH_PROCESS_EXITED.
func TestRestartAllWithActor_BlockedByPendingMigrations(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	monoRoot := t.TempDir()
	t.Setenv("MONOREPO_ROOT", monoRoot)
	writeMiniMigrateRepo(t, monoRoot)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{
			{Name: "g-app", Services: []Service{{
				Name:        "svc-ra-migblock",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	runner.progressBroadcaster = NewProgressBroadcaster()
	runner.migrateAppliedReader = staticAppliedReader{"001_schema.sql"} // 002_new.sql unapplied
	runner.migrateDirResolver = infrastructure.FilesystemDataMigrateDirResolver{}
	runner.migrateSQLLister = infrastructure.FilesystemSQLLister{}
	store.Update("svc-ra-migblock", StatusStopped, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = runner.RestartAllWithActor(ctx, "owner-session")
	if err == nil {
		t.Fatal("expected restart-all to be blocked when dataMigrate SQL is unapplied")
	}
	if !strings.Contains(err.Error(), "初始化全部数据库") {
		t.Fatalf("block message must point at initialize-all, got: %v", err)
	}
	// StartService must not have been invoked: the service stays stopped even
	// though the health server is live (a started service would reach healthy).
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if st := store.Get("svc-ra-migblock"); st != nil && st.Status == StatusHealthy {
			t.Fatal("StartService must not run when migrations are pending")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestAPIRestartAll_BlockedByPendingMigrations(t *testing.T) {
	monoRoot := t.TempDir()
	t.Setenv("MONOREPO_ROOT", monoRoot)
	writeMiniMigrateRepo(t, monoRoot)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{
			{Name: "g-app", Services: []Service{{
				Name: "svc-ra-migblock-api", Command: "echo ok",
			}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.migrateAppliedReader = staticAppliedReader{"001_schema.sql"}
	runner.migrateDirResolver = infrastructure.FilesystemDataMigrateDirResolver{}
	runner.migrateSQLLister = infrastructure.FilesystemSQLLister{}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/restart-all", strings.NewReader(`{"session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "初始化全部数据库") {
		t.Fatalf("body must point at initialize-all: %s", rec.Body.String())
	}
}

func TestAPIRestartAll_ProceedsWhenNoPendingMigrations(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	monoRoot := t.TempDir()
	t.Setenv("MONOREPO_ROOT", monoRoot)
	writeMiniMigrateRepo(t, monoRoot)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{
			{Name: "g-app", Services: []Service{{
				Name:        "svc-ra-mignone",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	runner.progressBroadcaster = NewProgressBroadcaster()
	// All local SQL applied -> no pending migrations -> restart-all accepted.
	runner.migrateAppliedReader = staticAppliedReader{"001_schema.sql", "002_new.sql"}
	runner.migrateDirResolver = infrastructure.FilesystemDataMigrateDirResolver{}
	runner.migrateSQLLister = infrastructure.FilesystemSQLLister{}
	store.Update("svc-ra-mignone", StatusStopped, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/restart-all", strings.NewReader(`{"session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d want 202, body=%s", rec.Code, rec.Body.String())
	}
	waitForServiceStatus(t, store, "svc-ra-mignone", StatusHealthy, 8*time.Second)
}

func TestStatusHTML_IncludesRestartAllControls(t *testing.T) {
	data, err := statusHTML.ReadFile("status.html")
	if err != nil {
		t.Fatalf("read status.html: %v", err)
	}
	html := string(data)
	for _, snippet := range []string{
		`id="restart-all-btn"`,
		`全部重启`,
		`function restartAllServices()`,
		`/api/restart-all`,
		`/api/restart-all/cancel`,
		`/api/restart-all/progress`,
		`function connectRestartAllSSE()`,
		`全部重启进度`,
		`金丝雀平滑重启`,
	} {
		if !strings.Contains(html, snippet) {
			t.Fatalf("status.html missing restart-all snippet %q", snippet)
		}
	}
}
