package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestartAllWithActor_StopThenStart(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-infra", Services: []Service{{
				Name:        "svc-ra-infra",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			}}},
			{Name: "g-app", Services: []Service{{
				Name:        "svc-ra-app",
				DependsOn:   []string{"svc-ra-infra"},
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
	store.Update("svc-ra-infra", StatusStopped, "")
	store.Update("svc-ra-app", StatusStopped, "")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := runner.RestartAllWithActor(ctx, "owner-session"); err != nil {
		t.Fatalf("RestartAllWithActor: %v", err)
	}

	waitForServiceStatus(t, store, "svc-ra-infra", StatusHealthy, 5*time.Second)
	waitForServiceStatus(t, store, "svc-ra-app", StatusHealthy, 5*time.Second)
	if runner.IsRestartAllActive() {
		t.Fatal("expected restart-all to be idle after completion")
	}
}

func TestTryBeginRestartAll_RejectsConcurrent(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if !runner.TryBeginRestartAll() {
		t.Fatal("first TryBeginRestartAll should succeed")
	}
	if runner.TryBeginRestartAll() {
		t.Fatal("second TryBeginRestartAll should fail")
	}
	runner.endRestartAll()
	if !runner.TryBeginRestartAll() {
		t.Fatal("TryBeginRestartAll should succeed after end")
	}
	runner.endRestartAll()
}

// ADR-0027: 全部重启只拉 last-good，即使配置了会失败的 build_command 也不得编译。
func TestRestartAll_DoesNotRunBuildCommand(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	dir := t.TempDir()
	buildMarker := filepath.Join(dir, "built.marker")

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-app", Services: []Service{{
				Name:         "svc-ra-nobuild",
				Command:      "sleep 30",
				BuildCommand: "touch " + buildMarker + " && exit 9",
				WorkingDir:   dir,
				HealthCheck:  HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	runner.progressBroadcaster = NewProgressBroadcaster()
	store.Update("svc-ra-nobuild", StatusStopped, "")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := runner.RestartAllWithActor(ctx, "owner-session"); err != nil {
		t.Fatalf("RestartAllWithActor: %v", err)
	}
	waitForServiceStatus(t, store, "svc-ra-nobuild", StatusHealthy, 5*time.Second)
	if _, err := os.Stat(buildMarker); err == nil {
		t.Fatal("restart-all must not run build_command")
	}
}
