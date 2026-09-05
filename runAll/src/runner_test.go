package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

type runtimePrereqProbeRepositoryStub struct {
	err error
}

func (s runtimePrereqProbeRepositoryStub) Probe(service domain.ManagedService) error {
	_ = service
	return s.err
}

type deleteOwnershipFailingRepositoryStub struct {
	delegate  domain.ServiceOwnershipRepository
	deleteErr error
}

func (s deleteOwnershipFailingRepositoryStub) FindByServiceName(serviceName string) (domain.ServiceOwnership, error) {
	return s.delegate.FindByServiceName(serviceName)
}

func (s deleteOwnershipFailingRepositoryStub) Save(ownership domain.ServiceOwnership) error {
	return s.delegate.Save(ownership)
}

func (s deleteOwnershipFailingRepositoryStub) DeleteByServiceName(serviceName string) error {
	_ = serviceName
	return s.deleteErr
}

func (s deleteOwnershipFailingRepositoryStub) ListAll() ([]domain.ServiceOwnership, error) {
	return s.delegate.ListAll()
}

func stubNoPortListenersForTest(runner *Runner) {
	if runner == nil {
		return
	}
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, nil
	}
}

func TestRunBuild_Success(t *testing.T) {
	runner := &Runner{}
	svc := &Service{
		Name:         "test-build",
		BuildCommand: "echo built",
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc, svc.BuildCommand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBuild_CommandFailed(t *testing.T) {
	runner := &Runner{}
	svc := &Service{
		Name:         "test-build-fail",
		BuildCommand: "exit 1",
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc, svc.BuildCommand)
	if err == nil {
		t.Fatal("expected error for failed build command")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Errorf("error should mention build failed, got: %v", err)
	}
}

func TestRunBuild_WithWorkingDir(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "built.txt")

	runner := &Runner{}
	svc := &Service{
		Name:         "test-build-dir",
		BuildCommand: "touch " + marker,
		WorkingDir:   dir,
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc, svc.BuildCommand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Errorf("build should have created marker file: %v", statErr)
	}
}

func TestRunBuild_WithEnv(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "env-out.txt")

	runner := &Runner{}
	svc := &Service{
		Name:         "test-build-env",
		BuildCommand: "printf '%s' \"$MY_VAR\" > " + marker,
		Env:          map[string]string{"MY_VAR": "hello"},
		WorkingDir:   dir,
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc, svc.BuildCommand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("could not read marker file: %v", readErr)
	}
	if string(data) != "hello" {
		t.Errorf("env output = %q, want %q", string(data), "hello")
	}
}

func TestRunBuild_ContextCanceled(t *testing.T) {
	runner := &Runner{}
	svc := &Service{
		Name:         "test-build-cancel",
		BuildCommand: "sleep 10",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := runner.runBuild(ctx, svc, svc.BuildCommand)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestRunBuild_AppendsStructuredLogs(t *testing.T) {
	repo := infrastructure.NewInMemoryServiceLogRepository(100)
	runner := &Runner{logRepository: repo}
	svc := &Service{
		Name:         "test-build-logs",
		BuildCommand: "echo out-line; echo err-line 1>&2",
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc, svc.BuildCommand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// runBuild must append the build output to the log repository before
	// returning. This used to race: cmd.Wait() closed the StdoutPipe/StderrPipe
	// read ends before the streamOutput goroutines were scheduled, silently
	// dropping the build logs (nightly failure: "output read error: read |0:
	// file already closed", "expected at least 2 log lines, got 0").
	logs := repo.Tail("test-build-logs", 10)
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 log lines immediately after runBuild, got %d", len(logs))
	}

	foundOut := false
	foundErr := false
	for _, line := range logs {
		if line.Message == "out-line" && line.Stream == domain.StreamStdout {
			foundOut = true
		}
		if line.Message == "err-line" && line.Stream == domain.StreamStderr {
			foundErr = true
		}
	}
	if !foundOut || !foundErr {
		t.Fatalf("structured logs missing expected entries: %#v", logs)
	}
}

func TestBuildService_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-only-ok",
					BuildCommand: "echo built",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9998"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("build-only-ok", StatusHealthy, "")
	err = runner.BuildService(context.Background(), "build-only-ok")
	if err != nil {
		t.Fatalf("BuildService: unexpected error: %v", err)
	}

	status := store.Get("build-only-ok")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", status)
	}
}

func TestBuildService_InferredFromRunScript(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "inferred-build",
					Command:     "bash run.sh",
					WorkingDir:  filepath.Join(repoRoot, "taskAuth"),
					HealthCheck: HealthCheck{URL: "http://localhost:8003/api/health/"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	inferred := resolveBuildCommand(Service{
		Command:    "bash run.sh",
		WorkingDir: filepath.Join(repoRoot, "taskAuth"),
	})
	if inferred == "" {
		t.Fatal("expected inferred build command for taskAuth run.sh")
	}

	store.Update("inferred-build", StatusHealthy, "")
	err = runner.BuildService(context.Background(), "inferred-build")
	if err != nil {
		t.Fatalf("BuildService inferred: %v", err)
	}

	status := store.Get("inferred-build")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", status)
	}
}

func TestBuildService_SuccessWhenStopped(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-only-stopped",
					BuildCommand: "echo built",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9994"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("build-only-stopped", StatusStopped, "")
	err = runner.BuildService(context.Background(), "build-only-stopped")
	if err != nil {
		t.Fatalf("BuildService: unexpected error: %v", err)
	}

	status := store.Get("build-only-stopped")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestBuildService_SuccessClearsFailedStatus(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-only-was-failed",
					BuildCommand: "echo built",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9993"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("build-only-was-failed", StatusFailed, "build failed: exit status 1")
	err = runner.BuildService(context.Background(), "build-only-was-failed")
	if err != nil {
		t.Fatalf("BuildService: unexpected error: %v", err)
	}

	status := store.Get("build-only-was-failed")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped after successful build from failed", status)
	}
	if status.Error != "" {
		t.Fatalf("error = %q, want empty after successful build", status.Error)
	}
}

