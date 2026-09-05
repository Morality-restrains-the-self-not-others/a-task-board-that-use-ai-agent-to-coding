package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"
)

// ADR-0035: cancelling the UI run loop must not SIGTERM managed process groups.
func TestRun_UIMode_CancelKeepsManagedProcess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "platform",
			Services: []Service{{
				Name:    "keep-alive",
				Command: "sleep 120",
				HealthCheck: HealthCheck{
					URL:     srv.URL,
					Timeout: 5,
					Retries: 3,
					Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) { return nil, nil }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx, false) }()

	if err := runner.StartAllWithActor(context.Background(), "test-session"); err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	waitForServiceStatus(t, store, "keep-alive", StatusHealthy, 10*time.Second)

	runner.mu.Lock()
	cmd := runner.processes["keep-alive"]
	runner.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		t.Fatal("keep-alive process missing")
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { runner.stopProcess("keep-alive") })

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}

	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("managed pid %d must still be alive after orchestrator cancel: %v", pid, err)
	}
	if !runner.skipShutdownServices {
		t.Fatal("UI cancel must skip managed shutdown")
	}
}

// ADR-0035 hole: StdoutPipe children die SIGPIPE when the orchestrator exits.
// A child that keeps writing stdout must survive Run() cancel when stdio is a file.
func TestRun_UIMode_CancelKeepsStdoutWriter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	logRoot := t.TempDir()
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: logRoot},
		Groups: []Group{{
			Name: "platform",
			Services: []Service{{
				Name:    "stdout-writer",
				Command: "bash -c 'while true; do echo tick; sleep 0.05; done'",
				HealthCheck: HealthCheck{
					URL:     srv.URL,
					Timeout: 5,
					Retries: 3,
					Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) { return nil, nil }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx, false) }()

	if err := runner.StartAllWithActor(context.Background(), "test-session"); err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	waitForServiceStatus(t, store, "stdout-writer", StatusHealthy, 10*time.Second)

	runner.mu.Lock()
	cmd := runner.processes["stdout-writer"]
	runner.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		t.Fatal("stdout-writer process missing")
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { runner.stopProcess("stdout-writer") })

	logPath := runner.stdioLogPath("stdout-writer")
	if logPath == "" {
		t.Fatal("expected stdio log path")
	}
	stBefore, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat before cancel: %v", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}

	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("stdout-writer pid %d must still be alive after orchestrator cancel: %v", pid, err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st, err := os.Stat(logPath)
		if err == nil && st.Size() > stBefore.Size() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	stAfter, _ := os.Stat(logPath)
	size := int64(0)
	if stAfter != nil {
		size = stAfter.Size()
	}
	t.Fatalf("stdio log must keep growing after orchestrator exit, before=%d after=%d", stBefore.Size(), size)
}
