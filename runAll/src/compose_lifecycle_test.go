package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestIsDetachLaunchCommand(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{command: "bash run-infra.sh managed", want: true},
		{command: "bash run.sh managed", want: true},
		{command: "docker compose -f docker-compose.yml up -d", want: true},
		{command: "docker-compose up -d", want: true},
		{command: "sleep 30", want: false},
		{command: "docker compose up", want: false},
	}
	for _, tt := range tests {
		if got := isDetachLaunchCommand(tt.command); got != tt.want {
			t.Errorf("isDetachLaunchCommand(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

func TestService_IsDetachLaunch_PrefersLaunchMode(t *testing.T) {
	svc := Service{
		StartCommand: "echo hi",
		LaunchMode:   "detach",
	}
	if !svc.IsDetachLaunch() {
		t.Fatal("expected detach from launch_mode")
	}
	svc.LaunchMode = "attach"
	if svc.IsDetachLaunch() {
		t.Fatal("expected attach launch_mode to disable detach")
	}
}

func TestWaitHealthyWithLaunchCheck_AllowsDetachLaunchExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer cmd.Wait()

	err := waitHealthyWithLaunchCheck(context.Background(), cmd.Process, HealthCheck{
		URL:     srv.URL,
		Timeout: 5,
		Retries: 5,
		Backoff: Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.5},
	}, true)
	if err != nil {
		t.Fatalf("waitHealthyWithLaunchCheck(allowLaunchExit=true): %v", err)
	}
}

func TestWaitHealthyWithLaunchCheck_RejectsEarlyExitWithoutDetach(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	<-done

	err := waitHealthyWithLaunchCheck(context.Background(), cmd.Process, HealthCheck{
		URL:     "http://127.0.0.1:1/health",
		Timeout: 1,
		Retries: 1,
		Backoff: Backoff{Initial: 0.05, Max: 0.05, Multiplier: 1},
	}, false)
	if err == nil || !strings.Contains(err.Error(), "launch process exited before readiness") {
		t.Fatalf("expected launch exit error, got: %v", err)
	}
}

func TestWaitHealthyWithLaunchCheck_DetectsUnreapedZombieExit(t *testing.T) {
	// Reproduces saas-backend TabError path: bash exits before readiness but
	// cmd.Wait has not reaped yet, so Signal(0) still succeeds on the zombie.
	cmd := exec.Command("bash", "-c", "exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Do NOT Wait — leave a zombie so Signal(0)-only checks would falsely say "alive".
	time.Sleep(50 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("expected unreaped zombie to still accept Signal(0), got: %v", err)
	}

	err := waitHealthyWithLaunchCheck(context.Background(), cmd.Process, HealthCheck{
		URL:     "http://127.0.0.1:1/health",
		Timeout: 2,
		Retries: 5,
		Backoff: Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.5},
	}, false)
	if err == nil || !strings.Contains(err.Error(), "launch process exited before readiness") {
		t.Fatalf("expected launch exit error for zombie, got: %v", err)
	}
	if !strings.Contains(err.Error(), "exit=7") {
		t.Fatalf("expected exit=7 in error, got: %v", err)
	}
	if strings.Contains(err.Error(), "health check timed out") {
		t.Fatalf("must not misclassify as readiness timeout, got: %v", err)
	}
}