func TestBuildService_Failure(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-only-fail",
					BuildCommand: "exit 7",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9997"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("build-only-fail", StatusHealthy, "")
	err = runner.BuildService(context.Background(), "build-only-fail")
	if err == nil {
		t.Fatal("expected build failure")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	status := store.Get("build-only-fail")
	if status == nil || status.Status != StatusFailed {
		t.Fatalf("status = %#v, want failed", status)
	}
}

func TestBuildService_NotFound(t *testing.T) {
	runner := &Runner{
		cfg:   &Config{},
		store: NewStatusStore(),
	}

	err := runner.BuildService(context.Background(), "missing-service")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildService_StatusConflict(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-conflict",
					BuildCommand: "echo built",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9996"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("build-conflict", StatusBuilding, "")
	err = runner.BuildService(context.Background(), "build-conflict")
	if err == nil {
		t.Fatal("expected status conflict error")
	}
	if !strings.Contains(err.Error(), "already building") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildService_StoppedDuringBuild_KeepsStoppedNotHealthy(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-stop-race",
					BuildCommand: "sleep 0.3",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9992"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("build-stop-race", StatusHealthy, "")

	go func() {
		time.Sleep(50 * time.Millisecond)
		store.Update("build-stop-race", StatusStopped, "")
	}()

	if err := runner.BuildService(context.Background(), "build-stop-race"); err != nil {
		t.Fatalf("BuildService: %v", err)
	}
	got := store.Get("build-stop-race")
	if got == nil || got.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped (not phantom healthy)", got)
	}
}

func TestBuildService_ConcurrentBuildRejected(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-concurrent",
					BuildCommand: "sleep 1",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9996"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("build-concurrent", StatusHealthy, "")

	firstErrCh := make(chan error, 1)
	go func() {
		firstErrCh <- runner.BuildService(context.Background(), "build-concurrent")
	}()

	waitForStatus(t, store, "build-concurrent", StatusBuilding, 2*time.Second)

	secondErr := runner.BuildService(context.Background(), "build-concurrent")
	if secondErr == nil {
		t.Fatal("expected second concurrent build to be rejected")
	}
	if !strings.Contains(secondErr.Error(), "already building") {
		t.Fatalf("unexpected second error: %v", secondErr)
	}

	firstErr := <-firstErrCh
	if firstErr != nil {
		t.Fatalf("first build should succeed, got: %v", firstErr)
	}
}

func TestRestartService_NoBuildCommand(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "echo-svc",
					Command:     "echo hello",
					HealthCheck: HealthCheck{URL: "http://localhost:9999"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("echo-svc", StatusHealthy, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "echo-svc")
	// The restart will likely fail at health check (no real server), but it
	// must NOT fail at the build step (there is no BuildCommand).
	if err != nil && strings.Contains(err.Error(), "build") {
		t.Errorf("restart without BuildCommand should not fail with a build error: %v", err)
	}
}

func TestRestartService_IgnoresBuildCommand(t *testing.T) {
	dir := t.TempDir()
	buildMarker := filepath.Join(dir, "built.marker")
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-ignored",
					BuildCommand: "touch " + buildMarker + " && exit 9",
					Command:      "echo running",
					WorkingDir:   dir,
					HealthCheck:  HealthCheck{URL: "http://localhost:9998"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("build-ignored", StatusHealthy, "")

	err = runner.RestartService(context.Background(), "build-ignored")
	if err != nil && strings.Contains(err.Error(), "build failed") {
		t.Fatalf("plain restart must not run build_command, got: %v", err)
	}
	if _, statErr := os.Stat(buildMarker); statErr == nil {
		t.Fatal("plain restart must not execute build_command")
	}
}

func TestRestartService_StoppedServiceAllowed(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "stopped-svc",
					Command:     "echo hi",
					HealthCheck: HealthCheck{URL: "http://localhost:9993"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("stopped-svc", StatusStopped, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "stopped-svc")
	if err != nil && strings.Contains(err.Error(), "can only restart") {
		t.Fatalf("stopped service should be restartable, got: %v", err)
	}
}

func TestRestartService_PendingServiceAllowed(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "pending-svc",
					Command:     "echo hi",
					HealthCheck: HealthCheck{URL: "http://localhost:9996"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("pending-svc", StatusPending, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "pending-svc")
	if err != nil && strings.Contains(err.Error(), "can only restart") {
		t.Fatalf("pending service should be restartable, got: %v", err)
	}
}

func TestRestartService_NonexistentService(t *testing.T) {
	store := NewStatusStore()
	runner := &Runner{cfg: &Config{}, store: store}

	ctx := context.Background()
	err := runner.RestartService(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent service")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRestartService_DoubleRestartRejected(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "busy-svc",
					Command:     "echo hi",
					HealthCheck: HealthCheck{URL: "http://localhost:9995"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("busy-svc", StatusRestarting, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "busy-svc")
	if err == nil {
		t.Fatal("expected error when restarting a service that is already restarting")
	}
}

func TestRestartService_DoubleRestartRejected_Building(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "building-svc",
					Command:     "echo hi",
					HealthCheck: HealthCheck{URL: "http://localhost:9994"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("building-svc", StatusBuilding, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "building-svc")
	if err == nil {
		t.Fatal("expected error when restarting a service that is building")
	}
}

// 旧 TestRestartService_KillFirst_BuildFailureMarksFailed 已被 ADR-0027 取代：
// 普通重启不编译；精准编译路径见 TestRestartService_CompileThenSwap_BuildFailureKeepsProcess。

func TestRunner_StopService_BlockedByActiveDownstreamDependency(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "db",
						Command:     "echo db",
						HealthCheck: HealthCheck{URL: "http://localhost:9101"},
					},
					{
						Name:        "api",
						Command:     "echo api",
						DependsOn:   []string{"db"},
						HealthCheck: HealthCheck{URL: "http://localhost:9102"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("db", StatusHealthy, "")
	store.Update("api", StatusHealthy, "")

	err = runner.StopService(context.Background(), "db")
	if err == nil {
		t.Fatal("expected stop to be blocked by active downstream dependency")
	}
	if !strings.Contains(err.Error(), "active downstream dependencies") || !strings.Contains(err.Error(), "api") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopService_NonOwnerDelegatesToOwner(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-stop",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9210"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-stop", StatusHealthy, "")

	ownership, err := domain.NewServiceOwnership(
		"owned-stop",
		"owner-session",
		1234,
		"config-hash",
		"http://localhost:9210",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	if got := runner.resolveCascadeStepActor("owned-stop", "other-session"); got != "owner-session" {
		t.Fatalf("resolveCascadeStepActor = %q, want owner-session", got)
	}
	err = runner.StopServiceWithActor(context.Background(), "owned-stop", "other-session")
	if err != nil {
		t.Fatalf("expected non-owner stop to delegate to owner, got: %v", err)
	}
	st := store.Get("owned-stop")
	if st == nil || st.Status != StatusStopped {
		t.Fatalf("status after delegated stop = %+v, want stopped", st)
	}
}

func TestStopServiceWithActor_ClearsOwnershipAfterStopped(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-stop-clear",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9213"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-stop-clear", StatusHealthy, "")
	store.SetPID("owned-stop-clear", 2233)

	ownership, err := domain.NewServiceOwnership(
		"owned-stop-clear",
		"owner-session",
		2233,
		"config-hash",
		"http://localhost:9213",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	if err := runner.StopServiceWithActor(context.Background(), "owned-stop-clear", "owner-session"); err != nil {
		t.Fatalf("StopServiceWithActor: %v", err)
	}

	cleared, err := runner.ownershipRepo.FindByServiceName("owned-stop-clear")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if cleared.ServiceName != "" || cleared.OwnerSessionID != "" || cleared.PID != 0 {
		t.Fatalf("ownership should be cleared after stop, got: %#v", cleared)
	}
}

func TestStopServiceWithActor_OwnershipCleanupFailureReturnsError(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-stop-cleanup-fail",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9214"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-stop-cleanup-fail", StatusHealthy, "")

	ownership, err := domain.NewServiceOwnership(
		"owned-stop-cleanup-fail",
		"owner-session",
		2233,
		"config-hash",
		"http://localhost:9214",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	runner.ownershipRepo = deleteOwnershipFailingRepositoryStub{
		delegate:  runner.ownershipRepo,
		deleteErr: errors.New("delete ownership boom"),
	}

	err = runner.StopServiceWithActor(context.Background(), "owned-stop-cleanup-fail", "owner-session")
	if err == nil {
		t.Fatal("expected stop to fail when ownership cleanup fails")
	}
	if !strings.Contains(err.Error(), "ownership cleanup failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	status := store.Get("owned-stop-cleanup-fail")
	if status == nil {
		t.Fatal("expected service status")
	}
	if status.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", status.Status, StatusFailed)
	}
	if !strings.Contains(status.Error, "ownership cleanup failed") {
		t.Fatalf("status error = %q, want ownership cleanup failed", status.Error)
	}
}

func TestStopCleanupFailure_TakeoverThenStartServiceWithActor_Recovers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "cleanup-fail-recover",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     srv.URL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("cleanup-fail-recover", StatusStopped, "")
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, nil
	}

	if err := runner.StartServiceWithActor(context.Background(), "cleanup-fail-recover", "owner-session"); err != nil {
		t.Fatalf("StartServiceWithActor(owner): %v", err)
	}

	originalOwnershipRepo := runner.ownershipRepo
	runner.ownershipRepo = deleteOwnershipFailingRepositoryStub{
		delegate:  originalOwnershipRepo,
		deleteErr: errors.New("delete ownership boom"),
	}

	err = runner.StopServiceWithActor(context.Background(), "cleanup-fail-recover", "owner-session")
	if err == nil {
		t.Fatal("expected stop to fail when ownership cleanup fails")
	}

	statusAfterStop := store.Get("cleanup-fail-recover")
	if statusAfterStop == nil || statusAfterStop.Status != StatusFailed {
		t.Fatalf("status after stop = %#v, want failed", statusAfterStop)
	}

	if err := runner.TakeoverService("cleanup-fail-recover", "other-session"); err != nil {
		t.Fatalf("TakeoverService(other): %v", err)
	}

	if err := runner.StartServiceWithActor(context.Background(), "cleanup-fail-recover", "other-session"); err != nil {
		t.Fatalf("StartServiceWithActor(other): %v", err)
	}

	statusAfterRecover := store.Get("cleanup-fail-recover")
	if statusAfterRecover == nil || statusAfterRecover.Status != StatusHealthy {
		t.Fatalf("status after recover = %#v, want healthy", statusAfterRecover)
	}

	runner.ownershipRepo = originalOwnershipRepo
	runner.ownershipGuard = domain.NewServiceOwnershipGuardService(originalOwnershipRepo)
	if err := runner.StopServiceWithActor(context.Background(), "cleanup-fail-recover", "other-session"); err != nil {
		t.Fatalf("cleanup StopServiceWithActor(other): %v", err)
	}
}

func TestRestartService_NonOwnerDelegatesToOwnerSession(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-restart",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9211"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-restart", StatusHealthy, "")

	ownership, err := domain.NewServiceOwnership(
		"owned-restart",
		"owner-session",
		1234,
		"config-hash",
		"http://localhost:9211",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	if got := runner.resolveCascadeStepActor("owned-restart", "other-session"); got != "owner-session" {
		t.Fatalf("resolveCascadeStepActor = %q, want owner-session", got)
	}
	// Ownership guard must accept the delegated owner session (no takeover required).
	if err := runner.ownershipGuard.EnsureOperableBySession("owned-restart", runner.resolveCascadeStepActor("owned-restart", "other-session")); err != nil {
		t.Fatalf("EnsureOperableBySession after delegate: %v", err)
	}
}

func TestTakeoverService_DiscoversListenerPIDWhenStorePIDMissing(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-takeover",
						Command:     "echo worker --port 9212",
						HealthCheck: HealthCheck{URL: "http://localhost:9212/health"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-takeover", StatusHealthy, "")
	store.SetPID("owned-takeover", 0)

	runner.listenerPIDsFn = func(port string) ([]int, error) {
		if port == "9212" {
			return []int{4321}, nil
		}
		return nil, nil
	}

	if err := runner.TakeoverService("owned-takeover", "takeover-session"); err != nil {
		t.Fatalf("TakeoverService: %v", err)
	}

	ownership, err := runner.ownershipRepo.FindByServiceName("owned-takeover")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if ownership.OwnerSessionID != "takeover-session" {
		t.Fatalf("owner session = %q, want takeover-session", ownership.OwnerSessionID)
	}
	if ownership.PID != 4321 {
		t.Fatalf("ownership pid = %d, want 4321", ownership.PID)
	}
}

func TestTakeoverService_StoppedServiceWithResidualOwnershipAllowsExplicitTakeover(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-stopped-takeover",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9215"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-stopped-takeover", StatusStopped, "")
	store.SetPID("owned-stopped-takeover", 0)

	ownership, err := domain.NewServiceOwnership(
		"owned-stopped-takeover",
		"owner-session",
		2233,
		"config-hash",
		"http://localhost:9215",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	if err := runner.TakeoverService("owned-stopped-takeover", "other-session"); err != nil {
		t.Fatalf("TakeoverService: %v", err)
	}

	takenOwnership, err := runner.ownershipRepo.FindByServiceName("owned-stopped-takeover")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if takenOwnership.OwnerSessionID != "other-session" {
		t.Fatalf("owner session = %q, want other-session", takenOwnership.OwnerSessionID)
	}
}

func TestTakeoverService_ThenStartServiceWithActor_AllowsRecoveryAfterStoppedResidualOwnership(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "stopped-recovery",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     srv.URL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("stopped-recovery", StatusStopped, "")
	store.SetPID("stopped-recovery", 0)
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, nil
	}

	ownership, err := domain.NewServiceOwnership(
		"stopped-recovery",
		"owner-session",
		2233,
		"config-hash",
		srv.URL,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	if err := runner.TakeoverService("stopped-recovery", "other-session"); err != nil {
		t.Fatalf("TakeoverService: %v", err)
	}

	if err := runner.StartServiceWithActor(context.Background(), "stopped-recovery", "other-session"); err != nil {
		t.Fatalf("StartServiceWithActor: %v", err)
	}

	status := store.Get("stopped-recovery")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", status)
	}

	takenOwnership, err := runner.ownershipRepo.FindByServiceName("stopped-recovery")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if takenOwnership.OwnerSessionID != "other-session" {
		t.Fatalf("owner session = %q, want other-session", takenOwnership.OwnerSessionID)
	}

	if err := runner.StopServiceWithActor(context.Background(), "stopped-recovery", "other-session"); err != nil {
		t.Fatalf("cleanup StopServiceWithActor: %v", err)
	}
}

func TestRunner_StopService_SetsStoppedStatus(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "worker",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9201"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("worker", StatusHealthy, "")

	if err := runner.StopService(context.Background(), "worker"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	status := store.Get("worker")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestRunner_StopService_AllowsRetryingStatus(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "retrying-worker",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9202"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("retrying-worker", StatusRetrying, "")

	if err := runner.StopService(context.Background(), "retrying-worker"); err != nil {
		t.Fatalf("StopService should allow retrying status, got: %v", err)
	}

	status := store.Get("retrying-worker")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestRunner_StopService_ClearsPID(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "pid-worker",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9203"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("pid-worker", StatusHealthy, "")
	store.SetPID("pid-worker", 12345)

	if err := runner.StopService(context.Background(), "pid-worker"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	status := store.Get("pid-worker")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.PID != 0 {
		t.Fatalf("pid = %d, want 0", status.PID)
	}
}

func TestRunner_StopService_RunsStopCommand(t *testing.T) {
	dir := t.TempDir()
	stopMarker := filepath.Join(dir, "stopped.marker")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infrastructure",
			Services: []Service{{
				Name:         "docker-kafka",
				StartCommand: "echo start",
				StopCommand:  fmt.Sprintf("touch %q", stopMarker),
				WorkingDir:   dir,
				HealthCheck:  HealthCheck{URL: healthURL, Timeout: 1, Retries: 1},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("docker-kafka", StatusHealthy, "")
	store.SetPID("docker-kafka", 12345)

	if err := runner.StopService(context.Background(), "docker-kafka"); err != nil {
		t.Fatalf("StopService: %v", err)
	}
	if _, err := os.Stat(stopMarker); err != nil {
		t.Fatalf("stop_command was not executed: %v", err)
	}
	status := store.Get("docker-kafka")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

// Regression: host systemd redis-server (user "redis") can keep :6379 open after
// docker-redis compose down. Non-root lsof sees no PIDs; stop must not fail with
// "still reachable after stop".
func TestRunner_StopService_DetachStopIgnoresInvisibleForeignReachability(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	tcpAddr := fmt.Sprintf("127.0.0.1:%d", port)

	dir := t.TempDir()
	stopMarker := filepath.Join(dir, "stopped.marker")
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "infrastructure",
			Services: []Service{{
				Name:         "docker-redis",
				StartCommand: "bash dockerInfra/redis/run.sh start",
				StopCommand:  fmt.Sprintf("touch %q", stopMarker),
				LaunchMode:   "detach",
				WorkingDir:   dir,
				HealthCheck: HealthCheck{
					TCP:     tcpAddr,
					Timeout: 1,
					Retries: 1,
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	// Simulate privileged foreign listener: probe succeeds, lsof returns nothing.
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, nil
	}

	store.Update("docker-redis", StatusHealthy, "")
	store.SetPID("docker-redis", 0)

	if err := runner.StopService(context.Background(), "docker-redis"); err != nil {
		t.Fatalf("StopService: %v", err)
	}
	if _, err := os.Stat(stopMarker); err != nil {
		t.Fatalf("stop_command was not executed: %v", err)
	}
	status := store.Get("docker-redis")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestRunner_StopService_StopsDetachedListenerByPortFallback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "detached-listener",
						Command: "echo managed-externally",
						HealthCheck: HealthCheck{
							URL:     healthURL,
							Timeout: 5,
							Retries: 10,
							Backoff: Backoff{
								Initial:    0.1,
								Max:        0.2,
								Multiplier: 1.2,
							},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	pid := startDetachedHTTPServer(t, port)
	defer func() {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}()
	waitForPortOpen(t, port, 5*time.Second)

	store.Update("detached-listener", StatusHealthy, "")
	store.SetPID("detached-listener", 0)

	if _, exists := runner.processes["detached-listener"]; exists {
		t.Fatal("test setup expects no tracked process entry")
	}

	if err := runner.StopService(context.Background(), "detached-listener"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	waitForPortClosed(t, port, 5*time.Second)
	status := store.Get("detached-listener")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestRunner_StopService_StopsStoppedStatusOrphanByPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:         "orphan-stopped",
				StartCommand: "echo start",
				StopCommand:  "true",
				HealthCheck: HealthCheck{
					URL:     healthURL,
					Timeout: 5,
					Retries: 3,
					Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.2},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	pid := startDetachedHTTPServer(t, port)
	defer func() {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}()
	waitForPortOpen(t, port, 5*time.Second)

	store.Update("orphan-stopped", StatusStopped, "")
	store.SetPID("orphan-stopped", 0)

	if err := runner.StopService(context.Background(), "orphan-stopped"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	waitForPortClosed(t, port, 5*time.Second)
	status := store.Get("orphan-stopped")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestRunner_StopService_detachStartCommandCleansPortListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	orphanPID := startDetachedHTTPServer(t, port)
	defer func() {
		_ = syscall.Kill(orphanPID, syscall.SIGKILL)
	}()
	waitForPortOpen(t, port, 5*time.Second)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:         "detach-orphan",
				StartCommand: "bash run.sh start email_sent/1_send_email",
				StopCommand:  "true",
				HealthCheck: HealthCheck{
					URL:     healthURL,
					Timeout: 5,
					Retries: 3,
					Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.2},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("detach-orphan", StatusStopped, "")
	store.SetPID("detach-orphan", 0)

	if err := runner.StopService(context.Background(), "detach-orphan"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	waitForPortClosed(t, port, 5*time.Second)
	status := store.Get("detach-orphan")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestRun_CleansForeignPortConflictBeforeStart(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	command := fmt.Sprintf("python3 -m http.server %d", port)
	foreignPID := startDetachedHTTPServer(t, port)
	defer func() {
		_ = syscall.Kill(foreignPID, syscall.SIGKILL)
	}()
	waitForPortOpen(t, port, 5*time.Second)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "conflict-svc",
						Command:     command,
						HealthCheck: HealthCheck{URL: healthURL},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Run(context.Background(), true)
	if err != nil {
		t.Fatalf("expected run to succeed after port cleanup, got: %v", err)
	}
	defer runner.Shutdown()

	waitForProcessExit(t, foreignPID, 5*time.Second)

	status := store.Get("conflict-svc")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusHealthy {
		t.Fatalf("status = %s, want %s", status.Status, StatusHealthy)
	}
	if status.PID <= 0 {
		t.Fatalf("status pid = %d, want positive", status.PID)
	}
	if status.PID == foreignPID {
		t.Fatalf("service should not reuse foreign pid %d", foreignPID)
	}
}

func TestRun_PreflightGenericErrorMarksServiceFailed(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "preflight-generic-fail",
						Command:     "sleep 30",
						HealthCheck: HealthCheck{URL: "http://127.0.0.1:65535/"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	runner.preflightFn = func(context.Context, Service) error {
		return errors.New("lsof probe failed")
	}

	err = runner.Run(context.Background(), true)
	if err == nil {
		t.Fatal("expected run to fail on generic preflight error")
	}
	if !strings.Contains(err.Error(), "lsof probe failed") {
		t.Fatalf("unexpected run error: %v", err)
	}

	status := store.Get("preflight-generic-fail")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", status.Status, StatusFailed)
	}
	if !strings.Contains(status.Error, "preflight failed") || !strings.Contains(status.Error, "lsof probe failed") {
		t.Fatalf("status error should contain diagnostic preflight failure, got: %q", status.Error)
	}
}

func TestRun_BlocksOnRuntimePrereqFailure(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "runtime-prereq-fail",
						Command:     "sleep 30",
						HealthCheck: HealthCheck{URL: "http://127.0.0.1:65535/"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	runner.runtimePrereqProbeRepository = runtimePrereqProbeRepositoryStub{
		err: errors.New("docker daemon unavailable"),
	}

	err = runner.Run(context.Background(), true)
	if err == nil {
		t.Fatal("expected run to fail on runtime prerequisite probe error")
	}
	if !strings.Contains(err.Error(), domain.ServiceFailureCodeRuntimePrereq) {
		t.Fatalf("error should contain %s, got: %v", domain.ServiceFailureCodeRuntimePrereq, err)
	}

	status := store.Get("runtime-prereq-fail")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", status.Status, StatusFailed)
	}
	if status.FailureCode != domain.ServiceFailureCodeRuntimePrereq {
		t.Fatalf("failure code = %q, want %q", status.FailureCode, domain.ServiceFailureCodeRuntimePrereq)
	}
}

func TestRun_SuccessEstablishesOwnership(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	command := fmt.Sprintf("python3 -m http.server %d", port)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "run-owned",
						Command: command,
						HealthCheck: HealthCheck{
							URL:     healthURL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer runner.Shutdown()

	if err := runner.Run(context.Background(), true); err != nil {
		t.Fatalf("Run: %v", err)
	}

	ownership, err := runner.ownershipRepo.FindByServiceName("run-owned")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if ownership.OwnerSessionID != defaultOwnershipSessionID {
		t.Fatalf("owner session = %q, want %q", ownership.OwnerSessionID, defaultOwnershipSessionID)
	}
	if ownership.PID <= 0 {
		t.Fatalf("ownership pid = %d, want positive", ownership.PID)
	}
}

func TestRun_RegressionMatrix(t *testing.T) {
	t.Run("port conflict cleaned", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			t.Fatalf("close listener: %v", err)
		}

		healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
		command := fmt.Sprintf("python3 -m http.server %d", port)
		foreignPID := startDetachedHTTPServer(t, port)
		defer func() {
			_ = syscall.Kill(foreignPID, syscall.SIGKILL)
		}()
		waitForPortOpen(t, port, 5*time.Second)

		store := NewStatusStore()
		runner, err := NewRunner(&Config{
			Version: "1",
			Groups: []Group{
				{
					Name: "matrix",
					Services: []Service{
						{
							Name:        "matrix-port-conflict",
							Command:     command,
							HealthCheck: HealthCheck{URL: healthURL},
						},
					},
				},
			},
		}, store)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}

		err = runner.Run(context.Background(), true)
		if err != nil {
			t.Fatalf("expected run to succeed after port cleanup, got: %v", err)
		}
		defer runner.Shutdown()
		waitForProcessExit(t, foreignPID, 5*time.Second)

		status := store.Get("matrix-port-conflict")
		if status == nil {
			t.Fatal("status should exist")
		}
		if status.Status != StatusHealthy {
			t.Fatalf("status = %s, want %s", status.Status, StatusHealthy)
		}
	})

	t.Run("runtime prereq blocked", func(t *testing.T) {
		store := NewStatusStore()
		runner, err := NewRunner(&Config{
			Version: "1",
			Groups: []Group{
				{
					Name: "matrix",
					Services: []Service{
						{
							Name:        "matrix-runtime-blocked",
							Command:     "sleep 30",
							HealthCheck: HealthCheck{URL: "http://127.0.0.1:65535/"},
						},
					},
				},
			},
		}, store)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}

		runner.runtimePrereqProbeRepository = runtimePrereqProbeRepositoryStub{
			err: errors.New("docker daemon unavailable"),
		}

		err = runner.Run(context.Background(), true)
		if err == nil {
			t.Fatal("expected run to fail on runtime prerequisite probe error")
		}
		if !strings.Contains(err.Error(), domain.ServiceFailureCodeRuntimePrereq) {
			t.Fatalf("error should contain %s, got: %v", domain.ServiceFailureCodeRuntimePrereq, err)
		}

		status := store.Get("matrix-runtime-blocked")
		if status == nil {
			t.Fatal("status should exist")
		}
		if status.FailureCode != domain.ServiceFailureCodeRuntimePrereq {
			t.Fatalf("failure code = %q, want %q", status.FailureCode, domain.ServiceFailureCodeRuntimePrereq)
		}
	})

	t.Run("prereq repaired healthy", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			t.Fatalf("close listener: %v", err)
		}
		healthURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
		command := fmt.Sprintf("python3 -m http.server %d", port)

		store := NewStatusStore()
		runner, err := NewRunner(&Config{
			Version: "1",
			Groups: []Group{
				{
					Name: "matrix",
					Services: []Service{
						{
							Name:        "matrix-prereq-repaired",
							Command:     command,
							HealthCheck: HealthCheck{URL: healthURL},
						},
					},
				},
			},
		}, store)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}
		defer runner.Shutdown()

		runner.runtimePrereqProbeRepository = runtimePrereqProbeRepositoryStub{
			err: errors.New("docker daemon unavailable"),
		}
		err = runner.Run(context.Background(), true)
		if err == nil {
			t.Fatal("expected first run to fail while runtime prereq is blocked")
		}

		runner.runtimePrereqProbeRepository = runtimePrereqProbeRepositoryStub{}
		err = runner.Run(context.Background(), true)
		if err != nil {
			t.Fatalf("expected second run to succeed after prereq repaired, got: %v", err)
		}

		status := store.Get("matrix-prereq-repaired")
		if status == nil {
			t.Fatal("status should exist")
		}
		if status.Status != StatusHealthy {
			t.Fatalf("status = %s, want %s", status.Status, StatusHealthy)
		}
		if status.FailureCode != "" {
			t.Fatalf("failure code = %q, want empty after successful rerun", status.FailureCode)
		}
	})

	t.Run("non-owner rejected", func(t *testing.T) {
		store := NewStatusStore()
		runner, err := NewRunner(&Config{
			Version: "1",
			Groups: []Group{
				{
					Name: "matrix",
					Services: []Service{
						{
							Name:        "matrix-owned",
							Command:     "echo worker",
							HealthCheck: HealthCheck{URL: "http://localhost:9311"},
						},
					},
				},
			},
		}, store)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}
		store.Update("matrix-owned", StatusHealthy, "")

		ownership, err := domain.NewServiceOwnership(
			"matrix-owned",
			"owner-session",
			2233,
			"config-hash",
			"http://localhost:9311",
			time.Now(),
		)
		if err != nil {
			t.Fatalf("NewServiceOwnership: %v", err)
		}
		if err := runner.ownershipRepo.Save(ownership); err != nil {
			t.Fatalf("Save ownership: %v", err)
		}

		if got := runner.resolveCascadeStepActor("matrix-owned", "other-session"); got != "owner-session" {
			t.Fatalf("resolveCascadeStepActor = %q, want owner-session", got)
		}
		err = runner.StopServiceWithActor(context.Background(), "matrix-owned", "other-session")
		if err != nil {
			t.Fatalf("expected non-owner stop to delegate to owner, got: %v", err)
		}
	})
}

func TestRunner_StartService_FromStopped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "startable",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:           srv.URL,
							Timeout:       2,
							Retries:       2,
							CheckInterval: 1,
							Backoff:       Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("startable", StatusStopped, "")

	if err := runner.StartService(context.Background(), "startable"); err != nil {
		t.Fatalf("StartService: %v", err)
	}

	got := store.Get("startable")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", got)
	}
	ownership, err := runner.ownershipRepo.FindByServiceName("startable")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if ownership.OwnerSessionID != defaultOwnershipSessionID {
		t.Fatalf("owner session = %q, want %q", ownership.OwnerSessionID, defaultOwnershipSessionID)
	}
	if ownership.PID <= 0 {
		t.Fatalf("ownership pid = %d, want positive", ownership.PID)
	}

	runner.monitorMu.Lock()
	_, monitorExists := runner.monitors["startable"]
	runner.monitorMu.Unlock()
	if !monitorExists {
		t.Fatal("expected monitor to be resumed after start")
	}

	if err := runner.StopService(context.Background(), "startable"); err != nil {
		t.Fatalf("cleanup StopService: %v", err)
	}
}

func TestRunner_StartService_FromRetrying(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "stuck-retry",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:           srv.URL,
							Timeout:       2,
							Retries:       2,
							CheckInterval: 1,
							Backoff:       Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	// Simulate a stuck retry (canary overlap / health check left the store on
	// Retrying with no live owner). An explicit Start must reclaim it instead of
	// rejecting with "can only start idle services".
	store.Update("stuck-retry", StatusRetrying, "previous health check left status on retrying")

	if err := runner.StartService(context.Background(), "stuck-retry"); err != nil {
		t.Fatalf("StartService from Retrying: %v", err)
	}

	got := store.Get("stuck-retry")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", got)
	}

	if err := runner.StopService(context.Background(), "stuck-retry"); err != nil {
		t.Fatalf("cleanup StopService: %v", err)
	}
}

func TestRunner_StartServiceWithActor_EstablishesOwnership(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "start-with-actor",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     srv.URL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("start-with-actor", StatusStopped, "")

	if err := runner.StartServiceWithActor(context.Background(), "start-with-actor", "ui-session-1"); err != nil {
		t.Fatalf("StartServiceWithActor: %v", err)
	}

	ownership, err := runner.ownershipRepo.FindByServiceName("start-with-actor")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if ownership.OwnerSessionID != "ui-session-1" {
		t.Fatalf("owner session = %q, want ui-session-1", ownership.OwnerSessionID)
	}
	if ownership.PID <= 0 {
		t.Fatalf("ownership pid = %d, want positive", ownership.PID)
	}

	if err := runner.StopServiceWithActor(context.Background(), "start-with-actor", "ui-session-1"); err != nil {
		t.Fatalf("cleanup StopServiceWithActor: %v", err)
	}
}

func TestRunner_StartServiceWithActor_NonOwnerDelegatesToOwner(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "owned-start",
						Command:     "echo worker",
						HealthCheck: HealthCheck{URL: "http://localhost:9511"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("owned-start", StatusStopped, "")

	ownership, err := domain.NewServiceOwnership(
		"owned-start",
		"owner-session",
		1234,
		"config-hash",
		"http://localhost:9511",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	if got := runner.resolveCascadeStepActor("owned-start", "other-session"); got != "owner-session" {
		t.Fatalf("resolveCascadeStepActor = %q, want owner-session", got)
	}
	if err := runner.ownershipGuard.EnsureOperableBySession("owned-start", "owner-session"); err != nil {
		t.Fatalf("owner session should be operable: %v", err)
	}
}

func TestRunner_StartServiceWithActor_AllowsNonOwnerAfterOwnerStops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "start-reacquire",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     srv.URL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("start-reacquire", StatusStopped, "")

	if err := runner.StartServiceWithActor(context.Background(), "start-reacquire", "owner-session"); err != nil {
		t.Fatalf("initial StartServiceWithActor: %v", err)
	}

	if err := runner.StopServiceWithActor(context.Background(), "start-reacquire", "owner-session"); err != nil {
		t.Fatalf("StopServiceWithActor: %v", err)
	}

	if err := runner.StartServiceWithActor(context.Background(), "start-reacquire", "other-session"); err != nil {
		t.Fatalf("non-owner should be able to start after stopped owner cleanup, got: %v", err)
	}

	ownership, err := runner.ownershipRepo.FindByServiceName("start-reacquire")
	if err != nil {
		t.Fatalf("FindByServiceName: %v", err)
	}
	if ownership.OwnerSessionID != "other-session" {
		t.Fatalf("owner session = %q, want other-session", ownership.OwnerSessionID)
	}

	if err := runner.StopServiceWithActor(context.Background(), "start-reacquire", "other-session"); err != nil {
		t.Fatalf("cleanup StopServiceWithActor: %v", err)
	}
}

func TestRunner_StartService_SkipsWhenAlreadyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "busy-svc",
						Command:     "echo busy",
						HealthCheck: HealthCheck{URL: srv.URL, Timeout: 2, Retries: 1, CheckInterval: 1},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("busy-svc", StatusHealthy, "")

	err = runner.StartService(context.Background(), "busy-svc")
	if !errors.Is(err, ErrStartSkippedAlreadyHealthy) {
		t.Fatalf("expected skip when already healthy, got: %v", err)
	}
}

func TestRunner_StartAll_RejectedDuringDAGBoot(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{{Name: "svc-a", Command: "sleep 1"}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	atomic.StoreInt32(&runner.dagBootInProgress, 1)
	defer atomic.StoreInt32(&runner.dagBootInProgress, 0)

	err = runner.StartAllWithActor(context.Background(), "test-session")
	if err == nil {
		t.Fatal("expected StartAll to be rejected during DAG boot")
	}
	if !strings.Contains(err.Error(), "自动引导") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunner_StartAll_AllowedAfterDAGBootCompletes(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "svc-ok", Command: "sleep 1"},
				{Name: "svc-failed", Command: "sleep 1"},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-ok", StatusHealthy, "")
	store.Update("svc-failed", StatusFailed, "boot failed")

	runner.setDAGBootInProgress(false)
	if runner.IsDAGBootInProgress() {
		t.Fatal("expected DAG boot flag cleared after boot phase")
	}
	if err := runner.rejectManualStartDuringDAGBoot(); err != nil {
		t.Fatalf("manual start should be allowed after boot: %v", err)
	}

	plan, err := runner.cascadeOrchestration().PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll: %v", err)
	}
	if len(plan.OrderedNames) != 1 || plan.OrderedNames[0] != "svc-failed" {
		t.Fatalf("plan = %#v, want [svc-failed]", plan.OrderedNames)
	}
}

func TestRunner_StartAll_AllHealthyIsNoOp(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{{Name: "svc-a", Command: "sleep 1"}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-a", StatusHealthy, "")

	if err := runner.StartAllWithActor(context.Background(), "test-session"); err != nil {
		t.Fatalf("StartAll with all healthy services: %v", err)
	}
}

func TestRunner_StartAll_ErrorReleasesRunID(t *testing.T) {
	prev := bulkProgressRetainAfterTerminal
	bulkProgressRetainAfterTerminal = 0
	t.Cleanup(func() { bulkProgressRetainAfterTerminal = prev })

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{{Name: "svc-a", Command: "sleep 1"}}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	atomic.StoreInt32(&runner.dagBootInProgress, 1)
	defer atomic.StoreInt32(&runner.dagBootInProgress, 0)

	runID := runner.SetActiveStartAllRunID()
	err = runner.StartAllWithActor(context.Background(), "test-session")
	if err == nil {
		t.Fatal("expected StartAll to be rejected during DAG boot")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runner.GetActiveStartAllRunID() == "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("start-all runID %s leaked after terminal error", runID)
}

func TestRunner_StartService_AllowsFailedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	hc := HealthCheck{URL: srv.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}}
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{Name: "failed-start", Command: "sleep 30", HealthCheck: hc},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("failed-start", StatusFailed, "previous health failure")

	if err := runner.StartService(context.Background(), "failed-start"); err != nil {
		t.Fatalf("StartService from failed: %v", err)
	}
	status := store.Get("failed-start")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", status)
	}

	if err := runner.StopService(context.Background(), "failed-start"); err != nil {
		t.Fatalf("cleanup StopService: %v", err)
	}
}

func TestRunner_StartService_OnFailureSkipStillReturnsError(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "skip-start",
						Command: "sleep 1",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:65534/unhealthy",
							Timeout: 1,
							Retries: 1,
							Backoff: Backoff{
								Initial:    0.1,
								Max:        0.1,
								Multiplier: 1.0,
							},
						},
						OnFailure: "skip",
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("skip-start", StatusStopped, "")
	err = runner.StartService(context.Background(), "skip-start")
	if err == nil {
		t.Fatal("expected start failure error even when on_failure=skip")
	}
}

func TestRunner_StartService_FailureUpdatesDependentStatus(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "worker",
						Command: "sleep 1",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:65534/unhealthy",
							Timeout: 1,
							Retries: 1,
							Backoff: Backoff{
								Initial:    0.1,
								Max:        0.1,
								Multiplier: 1.0,
							},
						},
						OnFailure: "skip",
					},
					{
						Name:      "api",
						Command:   "echo api",
						DependsOn: []string{"worker"},
						HealthCheck: HealthCheck{
							URL: "http://localhost:9401",
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("worker", StatusStopped, "")
	if err := runner.StartService(context.Background(), "worker"); err == nil {
		t.Fatal("expected start to fail")
	}

	apiStatus := store.Get("api")
	if apiStatus == nil {
		t.Fatal("api status should exist")
	}
	if len(apiStatus.DependsOn) != 1 {
		t.Fatalf("depends_on length = %d, want 1", len(apiStatus.DependsOn))
	}
	if apiStatus.DependsOn[0].Status != StatusFailed {
		t.Fatalf("dependency status = %s, want %s", apiStatus.DependsOn[0].Status, StatusFailed)
	}
}

func TestRunner_StartService_OnFailureExitCleansUpProcessAndPID(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "exit-start",
						Command: "sleep 1",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:65534/unhealthy",
							Timeout: 1,
							Retries: 1,
							Backoff: Backoff{
								Initial:    0.1,
								Max:        0.1,
								Multiplier: 1.0,
							},
						},
						OnFailure: "exit",
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("exit-start", StatusStopped, "")
	err = runner.StartService(context.Background(), "exit-start")
	if err == nil {
		t.Fatal("expected start to fail")
	}

	status := store.Get("exit-start")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.PID != 0 {
		t.Fatalf("pid = %d, want 0", status.PID)
	}

	runner.mu.Lock()
	_, exists := runner.processes["exit-start"]
	runner.mu.Unlock()
	if exists {
		t.Fatal("process should be removed after failed start cleanup")
	}
}

func TestRunner_RestartService_OnFailureSkipReturnsErrorAndCleansUp(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "restart-skip",
						Command: "sleep 1",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:65534/unhealthy",
							Timeout: 1,
							Retries: 1,
							Backoff: Backoff{
								Initial:    0.1,
								Max:        0.1,
								Multiplier: 1.0,
							},
						},
						OnFailure: "skip",
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("restart-skip", StatusHealthy, "")
	err = runner.RestartService(context.Background(), "restart-skip")
	if err == nil {
		t.Fatal("expected restart failure error when health check fails")
	}

	status := store.Get("restart-skip")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.PID != 0 {
		t.Fatalf("pid = %d, want 0", status.PID)
	}
}

func TestRunner_StopService_BlocksFailedDependentWithRunningPID(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "db",
						Command:     "echo db",
						HealthCheck: HealthCheck{URL: "http://localhost:9501"},
					},
					{
						Name:      "api",
						Command:   "echo api",
						DependsOn: []string{"db"},
						HealthCheck: HealthCheck{
							URL: "http://localhost:9502",
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("db", StatusHealthy, "")
	store.Update("api", StatusFailed, "health failed")
	store.SetPID("api", 9876)

	err = runner.StopService(context.Background(), "db")
	if err == nil {
		t.Fatal("expected stop to be blocked by failed dependent with running pid")
	}
	if !strings.Contains(err.Error(), "api") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunner_StopGroup_GroupNotFound(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "existing",
				Services: []Service{
					{
						Name:        "svc",
						Command:     "echo svc",
						HealthCheck: HealthCheck{URL: "http://localhost:9301"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.StopGroup(context.Background(), "missing-group")
	if err == nil {
		t.Fatal("expected group not found error")
	}
	if !strings.Contains(err.Error(), "group \"missing-group\" not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRunner_AppliesDefaults(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc",
					Command:     "echo hi",
					HealthCheck: HealthCheck{URL: "http://localhost:9999"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	svc := runner.findService("svc")
	if svc == nil {
		t.Fatal("service not found")
	}
	if svc.HealthCheck.CheckInterval != 10 {
		t.Errorf("CheckInterval default = %d, want 10", svc.HealthCheck.CheckInterval)
	}
	if svc.HealthCheck.UnhealthyThreshold != 2 {
		t.Errorf("UnhealthyThreshold default = %d, want 2", svc.HealthCheck.UnhealthyThreshold)
	}
	if svc.OnFailure != "exit" {
		t.Errorf("OnFailure default = %q, want 'exit'", svc.OnFailure)
	}
}

func waitForStatus(t *testing.T, store *StatusStore, name string, want Status, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := store.Get(name); s != nil && s.Status == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	got := store.Get(name)
	if got == nil {
		t.Fatalf("timed out waiting for status %q: service %q not found", want, name)
	}
	t.Fatalf("timed out waiting for status %q, got %q", want, got.Status)
}

func waitForPortClosed(t *testing.T, port int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
		if err != nil {
			return
		}
		_ = conn.Close()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("port %d still open after %s", port, timeout)
}

func waitForPortOpen(t *testing.T, port int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("port %d did not open within %s", port, timeout)
}

func waitForProcessExit(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	if pid <= 0 {
		t.Fatalf("invalid pid %d", pid)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			if err == syscall.ESRCH {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after %s", pid, timeout)
}

func startDetachedHTTPServer(t *testing.T, port int) int {
	t.Helper()
	cmd := exec.Command("sh", "-c", fmt.Sprintf("nohup python3 -m http.server %d >/dev/null 2>&1 & echo $!", port))
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("start detached server: %v", err)
	}
	pidText := strings.TrimSpace(string(output))
	pid, err := strconv.Atoi(pidText)
	if err != nil {
		t.Fatalf("parse detached pid %q: %v", pidText, err)
	}
	return pid
}

func TestMonitor_DetectsUnhealthy(t *testing.T) {
	healthy := true
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "test-svc",
					Command: "echo hi",
					HealthCheck: HealthCheck{
						URL:                srv.URL,
						Timeout:            30,
						Retries:            10,
						CheckInterval:      1,
						UnhealthyThreshold: 2,
					},
				},
			}},
		},
	}

	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("test-svc", StatusHealthy, "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := cfg.Groups[0].Services[0]
	runner.startMonitoring(ctx, svc)

	// Wait for at least one successful check (LastChecked populated)
	waitForLastChecked(t, store, "test-svc", 5*time.Second)

	// Service should still be healthy
	if got := store.Get("test-svc").Status; got != StatusHealthy {
		t.Fatalf("status = %q, want healthy", got)
	}

	// Make service unhealthy
	mu.Lock()
	healthy = false
	mu.Unlock()

	// Wait for monitor to detect unhealthy and mark failed
	waitForStatus(t, store, "test-svc", StatusFailed, 5*time.Second)
}

func waitForLastChecked(t *testing.T, store *StatusStore, name string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := store.Get(name); s != nil && s.LastChecked != "" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for LastChecked to be set on %q", name)
}

func TestMonitor_StopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "test-svc",
					Command: "echo hi",
					HealthCheck: HealthCheck{
						URL:                srv.URL,
						Timeout:            30,
						Retries:            10,
						CheckInterval:      1,
						UnhealthyThreshold: 2,
					},
				},
			}},
		},
	}

	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("test-svc", StatusHealthy, "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := cfg.Groups[0].Services[0]
	runner.startMonitoring(ctx, svc)

	// Wait for at least one check to pass
	waitForLastChecked(t, store, "test-svc", 5*time.Second)

	// Cancel the monitor via stopMonitoring (proper path)
	runner.stopMonitoring("test-svc")

	runner.monitorMu.Lock()
	_, exists := runner.monitors["test-svc"]
	runner.monitorMu.Unlock()
	if exists {
		t.Error("monitor entry should be removed after stopMonitoring")
	}
}

func TestMonitor_RecoversAfterBecomingHealthy(t *testing.T) {
	healthy := false
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ok := healthy
		mu.Unlock()
		if ok {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "test-svc",
					Command: "echo hi",
					HealthCheck: HealthCheck{
						URL:                srv.URL,
						Timeout:            30,
						Retries:            10,
						CheckInterval:      1,
						UnhealthyThreshold: 1,
					},
				},
			}},
		},
	}

	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("test-svc", StatusHealthy, "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := cfg.Groups[0].Services[0]
	runner.startMonitoring(ctx, svc)

	waitForStatus(t, store, "test-svc", StatusFailed, 5*time.Second)

	mu.Lock()
	healthy = true
	mu.Unlock()

	waitForStatus(t, store, "test-svc", StatusHealthy, 5*time.Second)

	runner.monitorMu.Lock()
	_, exists := runner.monitors["test-svc"]
	runner.monitorMu.Unlock()
	if !exists {
		t.Error("monitor should keep running after recovery")
	}
}

func TestReconcileFailedServiceHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "test-svc",
					Command: "echo hi",
					HealthCheck: HealthCheck{
						URL:                srv.URL,
						Timeout:            30,
						Retries:            10,
						CheckInterval:      1,
						UnhealthyThreshold: 2,
					},
				},
			}},
		},
	}

	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("test-svc", StatusFailed, "connection refused")

	runner.reconcileFailedServiceHealth(context.Background())

	if got := store.Get("test-svc").Status; got != StatusHealthy {
		t.Fatalf("status = %q, want healthy after reconcile", got)
	}
	if store.Get("test-svc").Error != "" {
		t.Fatalf("error = %q, want cleared", store.Get("test-svc").Error)
	}
}

func TestStartAndCheck_ProcessSurvivesContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	store.Init([]string{"sleep-svc"})

	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "sleep-svc",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:     srv.URL,
						Timeout: 5,
						Retries: 3,
						Backoff: Backoff{Initial: 0.1, Max: 0.5, Multiplier: 2.0},
					},
					OnFailure: "exit",
				},
			}},
		},
	}

	runner := &Runner{
		cfg:            cfg,
		store:          store,
		processes:      make(map[string]*exec.Cmd),
		monitors:       make(map[string]context.CancelFunc),
		listenerPIDsFn: func(string) ([]int, error) { return nil, nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node := &ServiceNode{Service: cfg.Flatten()[0]}

	// Start the service and wait for healthy
	err := runner.startAndCheck(ctx, node)
	if err != nil {
		t.Fatalf("startAndCheck: unexpected error: %v", err)
	}

	status := store.Get("sleep-svc")
	if status.Status != StatusHealthy {
		t.Fatalf("status = %q, want healthy", status.Status)
	}

	// Cancel the context — simulating DAG level completion
	cancel()

	// Give the kill signal time to arrive if it were still using CommandContext
	time.Sleep(200 * time.Millisecond)

	// The process MUST still be alive
	runner.mu.Lock()
	cmd := runner.processes["sleep-svc"]
	runner.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		t.Fatal("process should still be tracked after context cancel")
	}

	// Signal 0 checks if the process exists without sending a real signal
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("process should survive context cancel, but got: %v", err)
	}

	// Cleanup
	runner.stopProcess("sleep-svc")
}

