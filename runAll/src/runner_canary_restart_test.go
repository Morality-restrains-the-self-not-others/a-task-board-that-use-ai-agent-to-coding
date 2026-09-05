package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"runAll/src/domain"
)

func TestStartAndCheck_AllowOverlapStartDoesNotSkip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{{
				Name:    "overlap-svc",
				Command: "sleep 30",
				HealthCheck: HealthCheck{
					URL:           srv.URL,
					Timeout:       5,
					Retries:       8,
					CheckInterval: 1,
					Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
				},
			}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	oldReady := canaryPeerReadyTimeout
	canaryPeerReadyTimeout = 2 * time.Second
	t.Cleanup(func() { canaryPeerReadyTimeout = oldReady })

	runner.listenerPIDsFn = func(string) ([]int, error) {
		runner.mu.Lock()
		cmd := runner.processes["overlap-svc"]
		runner.mu.Unlock()
		if cmd != nil && cmd.Process != nil {
			return []int{cmd.Process.Pid}, nil
		}
		return []int{111001}, nil
	}
	t.Cleanup(func() { runner.stopProcess("overlap-svc") })

	ctx := t.Context()
	node := &ServiceNode{Service: *runner.findService("overlap-svc"), allowOverlapStart: true}
	if err := runner.startAndCheck(ctx, node); err != nil {
		t.Fatalf("startAndCheck overlap: %v", err)
	}
	runner.mu.Lock()
	_, started := runner.processes["overlap-svc"]
	runner.mu.Unlock()
	if !started {
		t.Fatal("allowOverlapStart must launch a peer process, not skip start")
	}
}

func TestPidInProcessGroup_Self(t *testing.T) {
	if !pidInProcessGroup(1, 1) {
		t.Fatal("pid==pgid")
	}
	if pidInProcessGroup(0, 1) {
		t.Fatal("zero pid")
	}
}

func overlapHealthCheck(url string) HealthCheck {
	return HealthCheck{
		URL:           url,
		Timeout:       5,
		Retries:       8,
		CheckInterval: 1,
		Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
	}
}

// Old listener still answers health while the new process never appears on
// the port (taskEvents run.sh exclusive preflight / pidfile). startAndCheck
// must not leave Retrying — that is what made restart fallback print
// "service is retrying, cannot start".
func TestStartAndCheck_OverlapPeerNeverListensLeavesFailedNotRetrying(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	const name = "overlap-false-health"
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups:  []Group{{Name: "g1", Services: []Service{{Name: name, Command: "sleep 30", HealthCheck: overlapHealthCheck(srv.URL)}}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	oldReady := canaryPeerReadyTimeout
	canaryPeerReadyTimeout = 300 * time.Millisecond
	t.Cleanup(func() { canaryPeerReadyTimeout = oldReady })

	runner.listenerPIDsFn = func(string) ([]int, error) {
		return []int{111001}, nil
	}
	t.Cleanup(func() { runner.stopProcess(name) })

	node := &ServiceNode{Service: *runner.findService(name), allowOverlapStart: true}
	err = runner.startAndCheck(t.Context(), node)
	if err == nil {
		t.Fatal("overlap with no peer listener must fail")
	}
	if strings.Contains(err.Error(), "cannot start") {
		t.Fatalf("overlap fail returned start gate error: %v", err)
	}
	st := store.Get(name)
	if st == nil {
		t.Fatal("missing status")
	}
	if st.Status == StatusRetrying {
		t.Fatalf("overlap false-healthy must not leave retrying (got %+v); restart fallback then cannot start", st)
	}
	if st.Status != StatusFailed {
		t.Fatalf("status = %s, want failed", st.Status)
	}
	if st.FailureCode != domain.ServiceFailureCodeReadinessTimeout {
		t.Fatalf("failure_code = %q, want %s", st.FailureCode, domain.ServiceFailureCodeReadinessTimeout)
	}
}

func TestStartAndCheck_OverlapSetsCanaryEnv(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	envFile := filepath.Join(t.TempDir(), "canary.env")
	const name = "overlap-canary-env"
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{{
			Name:        name,
			Command:     "printf '%s' \"$RUNALL_CANARY_OVERLAP\" > " + envFile + "; sleep 30",
			HealthCheck: overlapHealthCheck(srv.URL),
		}}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	oldReady := canaryPeerReadyTimeout
	canaryPeerReadyTimeout = 2 * time.Second
	t.Cleanup(func() { canaryPeerReadyTimeout = oldReady })
	runner.listenerPIDsFn = func(string) ([]int, error) {
		runner.mu.Lock()
		cmd := runner.processes[name]
		runner.mu.Unlock()
		if cmd != nil && cmd.Process != nil {
			return []int{cmd.Process.Pid}, nil
		}
		return []int{111001}, nil
	}
	t.Cleanup(func() { runner.stopProcess(name) })

	node := &ServiceNode{Service: *runner.findService(name), allowOverlapStart: true}
	if err := runner.startAndCheck(t.Context(), node); err != nil {
		t.Fatalf("startAndCheck overlap: %v", err)
	}
	got, readErr := os.ReadFile(envFile)
	if readErr != nil {
		t.Fatalf("read canary env file: %v", readErr)
	}
	if string(got) != "1" {
		t.Fatalf("RUNALL_CANARY_OVERLAP = %q, want 1 so taskEvents run.sh can start a peer", got)
	}
}

func TestStartAndCheck_ForceFreshStartFromRetryingDoesNotCannotStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	const name = "retrying-force-fresh"
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:        name,
		Command:     "sleep 30",
		HealthCheck: overlapHealthCheck(srv.URL),
	}}}}}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{name})
	store.Update(name, StatusRetrying, "false-healthy old listener")
	t.Cleanup(func() { runner.stopProcess(name) })

	node := &ServiceNode{Service: *runner.findService(name), forceFreshStart: true}
	if err := runner.startAndCheck(t.Context(), node); err != nil {
		t.Fatalf("forceFreshStart from retrying: %v", err)
	}
	st := store.Get(name)
	if st == nil || st.Status != StatusHealthy {
		t.Fatalf("status = %+v, want healthy (retrying+forceFreshStart must not return cannot start)", st)
	}
}

func TestReclaimRestartAfterCanaryFallback_RetryingToRestarting(t *testing.T) {
	t.Parallel()
	store := NewStatusStore()
	store.Init([]string{"svc"})
	store.Update("svc", StatusRetrying, "peer never listened")
	reclaimRestartAfterCanaryFallback(store, "svc")
	st := store.Get("svc")
	if st == nil || st.Status != StatusRestarting {
		t.Fatalf("status = %+v, want restarting", st)
	}
}
