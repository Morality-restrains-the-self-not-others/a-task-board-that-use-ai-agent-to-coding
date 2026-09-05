package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// ADR-0027：精准编译重启在 build_command 失败时必须保留旧进程与登记。
func TestPreciseRestart_CompileFailureKeepsProcessAndRegistration(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	dir := t.TempDir()
	buildMarker := filepath.Join(dir, "built.marker")
	cfg := &Config{
		Groups: []Group{{Services: []Service{{
			Name:         "svc-compile-fail",
			BuildCommand: "touch " + buildMarker + " && exit 9",
			Command:      "sleep 60",
			WorkingDir:   dir,
			HealthCheck: HealthCheck{
				URL:           healthServer.URL,
				Timeout:       2,
				Retries:       2,
				CheckInterval: 1,
				Backoff:       Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
			},
		}}}},
	}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{"svc-compile-fail"})
	store.Update("svc-compile-fail", StatusHealthy, "")

	proc := exec.Command("sleep", "60")
	proc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := proc.Start(); err != nil {
		t.Fatalf("start old process: %v", err)
	}
	t.Cleanup(func() {
		runner.stopMonitoring("svc-compile-fail")
		_ = syscall.Kill(-proc.Process.Pid, syscall.SIGKILL)
	})
	runner.mu.Lock()
	runner.processes["svc-compile-fail"] = proc
	runner.mu.Unlock()

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"svc-compile-fail"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-compile-fail") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	keep, err := runner.PreciseRestart(ctx, "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 1 || keep[0] != "svc-compile-fail" {
		t.Fatalf("keep = %v, want [svc-compile-fail]", keep)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 1 || after[0] != "svc-compile-fail" {
		t.Fatalf("file after = %v, want [svc-compile-fail]", after)
	}
	if perr := syscall.Kill(proc.Process.Pid, 0); perr != nil {
		t.Errorf("old process must stay alive after compile failure: %v", perr)
	}
	if st := store.Get("svc-compile-fail"); st == nil || st.Status != StatusHealthy {
		t.Fatalf("status = %+v, want healthy", st)
	}
	if _, err := os.Stat(buildMarker); err != nil {
		t.Fatalf("compile-then-swap must still attempt build: %v", err)
	}
}