func TestStartAndCheck_ClassifiesReadinessTimeout(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"readiness-timeout-svc"})

	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "readiness-timeout-svc",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:     "http://127.0.0.1:1/health",
						Timeout: 1,
						Retries: 1,
						Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1},
					},
					OnFailure: "exit",
				},
			}},
		},
	}

	runner := &Runner{
		cfg:            cfg,
		store:          store,
		processes:      make(map[string]*exec.Cmd),
		monitors:       make(map[string]context.CancelFunc),
		listenerPIDsFn: func(string) ([]int, error) { return nil, nil },
	}

	node := &ServiceNode{Service: cfg.Flatten()[0]}
	err := runner.startAndCheck(context.Background(), node)
	if err == nil {
		t.Fatal("expected readiness failure")
	}

	status := store.Get("readiness-timeout-svc")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", status.Status, StatusFailed)
	}
	if status.Phase != domain.ServiceLifecyclePhaseReadiness {
		t.Fatalf("phase = %q, want %q", status.Phase, domain.ServiceLifecyclePhaseReadiness)
	}
	if status.FailurePhase != domain.ServiceLifecyclePhaseReadiness {
		t.Fatalf("failure_phase = %q, want %q", status.FailurePhase, domain.ServiceLifecyclePhaseReadiness)
	}
	if status.FailureCode != domain.ServiceFailureCodeReadinessTimeout {
		t.Fatalf("failure_code = %q, want %q", status.FailureCode, domain.ServiceFailureCodeReadinessTimeout)
	}

	runner.stopProcess("readiness-timeout-svc")
}

