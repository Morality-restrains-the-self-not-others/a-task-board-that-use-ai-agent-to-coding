package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"runAll/src/domain"
)

func TestRunner_IsServiceRunning_unreachableDespiteStaleHealthy(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"ai-monitor"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infra",
			Services: []Service{{
				Name: "ai-monitor",
				HealthCheck: HealthCheck{
					URL:     "http://127.0.0.1:1/",
					Timeout: 1,
					Retries: 1,
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("ai-monitor", StatusHealthy, "")
	store.SetReadiness("ai-monitor", ReadinessDegraded, "not ready")
	store.SetPID("ai-monitor", 0)

	if runner.IsServiceRunning("ai-monitor") {
		t.Fatal("expected unreachable healthy+degraded service to not count as running")
	}
}

func TestRunner_IsServiceRunning_unreachableDespiteFailed(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"ai-monitor"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infra",
			Services: []Service{{
				Name: "ai-monitor",
				HealthCheck: HealthCheck{
					URL:     "http://127.0.0.1:1/",
					Timeout: 1,
					Retries: 1,
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("ai-monitor", StatusFailed, "stopped manually")
	store.SetPID("ai-monitor", 0)

	if runner.IsServiceRunning("ai-monitor") {
		t.Fatal("expected unreachable failed service to not count as running")
	}
}

func TestRunner_IsServiceRunning_reachableHealthy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	defer listener.Close()

	store := NewStatusStore()
	store.Init([]string{"ai-monitor"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infra",
			Services: []Service{{
				Name: "ai-monitor",
				HealthCheck: HealthCheck{
					TCP:     addr,
					Timeout: 2,
					Retries: 1,
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("ai-monitor", StatusHealthy, "")
	store.SetPID("ai-monitor", 0)

	if !runner.IsServiceRunning("ai-monitor") {
		t.Fatal("expected reachable service to count as running")
	}
}

func TestRunner_IsServiceRunning_stoppedStatus(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"ai-monitor"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infra",
			Services: []Service{{
				Name:        "ai-monitor",
				HealthCheck: HealthCheck{URL: "http://127.0.0.1:3000/", Timeout: 1, Retries: 1},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("ai-monitor", StatusStopped, "")

	if runner.IsServiceRunning("ai-monitor") {
		t.Fatal("expected stopped status to not count as running")
	}
}

func TestRunner_IsServiceRunning_stoppedWithPortListeners(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"task-events-email-sent-1-send-email"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "domain-events",
			Services: []Service{{
				Name:        "task-events-email-sent-1-send-email",
				HealthCheck: HealthCheck{URL: "http://127.0.0.1:18022/api/health/", Timeout: 1, Retries: 1},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(port string) ([]int, error) {
		if port == "18022" {
			return []int{41414}, nil
		}
		return nil, nil
	}
	store.Update("task-events-email-sent-1-send-email", StatusStopped, "")
	store.SetPID("task-events-email-sent-1-send-email", 0)

	if !runner.IsServiceRunning("task-events-email-sent-1-send-email") {
		t.Fatal("expected stopped service with port listeners to count as running")
	}

	running := runner.RunningApplicationsExcept(domain.DevDatabaseClearExcludeServices)
	found := false
	for _, name := range running {
		if name == "task-events-email-sent-1-send-email" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("RunningApplicationsExcept = %#v, want task-events-email-sent", running)
	}
}

func TestRunner_IsServiceRunning_portProbeErrorConservative(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"probe-fail"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:        "probe-fail",
				HealthCheck: HealthCheck{URL: "http://127.0.0.1:18022/api/health/", Timeout: 1, Retries: 1},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, fmt.Errorf("lsof unavailable")
	}
	store.Update("probe-fail", StatusStopped, "")
	store.SetPID("probe-fail", 0)

	if !runner.IsServiceRunning("probe-fail") {
		t.Fatal("expected port probe error to conservatively count as running")
	}
}

func TestRunner_RunningApplicationsExcept_ignoresUnreachableStale(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"ai-monitor", "saas-backend"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infra",
			Services: []Service{
				{
					Name: "ai-monitor",
					HealthCheck: HealthCheck{
						URL:     "http://127.0.0.1:1/",
						Timeout: 1,
						Retries: 1,
					},
				},
				{
					Name: "saas-backend",
					HealthCheck: HealthCheck{
						URL:     "http://127.0.0.1:2/",
						Timeout: 1,
						Retries: 1,
					},
				},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("ai-monitor", StatusFailed, "cleanup")
	store.SetPID("ai-monitor", 0)
	store.Update("saas-backend", StatusStopped, "")

	running := runner.RunningApplicationsExcept(domain.DevDatabaseClearExcludeServices)
	if len(running) != 0 {
		t.Fatalf("running = %#v, want none", running)
	}
}

func TestRunner_IsServiceRunning_trackedProcess(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:        "svc",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: "http://127.0.0.1:9/", Timeout: 1, Retries: 1},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	cmd := exec.CommandContext(context.Background(), "sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	runner.mu.Lock()
	runner.processes["svc"] = cmd
	runner.mu.Unlock()
	store.Update("svc", StatusStopped, "")

	if !runner.IsServiceRunning("svc") {
		t.Fatal("expected tracked process to count as running")
	}
}

func TestRunner_IsServiceRunning_detachListenerOnPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	go func() {
		_ = http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}()
	defer func() {
		_ = listener.Close()
	}()
	waitForPortOpen(t, port, 2*time.Second)

	store := NewStatusStore()
	store.Init([]string{"detached"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name: "detached",
				HealthCheck: HealthCheck{
					URL:     fmt.Sprintf("http://127.0.0.1:%d/", port),
					Timeout: 2,
					Retries: 1,
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("detached", StatusHealthy, "")
	store.SetPID("detached", 0)

	if !runner.IsServiceRunning("detached") {
		t.Fatal("expected port listener to count as running")
	}
}
