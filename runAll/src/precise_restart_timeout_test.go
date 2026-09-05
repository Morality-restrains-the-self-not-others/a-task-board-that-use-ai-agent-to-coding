package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestPreciseRestart_PerServiceTimeoutFailsAndContinues 回归：单服务 stop/build/start
// 若无限挂起，不得永久卡住进度面板；超时后记失败并保留登记以便重试。
func TestPreciseRestart_PerServiceTimeoutFailsAndContinues(t *testing.T) {
	old := preciseRestartPerServiceTimeout
	preciseRestartPerServiceTimeout = 400 * time.Millisecond
	t.Cleanup(func() { preciseRestartPerServiceTimeout = old })

	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 50 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })

	cfg := &Config{
		Groups: []Group{{Services: []Service{{
			Name:         "svc-precise-hang",
			BuildCommand: "sleep 60",
			Command:      "true",
			// Unreachable probe (connection refused) so stop-phase checkProbe returns fast.
			HealthCheck: HealthCheck{
				URL: "http://127.0.0.1:1/health", Timeout: 1, Retries: 1,
				CheckInterval: 1, Backoff: Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.5},
			},
		}}}},
	}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{"svc-precise-hang"})
	store.Update("svc-precise-hang", StatusStopped, "")

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"svc-precise-hang"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-timeout-test") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	start := time.Now()
	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("PreciseRestart took %v, want per-service timeout (~400ms) to bound it", elapsed)
	}
	if len(keep) != 1 || keep[0] != "svc-precise-hang" {
		t.Fatalf("keep = %v, want [svc-precise-hang]", keep)
	}
	st := store.Get("svc-precise-hang")
	if st == nil || st.Status == StatusHealthy {
		t.Fatalf("hung service should not be healthy, got %#v", st)
	}
}