func TestStartAndCheck_ClassifiesBadReadinessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	store := NewStatusStore()
	store.Init([]string{"bad-readiness-svc"})

	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "bad-readiness-svc",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:     srv.URL,
						Timeout: 1,
						Retries: 1,
						Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1},
					},
					OnFailure: "exit",
				},
			}},
		},
	}

	runner := &Runner{
		cfg:            cfg,
		store:          store,
		processes:      make(map[string]*exec.Cmd),
		monitors:       make(map[string]context.CancelFunc),
		listenerPIDsFn: func(string) ([]int, error) { return nil, nil },
	}

	node := &ServiceNode{Service: cfg.Flatten()[0]}
	err := runner.startAndCheck(context.Background(), node)
	if err == nil {
		t.Fatal("expected bad readiness status failure")
	}

	status := store.Get("bad-readiness-svc")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", status.Status, StatusFailed)
	}
	if status.Phase != domain.ServiceLifecyclePhaseReadiness {
		t.Fatalf("phase = %q, want %q", status.Phase, domain.ServiceLifecyclePhaseReadiness)
	}
	if status.FailureCode != domain.ServiceFailureCodeBadReadiness {
		t.Fatalf("failure_code = %q, want %q", status.FailureCode, domain.ServiceFailureCodeBadReadiness)
	}

	runner.stopProcess("bad-readiness-svc")
}

func TestStartAndCheck_ClassifiesLaunchProcessExited(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"launch-fail-svc"})

	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:       "launch-fail-svc",
					Command:    "sleep 30",
					WorkingDir: "/path/does/not/exist",
					HealthCheck: HealthCheck{
						URL:     "http://127.0.0.1:1/health",
						Timeout: 1,
						Retries: 1,
						Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1},
					},
					OnFailure: "exit",
				},
			}},
		},
	}

	runner := &Runner{
		cfg:            cfg,
		store:          store,
		processes:      make(map[string]*exec.Cmd),
		monitors:       make(map[string]context.CancelFunc),
		listenerPIDsFn: func(string) ([]int, error) { return nil, nil },
	}

	node := &ServiceNode{Service: cfg.Flatten()[0]}
	err := runner.startAndCheck(context.Background(), node)
	if err == nil {
		t.Fatal("expected launch failure")
	}
	if !strings.Contains(err.Error(), "working_dir") {
		t.Fatalf("error=%q want working_dir (not fake fork/exec bash ENOENT)", err)
	}

	status := store.Get("launch-fail-svc")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", status.Status, StatusFailed)
	}
	if status.Phase != domain.ServiceLifecyclePhaseLaunch {
		t.Fatalf("phase = %q, want %q", status.Phase, domain.ServiceLifecyclePhaseLaunch)
	}
	if status.FailureCode != domain.ServiceFailureCodeProcessExited {
		t.Fatalf("failure_code = %q, want %q", status.FailureCode, domain.ServiceFailureCodeProcessExited)
	}
}

func TestStartAndCheck_ProcessExitIncludesRecentStderr(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"stderr-exit-svc"})
	repo := infrastructure.NewInMemoryServiceLogRepository(100)

	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name: "stderr-exit-svc",
					Command: "bash -c 'echo \"TabError: inconsistent use of tabs and spaces in indentation\" >&2; " +
						"echo \"File \\\"taskproject_internal_views.py\\\", line 455\" >&2; exit 1'",
					HealthCheck: HealthCheck{
						URL:     "http://127.0.0.1:1/health",
						Timeout: 3,
						Retries: 8,
						Backoff: Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.5},
					},
					OnFailure: "exit",
				},
			}},
		},
	}

	runner := &Runner{
		cfg:            cfg,
		store:          store,
		processes:      make(map[string]*exec.Cmd),
		monitors:       make(map[string]context.CancelFunc),
		logRepository:  repo,
		listenerPIDsFn: func(string) ([]int, error) { return nil, nil },
	}

	node := &ServiceNode{Service: cfg.Flatten()[0]}
	err := runner.startAndCheck(context.Background(), node)
	if err == nil {
		t.Fatal("expected launch failure")
	}
	if !strings.Contains(err.Error(), "TabError") {
		t.Fatalf("error should include TabError stderr, got: %v", err)
	}
	if !strings.Contains(err.Error(), "recent stderr:") {
		t.Fatalf("error should include recent stderr marker, got: %v", err)
	}

	status := store.Get("stderr-exit-svc")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.FailureCode != domain.ServiceFailureCodeProcessExited {
		t.Fatalf("failure_code = %q, want %q", status.FailureCode, domain.ServiceFailureCodeProcessExited)
	}
	if !strings.Contains(status.Error, "TabError") {
		t.Fatalf("stored error should include TabError, got: %q", status.Error)
	}
}

func TestEnrichStartupFailureWithRecentLogs_PrefersStderr(t *testing.T) {
	repo := infrastructure.NewInMemoryServiceLogRepository(50)
	stdoutEntry, err := domain.NewLogEntry(time.Now(), "svc", domain.StreamStdout, "stdout only")
	if err != nil {
		t.Fatal(err)
	}
	stderrEntry, err := domain.NewLogEntry(time.Now(), "svc", domain.StreamStderr, "TabError: boom")
	if err != nil {
		t.Fatal(err)
	}
	repo.Append("svc", stdoutEntry)
	repo.Append("svc", stderrEntry)

	runner := &Runner{logRepository: repo}
	got := runner.enrichStartupFailureWithRecentLogs("svc", fmt.Errorf("launch process exited before readiness (exit=1)"))
	if !strings.Contains(got.Error(), "recent stderr:") || !strings.Contains(got.Error(), "TabError: boom") {
		t.Fatalf("unexpected enriched error: %v", got)
	}
	if strings.Contains(got.Error(), "stdout only") {
		t.Fatalf("should prefer stderr over stdout, got: %v", got)
	}
}

func TestEnrichStartupFailureWithRecentLogs_PrefersStdoutFatalOverStderrNoise(t *testing.T) {
	repo := infrastructure.NewInMemoryServiceLogRepository(50)
	stdoutEntry, err := domain.NewLogEntry(time.Now(), "svc", domain.StreamStdout,
		`{"level":"info","msg":"[taskTaskService] db: migration: database disk image is malformed (11)"}`)
	if err != nil {
		t.Fatal(err)
	}
	stderrEntry, err := domain.NewLogEntry(time.Now(), "svc", domain.StreamStderr,
		`[confload] base.yaml scheme=https baseDomain=example.com`)
	if err != nil {
		t.Fatal(err)
	}
	repo.Append("svc", stdoutEntry)
	repo.Append("svc", stderrEntry)

	runner := &Runner{logRepository: repo}
	got := runner.enrichStartupFailureWithRecentLogs("svc", fmt.Errorf("launch process exited before readiness (exit=1)"))
	if !strings.Contains(got.Error(), "recent logs:") || !strings.Contains(got.Error(), "malformed") {
		t.Fatalf("expected stdout fatal over stderr noise, got: %v", got)
	}
	if strings.Contains(got.Error(), "confload") {
		t.Fatalf("should not surface confload noise when fatal exists, got: %v", got)
	}
}

func TestJoinLogSnippet_TruncatesWithEllipsis(t *testing.T) {
	got := joinLogSnippet([]string{"abcdefghij"}, 8)
	if got != "...fghij" {
		t.Fatalf("got %q, want %q", got, "...fghij")
	}
}

func TestStartAndCheck_ContextCancellationNotClassifiedAsReadinessTimeout(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		store := NewStatusStore()
		store.Init([]string{"ctx-canceled-svc"})

		cfg := &Config{
			Version: "1",
			Groups: []Group{
				{Name: "g1", Services: []Service{
					{
						Name:    "ctx-canceled-svc",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:1/health",
							Timeout: 5,
							Retries: 5,
							Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1},
						},
						OnFailure: "exit",
					},
				}},
			},
		}

		runner := &Runner{
			cfg:       cfg,
			store:     store,
			processes: make(map[string]*exec.Cmd),
			monitors:  make(map[string]context.CancelFunc),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		node := &ServiceNode{Service: cfg.Flatten()[0]}
		err := runner.startAndCheck(ctx, node)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got: %v", err)
		}

		status := store.Get("ctx-canceled-svc")
		if status == nil {
			t.Fatal("status should exist")
		}
		if status.FailureCode == domain.ServiceFailureCodeReadinessTimeout {
			t.Fatalf("failure_code should not be readiness timeout, got %q", status.FailureCode)
		}
		if status.FailureCode != "" {
			t.Fatalf("failure_code should be empty for context cancellation, got %q", status.FailureCode)
		}
		if status.FailurePhase != "" {
			t.Fatalf("failure_phase should be empty for context cancellation, got %q", status.FailurePhase)
		}

		runner.stopProcess("ctx-canceled-svc")
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		store := NewStatusStore()
		store.Init([]string{"ctx-deadline-svc"})

		cfg := &Config{
			Version: "1",
			Groups: []Group{
				{Name: "g1", Services: []Service{
					{
						Name:    "ctx-deadline-svc",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:1/health",
							Timeout: 5,
							Retries: 5,
							Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1},
						},
						OnFailure: "exit",
					},
				}},
			},
		}

		runner := &Runner{
			cfg:       cfg,
			store:     store,
			processes: make(map[string]*exec.Cmd),
			monitors:  make(map[string]context.CancelFunc),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		time.Sleep(20 * time.Millisecond)

		node := &ServiceNode{Service: cfg.Flatten()[0]}
		err := runner.startAndCheck(ctx, node)
		if err == nil || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got: %v", err)
		}

		status := store.Get("ctx-deadline-svc")
		if status == nil {
			t.Fatal("status should exist")
		}
		if status.FailureCode == domain.ServiceFailureCodeReadinessTimeout {
			t.Fatalf("failure_code should not be readiness timeout, got %q", status.FailureCode)
		}
		if status.FailureCode != "" {
			t.Fatalf("failure_code should be empty for deadline exceeded, got %q", status.FailureCode)
		}
		if status.FailurePhase != "" {
			t.Fatalf("failure_phase should be empty for deadline exceeded, got %q", status.FailurePhase)
		}

		runner.stopProcess("ctx-deadline-svc")
	})
}

func TestRunner_StartServiceWithActor_RunsPreflightBeforeLaunch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:    "preflight-before-launch",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     srv.URL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("preflight-before-launch", StatusStopped, "")

	preflightCalled := false
	runner.preflightFn = func(ctx context.Context, svc Service) error {
		preflightCalled = true
		return nil
	}

	if err := runner.StartServiceWithActor(context.Background(), "preflight-before-launch", "ui-session"); err != nil {
		t.Fatalf("StartServiceWithActor: %v", err)
	}
	if !preflightCalled {
		t.Fatal("expected preflight before manual launch")
	}

	if err := runner.StopServiceWithActor(context.Background(), "preflight-before-launch", "ui-session"); err != nil {
		t.Fatalf("cleanup StopServiceWithActor: %v", err)
	}
}

func TestRunner_StartServiceWithActor_CleansForeignPortConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "platform",
				Services: []Service{
					{
						Name:    "git-oauth",
						Command: "sleep 30",
						HealthCheck: HealthCheck{
							URL:     srv.URL,
							Timeout: 2,
							Retries: 2,
							Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("git-oauth", StatusStopped, "")

	calls := 0
	healthPort := domain.ResolveHealthPort(srv.URL)
	runner.listenerPIDsFn = func(port string) ([]int, error) {
		if port != healthPort {
			return nil, nil
		}
		calls++
		if calls == 1 {
			return []int{8765}, nil
		}
		return nil, nil
	}

	if err := runner.StartServiceWithActor(context.Background(), "git-oauth", "ui-session"); err != nil {
		t.Fatalf("StartServiceWithActor after conflict cleanup: %v", err)
	}

	status := store.Get("git-oauth")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("expected healthy git-oauth, got %+v", status)
	}

	if err := runner.StopServiceWithActor(context.Background(), "git-oauth", "ui-session"); err != nil {
		t.Fatalf("cleanup StopServiceWithActor: %v", err)
	}
}

func TestRunner_StartServiceCascade_StartsUpstreamFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "a",
						Command:     "sleep 30",
						HealthCheck: HealthCheck{URL: srv.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
					},
					{
						Name:        "b",
						Command:     "sleep 30",
						DependsOn:   []string{"a"},
						HealthCheck: HealthCheck{URL: srv.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
					},
					{
						Name:        "c",
						Command:     "sleep 30",
						DependsOn:   []string{"b"},
						HealthCheck: HealthCheck{URL: srv.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	for _, name := range []string{"a", "b", "c"} {
		store.Update(name, StatusStopped, "")
	}

	if err := runner.StartServiceCascadeWithActor(context.Background(), "c", "cascade-session"); err != nil {
		t.Fatalf("StartServiceCascadeWithActor: %v", err)
	}
	for _, name := range []string{"a", "b", "c"} {
		status := store.Get(name)
		if status == nil || status.Status != StatusHealthy {
			t.Fatalf("service %s status = %#v, want healthy", name, status)
		}
	}

	for _, name := range []string{"c", "b", "a"} {
		if err := runner.StopServiceWithActor(context.Background(), name, "cascade-session"); err != nil {
			t.Fatalf("cleanup StopServiceWithActor(%s): %v", name, err)
		}
	}
}

func TestRunner_StopServiceCascade_StopsDownstreamFirst(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{Name: "a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "b", Command: "echo b", DependsOn: []string{"a"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "c", Command: "echo c", DependsOn: []string{"b"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	for _, name := range []string{"a", "b", "c"} {
		store.Update(name, StatusHealthy, "")
	}

	if err := runner.StopServiceCascadeWithActor(context.Background(), "a", "cascade-session"); err != nil {
		t.Fatalf("StopServiceCascadeWithActor: %v", err)
	}
	for _, name := range []string{"a", "b", "c"} {
		status := store.Get(name)
		if status == nil || status.Status != StatusStopped {
			t.Fatalf("service %s status = %#v, want stopped", name, status)
		}
	}
}

func TestRunner_StopServiceCascade_DelegatesOwnershipPerStep(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{Name: "git-oauth", Command: "echo git-oauth", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "saas-backend", Command: "echo saas-backend", DependsOn: []string{"git-oauth"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "taskFE", Command: "echo taskFE", DependsOn: []string{"saas-backend"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	for _, name := range []string{"git-oauth", "saas-backend", "taskFE"} {
		store.Update(name, StatusHealthy, "")
	}

	saveOwnership := func(serviceName, ownerSession string) {
		t.Helper()
		ownership, err := domain.NewServiceOwnership(
			serviceName,
			ownerSession,
			1234,
			"config-hash",
			"http://127.0.0.1:1",
			time.Now(),
		)
		if err != nil {
			t.Fatalf("NewServiceOwnership(%s): %v", serviceName, err)
		}
		if err := runner.ownershipRepo.Save(ownership); err != nil {
			t.Fatalf("Save ownership(%s): %v", serviceName, err)
		}
	}
	saveOwnership("taskFE", defaultOwnershipSessionID)
	saveOwnership("saas-backend", defaultOwnershipSessionID)
	saveOwnership("git-oauth", "ui-session")

	if err := runner.StopServiceCascadeWithActor(context.Background(), "git-oauth", "ui-session"); err != nil {
		t.Fatalf("StopServiceCascadeWithActor: %v", err)
	}
	for _, name := range []string{"git-oauth", "saas-backend", "taskFE"} {
		status := store.Get(name)
		if status == nil || status.Status != StatusStopped {
			t.Fatalf("service %s status = %#v, want stopped", name, status)
		}
	}
}

func TestRunner_StopServiceCascade_StopsFailedDownstreamBeforeUpstream(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "platform",
				Services: []Service{
					{Name: "git-oauth", Command: "echo git-oauth", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "saas-backend", Command: "echo saas-backend", DependsOn: []string{"git-oauth"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "taskFE", Command: "echo taskFE", DependsOn: []string{"saas-backend"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "ai-provider", Command: "echo ai-provider", DependsOn: []string{"saas-backend"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("git-oauth", StatusHealthy, "")
	store.Update("saas-backend", StatusFailed, "health check failed")
	store.SetPID("saas-backend", 4321)
	store.Update("taskFE", StatusStopped, "")
	store.Update("ai-provider", StatusHealthy, "")

	saveOwnership := func(serviceName, ownerSession string) {
		t.Helper()
		ownership, err := domain.NewServiceOwnership(
			serviceName,
			ownerSession,
			1234,
			"config-hash",
			"http://127.0.0.1:1",
			time.Now(),
		)
		if err != nil {
			t.Fatalf("NewServiceOwnership(%s): %v", serviceName, err)
		}
		if err := runner.ownershipRepo.Save(ownership); err != nil {
			t.Fatalf("Save ownership(%s): %v", serviceName, err)
		}
	}
	saveOwnership("ai-provider", defaultOwnershipSessionID)
	saveOwnership("saas-backend", defaultOwnershipSessionID)
	saveOwnership("git-oauth", "ui-session")

	if err := runner.StopServiceCascadeWithActor(context.Background(), "git-oauth", "ui-session"); err != nil {
		t.Fatalf("StopServiceCascadeWithActor: %v", err)
	}
	for _, name := range []string{"git-oauth", "saas-backend", "ai-provider"} {
		status := store.Get(name)
		if status == nil || status.Status != StatusStopped {
			t.Fatalf("service %s status = %#v, want stopped", name, status)
		}
	}
}

func TestRunner_StartServiceCascade_StartsPendingChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	hc := HealthCheck{URL: srv.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}}
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "platform",
				Services: []Service{
					// detach: echo exits immediately; readiness comes from the shared test health server
					{Name: "git-oauth", Command: "echo git-oauth", LaunchMode: "detach", HealthCheck: hc},
					{Name: "saas-backend", Command: "echo saas-backend", LaunchMode: "detach", DependsOn: []string{"git-oauth"}, HealthCheck: hc},
					{Name: "taskFE", Command: "echo taskFE", LaunchMode: "detach", DependsOn: []string{"saas-backend"}, HealthCheck: hc},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("git-oauth", StatusHealthy, "")
	store.Update("saas-backend", StatusPending, "")
	store.Update("taskFE", StatusPending, "")

	if err := runner.StartServiceCascadeWithActor(context.Background(), "taskFE", "ui-session"); err != nil {
		t.Fatalf("StartServiceCascadeWithActor: %v", err)
	}
	for _, name := range []string{"saas-backend", "taskFE"} {
		status := store.Get(name)
		if status == nil || status.Status != StatusHealthy {
			t.Fatalf("service %s status = %#v, want healthy", name, status)
		}
	}
}

func TestRunner_StopService_BlocksWhenCascadeFalse(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{Name: "a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "b", Command: "echo b", DependsOn: []string{"a"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("a", StatusHealthy, "")
	store.Update("b", StatusHealthy, "")

	err = runner.StopServiceWithActor(context.Background(), "a", "owner-session")
	if err == nil {
		t.Fatal("expected single stop to be blocked by active downstream")
	}
	if !strings.Contains(err.Error(), "active downstream dependencies") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunner_StartGroup_StartsStoppedServicesInOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	hc := HealthCheck{URL: srv.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}}
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "infra",
				Services: []Service{
					{Name: "git-oauth", Command: "sleep 30", HealthCheck: hc},
				},
			},
			{
				Name: "platform",
				Services: []Service{
					{Name: "saas-backend", Command: "sleep 30", DependsOn: []string{"git-oauth"}, HealthCheck: hc},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("git-oauth", StatusStopped, "")
	store.Update("saas-backend", StatusStopped, "")

	if err := runner.StartGroupWithActor(context.Background(), "platform", "group-session"); err != nil {
		t.Fatalf("StartGroupWithActor: %v", err)
	}
	for _, name := range []string{"git-oauth", "saas-backend"} {
		status := store.Get(name)
		if status == nil || status.Status != StatusHealthy {
			t.Fatalf("service %s status = %#v, want healthy", name, status)
		}
	}

	for _, name := range []string{"saas-backend", "git-oauth"} {
		if err := runner.StopServiceWithActor(context.Background(), name, "group-session"); err != nil {
			t.Fatalf("cleanup StopServiceWithActor(%s): %v", name, err)
		}
	}
}

func TestExecuteLevel_ResilientMode_PreservesHealthyPeer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "platform",
			Services: []Service{
				{
					Name:    "peer-healthy",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:     srv.URL,
						Timeout: 5,
						Retries: 3,
						Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
					},
				},
				{
					Name:        "peer-broken",
					Command:     "sleep 30",
					OnFailure:   "exit",
					HealthCheck: HealthCheck{URL: "http://127.0.0.1:65534/unhealthy", Timeout: 1, Retries: 1, Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1}},
				},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, nil
	}

	level := runner.levels[0]
	if err := runner.executeLevel(context.Background(), level, true); err != nil {
		t.Fatalf("executeLevel resilient: %v", err)
	}

	waitForServiceStatus(t, store, "peer-healthy", StatusHealthy, 10*time.Second)
	waitForServiceStatus(t, store, "peer-broken", StatusFailed, 10*time.Second)

	runner.mu.Lock()
	_, okHealthy := runner.processes["peer-healthy"]
	runner.mu.Unlock()
	if !okHealthy {
		t.Fatal("peer-healthy process should still be running after peer-broken failed")
	}

	runner.stopProcess("peer-healthy")
	runner.stopProcess("peer-broken")
}

func TestRun_UIMode_ContinuesWhenPeerFailsWithOnFailureExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "platform",
			Services: []Service{
				{
					Name:    "peer-healthy",
					Command: "sleep 120",
					HealthCheck: HealthCheck{
						URL:     srv.URL,
						Timeout: 5,
						Retries: 3,
						Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
					},
				},
				{
					Name:        "peer-broken",
					Command:     "sleep 120",
					OnFailure:   "exit",
					HealthCheck: HealthCheck{URL: "http://127.0.0.1:65534/unhealthy", Timeout: 1, Retries: 1, Backoff: Backoff{Initial: 0.1, Max: 0.1, Multiplier: 1}},
				},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, false)
	}()
	t.Cleanup(func() {
		runner.stopProcess("peer-healthy")
		runner.stopProcess("peer-broken")
	})

	// UI mode no longer auto-starts services; trigger start-all explicitly.
	if err := runner.StartAllWithActor(context.Background(), "test-session"); err != nil {
		t.Fatalf("StartAll: %v", err)
	}

	waitForServiceStatus(t, store, "peer-healthy", StatusHealthy, 10*time.Second)
	waitForServiceStatus(t, store, "peer-broken", StatusFailed, 10*time.Second)

	runner.mu.Lock()
	_, okHealthy := runner.processes["peer-healthy"]
	runner.mu.Unlock()
	if !okHealthy {
		t.Fatal("peer-healthy process should still be running after peer-broken failed")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error in UI mode: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}

func TestBuildParallelStartLevels(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "platform",
				Services: []Service{
					{Name: "git-oauth", Command: "echo git-oauth", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "saas-backend", Command: "echo saas-backend", DependsOn: []string{"git-oauth"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "taskFE", Command: "echo taskFE", DependsOn: []string{"saas-backend"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "ai-provider", Command: "echo ai-provider", DependsOn: []string{"saas-backend"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "standalone", Command: "echo standalone", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	ordered := []string{"git-oauth", "saas-backend", "taskFE", "ai-provider", "standalone"}
	levels, err := runner.buildParallelStartLevels(ordered)
	if err != nil {
		t.Fatalf("buildParallelStartLevels: %v", err)
	}
	want := [][]string{
		{"git-oauth", "standalone"},
		{"saas-backend"},
		{"ai-provider", "taskFE"},
	}
	if len(levels) != len(want) {
		t.Fatalf("levels = %#v, want %#v", levels, want)
	}
	for i := range want {
		if len(levels[i]) != len(want[i]) {
			t.Fatalf("level %d = %#v, want %#v", i, levels[i], want[i])
		}
		for j := range want[i] {
			if levels[i][j] != want[i][j] {
				t.Fatalf("level %d = %#v, want %#v", i, levels[i], want[i])
			}
		}
	}

	orderedPartial := []string{"taskFE", "ai-provider"}
	levelsPartial, err := runner.buildParallelStartLevels(orderedPartial)
	if err != nil {
		t.Fatalf("buildParallelStartLevels partial: %v", err)
	}
	if len(levelsPartial) != 1 || len(levelsPartial[0]) != 2 {
		t.Fatalf("partial levels = %#v, want single parallel level of both services", levelsPartial)
	}
}

func TestExplainPendingCause_BlockedByDependency(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "upstream", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				{Name: "downstream", Command: "sleep 1", DependsOn: []string{"upstream"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:2"}},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("upstream", StatusStarting, "")
	store.SetDependsOn("downstream", []DepStatus{{Name: "upstream", Status: StatusStarting}})

	cause := runner.explainPendingCause("downstream")
	if !strings.Contains(cause, "upstream") {
		t.Fatalf("cause = %q, want upstream dependency mention", cause)
	}
}

func TestStreamOutput_SkipsBlankLines(t *testing.T) {
	repo := infrastructure.NewInMemoryServiceLogRepository(100)
	input := strings.NewReader("line-one\n\n  \nline-two\n")
	streamOutput(input, "svc", domain.StreamStdout, repo)

	entries := repo.Tail("svc", 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	if entries[0].Message != "line-one" || entries[1].Message != "line-two" {
		t.Fatalf("unexpected messages: %#v", entries)
	}
}

func TestStartAndCheck_UsesLivenessURLForStartup(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/live/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/health/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	store := NewStatusStore()
	store.Init([]string{"liveness-svc"})
	cfg := &Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:    "liveness-svc",
				Command: "sleep 30",
				HealthCheck: HealthCheck{
					LivenessURL: srv.URL + "/api/live/",
					URL:         srv.URL + "/api/health/",
					Timeout:     5,
					Retries:     3,
					Backoff:     Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
				},
				OnFailure: "skip",
			}},
		}},
	}
	runner := &Runner{
		cfg:            cfg,
		store:          store,
		processes:      make(map[string]*exec.Cmd),
		monitors:       make(map[string]context.CancelFunc),
		listenerPIDsFn: func(string) ([]int, error) { return nil, nil },
	}

	node := &ServiceNode{Service: cfg.Flatten()[0]}
	if err := runner.startAndCheck(context.Background(), node); err != nil {
		t.Fatalf("startAndCheck with liveness probe: %v", err)
	}
	status := store.Get("liveness-svc")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", status)
	}
	runner.stopProcess("liveness-svc")
}

func TestRunMonitor_SplitProbeReadinessDegradedPreservesLive(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:    "split-svc",
				Command: "sleep 30",
				HealthCheck: HealthCheck{
					LivenessURL:        srv.URL + "/live",
					URL:                srv.URL + "/ready",
					CheckInterval:      1,
					UnhealthyThreshold: 2,
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("split-svc", StatusHealthy, "")
	store.SetReadiness("split-svc", ReadinessReady, "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner.startMonitoring(ctx, cfg.Groups[0].Services[0])

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st := store.Get("split-svc")
		if st != nil && st.Status == StatusHealthy && st.Readiness == ReadinessDegraded {
			cancel()
			runner.stopProcess("split-svc")
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected healthy + readiness degraded, got %#v", store.Get("split-svc"))
}

// --- BuildGroup tests ---

func TestBuildGroup_AllSucceed(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "buildgroup-test", Services: []Service{
				{
					Name:         "bg-svc-a",
					BuildCommand: "echo built-a",
					Command:      "echo running-a",
					HealthCheck:  HealthCheck{URL: "http://localhost:9901"},
				},
				{
					Name:         "bg-svc-b",
					BuildCommand: "echo built-b",
					Command:      "echo running-b",
					HealthCheck:  HealthCheck{URL: "http://localhost:9902"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("bg-svc-a", StatusHealthy, "")
	store.Update("bg-svc-b", StatusFailed, "")

	result, err := runner.BuildGroup(context.Background(), "buildgroup-test")
	if err != nil {
		t.Fatalf("BuildGroup: unexpected error: %v", err)
	}
	if result.Status != domain.BuildGroupStatusOK {
		t.Errorf("status = %s, want %s", result.Status, domain.BuildGroupStatusOK)
	}
	if result.Built != 2 {
		t.Errorf("built = %d, want 2", result.Built)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if len(result.Failed) != 0 {
		t.Errorf("failed = %v, want empty", result.Failed)
	}
}

// 分组编译成功后仅移除本组已登记且编译成功的服务名，其它登记保留（OPT-20260811-036）。
func TestBuildGroup_TrimsOnlyBuiltGroupRegistrations(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "bg-trim", Services: []Service{
				{
					Name:         "bg-trim-a",
					BuildCommand: "echo built-a",
					Command:      "echo running-a",
					HealthCheck:  HealthCheck{URL: "http://localhost:9911"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("bg-trim-a", StatusHealthy, "")

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "bg-trim-a", State: RegistrationStatePending, RegisteredAt: 100},       // 组内待重启
		{Name: "other-group-svc", State: RegistrationStatePending, RegisteredAt: 110}, // 他组
		{Name: "bg-trim-a-failed", State: RegistrationStateFailed, RegisteredAt: 120}, // 组内失败项
	}); err != nil {
		t.Fatal(err)
	}

	result, err := runner.BuildGroup(context.Background(), "bg-trim")
	if err != nil {
		t.Fatalf("BuildGroup: %v", err)
	}
	if result.Built != 1 {
		t.Fatalf("built = %d, want 1", result.Built)
	}

	entries, _ := readRegistrationEntries(path)
	byName := map[string]RegistrationState{}
	for _, e := range entries {
		byName[e.Name] = e.State
	}
	if _, ok := byName["bg-trim-a"]; ok {
		t.Fatalf("built group service bg-trim-a must be removed from registration, got %+v", entries)
	}
	if byName["other-group-svc"] != RegistrationStatePending {
		t.Fatalf("other-group service must be preserved, got %+v", entries)
	}
	if byName["bg-trim-a-failed"] != RegistrationStateFailed {
		t.Fatalf("failed group service must be preserved as failed, got %+v", entries)
	}
}

func TestBuildGroup_GroupNotFound(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "existing", Services: []Service{
				{
					Name:        "svc",
					Command:     "echo svc",
					HealthCheck: HealthCheck{URL: "http://localhost:9903"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	_, err = runner.BuildGroup(context.Background(), "missing-group")
	if err == nil {
		t.Fatal("expected group not found error")
	}
	if !strings.Contains(err.Error(), "group \"missing-group\" not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildGroup_SkipsWhenBuilding(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "bg-skip", Services: []Service{
				{
					Name:         "bg-skip-building",
					BuildCommand: "echo should-not-run",
					Command:      "sleep 30",
					HealthCheck:  HealthCheck{URL: "http://localhost:9904"},
				},
				{
					Name:         "bg-skip-healthy",
					BuildCommand: "echo should-run",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9905"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("bg-skip-building", StatusBuilding, "")
	store.Update("bg-skip-healthy", StatusHealthy, "")

	result, err := runner.BuildGroup(context.Background(), "bg-skip")
	if err != nil {
		t.Fatalf("BuildGroup: unexpected error: %v", err)
	}
	if result.Built != 1 {
		t.Errorf("built = %d, want 1", result.Built)
	}
	if len(result.Skipped) != 1 {
		t.Errorf("skipped = %v, want 1", result.Skipped)
	}
	if result.Skipped[0] != "bg-skip-building" {
		t.Errorf("skipped[0] = %s, want bg-skip-building", result.Skipped[0])
	}
}

func TestBuildGroup_NoBuildableServices(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "bg-nobuild", Services: []Service{
				{
					Name:        "bg-no-cmd",
					Command:     "echo hello",
					HealthCheck: HealthCheck{URL: "http://localhost:9906"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("bg-no-cmd", StatusHealthy, "")

	result, err := runner.BuildGroup(context.Background(), "bg-nobuild")
	if err != nil {
		t.Fatalf("BuildGroup: unexpected error: %v", err)
	}
	if result.Status != domain.BuildGroupStatusNone {
		t.Errorf("status = %s, want %s", result.Status, domain.BuildGroupStatusNone)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
	if len(result.NoBuild) != 1 {
		t.Errorf("no_build = %v, want 1", result.NoBuild)
	}
}

func TestBuildGroup_PartialFailure(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "bg-partial", Services: []Service{
				{
					Name:         "bg-fail",
					BuildCommand: "exit 1",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9907"},
				},
				{
					Name:         "bg-ok",
					BuildCommand: "echo built-ok",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9908"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("bg-fail", StatusHealthy, "")
	store.Update("bg-ok", StatusHealthy, "")

	result, err := runner.BuildGroup(context.Background(), "bg-partial")
	if err != nil {
		t.Fatalf("BuildGroup: unexpected error: %v", err)
	}
	if result.Status != domain.BuildGroupStatusPartial {
		t.Errorf("status = %s, want %s", result.Status, domain.BuildGroupStatusPartial)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if result.Built != 1 {
		t.Errorf("built = %d, want 1", result.Built)
	}
	if len(result.Failed) != 1 {
		t.Errorf("failed = %v, want 1", result.Failed)
	}
}

// forceStopService stops a service even when it has active downstream dependents.
// This is the core fix: StopAll should not be blocked by dependency relationships.
func TestForceStopService_StopsDespiteActiveDownstreamDependency(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "redis",
						Command:     "echo redis",
						HealthCheck: HealthCheck{URL: "http://localhost:9601"},
					},
					{
						Name:      "api",
						Command:   "echo api",
						DependsOn: []string{"redis"},
						HealthCheck: HealthCheck{
							URL: "http://localhost:9602",
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("redis", StatusHealthy, "")
	store.Update("api", StatusHealthy, "")

	if err := runner.forceStopService(context.Background(), "redis"); err != nil {
		t.Fatalf("forceStopService should not be blocked by active dependents, got: %v", err)
	}

	status := store.Get("redis")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("redis status = %#v, want stopped", status)
	}
}

func TestForceStopService_SetsStoppedStatusAndClearsPID(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "svc",
						Command:     "echo svc",
						HealthCheck: HealthCheck{URL: "http://localhost:9701"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc", StatusHealthy, "")
	store.SetPID("svc", 12345)

	if err := runner.forceStopService(context.Background(), "svc"); err != nil {
		t.Fatalf("forceStopService: %v", err)
	}

	s := store.Get("svc")
	if s == nil {
		t.Fatal("status should exist")
	}
	if s.Status != StatusStopped {
		t.Fatalf("status = %s, want stopped", s.Status)
	}
	if s.PID != 0 {
		t.Fatalf("pid = %d, want 0", s.PID)
	}
}

func TestForceStopService_SkipsAlreadyStopped(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "svc",
						Command:     "echo svc",
						HealthCheck: HealthCheck{URL: "http://localhost:9801"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc", StatusStopped, "")

	if err := runner.forceStopService(context.Background(), "svc"); err != nil {
		t.Fatalf("forceStopService on already-stopped should not error: %v", err)
	}

	s := store.Get("svc")
	if s == nil || s.Status != StatusStopped {
		t.Fatalf("status = %#v, want stopped (unchanged)", s)
	}
}

func TestForceStopService_ServiceNotFound(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups:  []Group{},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.forceStopService(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent service")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopAllWithActor_StopsInReverseDependencyOrder(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "redis",
						Command:     "echo redis",
						HealthCheck: HealthCheck{URL: "http://localhost:9901"},
					},
					{
						Name:      "api",
						Command:   "echo api",
						DependsOn: []string{"redis"},
						HealthCheck: HealthCheck{
							URL: "http://localhost:9902",
						},
					},
					{
						Name:      "web",
						Command:   "echo web",
						DependsOn: []string{"api"},
						HealthCheck: HealthCheck{
							URL: "http://localhost:9903",
						},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("redis", StatusHealthy, "")
	store.Update("api", StatusHealthy, "")
	store.Update("web", StatusHealthy, "")

	err = runner.StopAllWithActor(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("StopAllWithActor should not be blocked by dependencies, got: %v", err)
	}

	for _, name := range []string{"redis", "api", "web"} {
		s := store.Get(name)
		if s == nil || s.Status != StatusStopped {
			t.Errorf("%s status = %#v, want stopped", name, s)
		}
	}
}

func TestStopAllWithActor_IncludesErrorDetails(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{
						Name:        "svc1",
						Command:     "echo svc1",
						HealthCheck: HealthCheck{URL: "http://localhost:9991"},
					},
					{
						Name:        "svc2",
						Command:     "echo svc2",
						HealthCheck: HealthCheck{URL: "http://localhost:9992"},
					},
				},
			},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("svc1", StatusHealthy, "")
	store.Update("svc2", StatusHealthy, "")

	stopCmd := "exit 1"
	svc2 := runner.findService("svc2")
	if svc2 == nil {
		t.Fatal("svc2 not found")
	}
	svc2.StopCommand = stopCmd

	err = runner.StopAllWithActor(context.Background(), "test-session")
	if err == nil {
		t.Fatal("expected error from StopAllWithActor when one service fails")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "svc2") {
		t.Errorf("error should mention failing service name 'svc2': %s", errMsg)
	}
	if strings.HasSuffix(strings.TrimSpace(errMsg), "svc2") && !strings.Contains(errMsg, "exit") {
		t.Errorf("error should include failure reason, not just service name: %s", errMsg)
	}
}

// OPT-20260818-005 回归：清库停止阶段每停一个服务上报进度。用 exec 探针
// 服务：stop 命令移除探针文件使探针不可达，从而完整走 stop → 探针确认 →
// 已停止 的路径，断言进度回调收到逐服务消息。
func TestStopAllApplicationsExcept_EmitsPerServiceProgress(t *testing.T) {
	dir := t.TempDir()
	healthFile := filepath.Join(dir, "svc-health")
	if err := os.WriteFile(healthFile, []byte("up"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{
				{Name: "svc-a", Command: "true", StopCommand: "rm -f " + healthFile, HealthCheck: HealthCheck{Exec: "test -f " + healthFile}},
			},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-a", StatusHealthy, "")

	var messages []string
	stopped, still := runner.StopAllApplicationsExcept(context.Background(), nil, func(msg string) {
		messages = append(messages, msg)
	})
	if len(stopped) != 1 || stopped[0] != "svc-a" {
		t.Fatalf("stopped=%v want [svc-a]", stopped)
	}
	if len(still) != 0 {
		t.Fatalf("still=%v want empty", still)
	}
	sawProgress := false
	for _, m := range messages {
		if m == "已停止服务: svc-a" {
			sawProgress = true
		}
	}
	if !sawProgress {
		t.Fatalf("messages=%v want per-service progress for svc-a", messages)
	}
}

// OPT-20260817-034 回归：ensureServiceNotReachable 必须响应调用方 ctx，
// 阻塞的健康端点不能突破 precise-restart / stop 阶段的截止时间。
func TestEnsureServiceNotReachable_RespondsToCallerCtx(t *testing.T) {
	blockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 阻塞直到客户端取消/连接关闭，保证 Close 能完成。
		<-r.Context().Done()
	}))
	defer blockSrv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{
				{Name: "svc", Command: "true", HealthCheck: HealthCheck{URL: blockSrv.URL}},
			},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runner.ensureServiceNotReachable(ctx, runner.findService("svc"))
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		// 阻塞端点随 ctx 取消视为「不可达」，返回 nil；核心是快速返回。
		if err != nil {
			t.Fatalf("ensureServiceNotReachable err=%v want nil after ctx cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ensureServiceNotReachable did not respond to caller ctx cancel")
	}
}
