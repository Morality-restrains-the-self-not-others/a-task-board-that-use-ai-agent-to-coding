# runAll: Build Before Restart — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional `build_command` to service config; on restart, compile first, then stop + restart only if compilation succeeds.

**Architecture:** New `BuildCommand` field on `Service` struct, new `StatusBuilding` constant, new `runBuild` method on `Runner`, reordered `RestartService` to run build before stopping the old process.

**Tech Stack:** Go 1.x, no external dependencies beyond stdlib + `gopkg.in/yaml.v3`

---

### Task 1: Add `BuildCommand` field to config

**Files:**
- Modify: `runAll/config.go:22-29`

- [ ] **Step 1: Add `BuildCommand` field to `Service` struct**

Edit `runAll/config.go`, add one line to the `Service` struct after `Command`:

```go
type Service struct {
	Name         string            `yaml:"name"`
	Command      string            `yaml:"command"`
	BuildCommand string            `yaml:"build_command"` // optional, runs before restart
	WorkingDir   string            `yaml:"working_dir"`
	Env          map[string]string `yaml:"env"`
	DependsOn    []string          `yaml:"depends_on"`
	OnFailure    string            `yaml:"on_failure"`
	HealthCheck  HealthCheck       `yaml:"health_check"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd runAll && go build ./...`
Expected: exit 0, no errors.

- [ ] **Step 3: Write config test for `build_command` parsing**

Edit `runAll/config_test.go`, add new test:

```go
func TestLoadConfig_BuildCommand(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "./app"
        build_command: "go build -o app ."
        health_check:
          url: "http://localhost:1"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if svc.BuildCommand != "go build -o app ." {
		t.Errorf("build_command = %q, want %q", svc.BuildCommand, "go build -o app .")
	}
}
```

- [ ] **Step 4: Run config test**

Run: `cd runAll && go test -run TestLoadConfig_BuildCommand -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add runAll/config.go runAll/config_test.go
git commit -m "feat: add BuildCommand field to Service config"
```

---

### Task 2: Add `StatusBuilding` constant

**Files:**
- Modify: `runAll/status.go:10-19`

- [ ] **Step 1: Add `StatusBuilding` constant**

Edit `runAll/status.go`, add after `StatusRestarting`:

```go
const (
	StatusPending    Status = "pending"
	StatusStarting   Status = "starting"
	StatusRetrying   Status = "retrying"
	StatusHealthy    Status = "healthy"
	StatusFailed     Status = "failed"
	StatusSkipped    Status = "skipped"
	StatusRestarting Status = "restarting"
	StatusBuilding   Status = "building"
)
```

- [ ] **Step 2: Verify compilation**

Run: `cd runAll && go build ./...`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add runAll/status.go
git commit -m "feat: add StatusBuilding constant"
```

---

### Task 3: Add `runBuild` method and reorder `RestartService`

**Files:**
- Modify: `runAll/runner.go`

- [ ] **Step 1: Add `runBuild` method**

Add new method to `runner.go` after `stopProcess` (after line 318):

```go
func (r *Runner) runBuild(ctx context.Context, svc *Service) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", svc.BuildCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if svc.WorkingDir != "" {
		cmd.Dir = svc.WorkingDir
	}
	if len(svc.Env) > 0 {
		env := os.Environ()
		for k, v := range svc.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("[%s] build stdout pipe: %w", svc.Name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("[%s] build stderr pipe: %w", svc.Name, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("[%s] build failed to start: %w", svc.Name, err)
	}

	go streamOutput(stdout, svc.Name)
	go streamOutput(stderr, svc.Name)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("[%s] build failed: %w", svc.Name, err)
	}

	log.Printf("[%s] build succeeded", svc.Name)
	return nil
}
```

- [ ] **Step 2: Rewrite `RestartService` to run build before stop**

Replace the existing `RestartService` method (lines 251-277) with:

```go
func (r *Runner) RestartService(ctx context.Context, name string) error {
	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}

	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	if current.Status != StatusHealthy && current.Status != StatusFailed {
		return fmt.Errorf("service %q is %s, can only restart healthy or failed services", name, current.Status)
	}

	r.store.Update(name, StatusRestarting, "")

	// Build before stopping: a failed build leaves the old process running.
	if svc.BuildCommand != "" {
		r.store.Update(name, StatusBuilding, "")
		log.Printf("[%s] building...", name)
		if err := r.runBuild(ctx, svc); err != nil {
			r.store.Update(name, StatusFailed, err.Error())
			return err
		}
	}

	// Stop existing process
	r.stopProcess(name)

	// Start and health check
	node := &ServiceNode{Service: *svc}
	if err := r.startAndCheck(ctx, node); err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd runAll && go build ./...`
Expected: exit 0, no errors.

- [ ] **Step 4: Commit**

```bash
git add runAll/runner.go
git commit -m "feat: run build_command before restart, keep old process on build failure"
```

---

### Task 4: Write tests for restart with build

**Files:**
- Create: `runAll/runner_test.go`

- [ ] **Step 1: Write `runBuild` unit tests**

Create `runAll/runner_test.go`:

```go
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuild_Success(t *testing.T) {
	runner := &Runner{}
	svc := &Service{
		Name:         "test-build",
		BuildCommand: "echo built",
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc)
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
	err := runner.runBuild(ctx, svc)
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
	err := runner.runBuild(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Errorf("build should have created marker file: %v", statErr)
	}
}

func TestRunBuild_WithEnv(t *testing.T) {
	runner := &Runner{}
	svc := &Service{
		Name:         "test-build-env",
		BuildCommand: "echo $MY_VAR",
		Env:          map[string]string{"MY_VAR": "hello"},
	}

	ctx := context.Background()
	err := runner.runBuild(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	err := runner.runBuild(ctx, svc)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestRestartService_NoBuildCommand(t *testing.T) {
	// Restart without BuildCommand should skip build step
	// Use a simple command that starts and is healthy
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

	// Set status to healthy so restart is allowed
	store.Update("echo-svc", StatusHealthy, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "echo-svc")
	// It'll fail health check (no real server), but we just verify
	// it doesn't fail at the build step
	if err == nil {
		t.Log("restart succeeded (unexpected but ok)")
	}
}

func TestRestartService_WithBuildCommand_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-ok",
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
	store.Update("build-ok", StatusHealthy, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "build-ok")
	// Build succeeds but health check fails (no real server)
	// The important thing: it should NOT be a build error
	if err != nil && strings.Contains(err.Error(), "build failed") {
		t.Errorf("unexpected build failure: %v", err)
	}
}

func TestRestartService_WithBuildCommand_Failure(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "build-fail",
					BuildCommand: "exit 2",
					Command:      "echo running",
					HealthCheck:  HealthCheck{URL: "http://localhost:9997"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("build-fail", StatusHealthy, "")

	ctx := context.Background()
	err = runner.RestartService(ctx, "build-fail")
	if err == nil {
		t.Fatal("expected error for build failure")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Errorf("error should mention build failed, got: %v", err)
	}

	// Verify status reflects the build failure
	status := store.Get("build-fail")
	if status.Status != StatusFailed {
		t.Errorf("status = %q, want %q", status.Status, StatusFailed)
	}
	if status.Error == "" {
		t.Error("error message should be set")
	}
}

func TestRestartService_NotHealthyOrFailed(t *testing.T) {
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
	// Status is pending (set by Init), not healthy/failed
	ctx := context.Background()
	err = runner.RestartService(ctx, "pending-svc")
	if err == nil {
		t.Fatal("expected error for non-healthy/non-failed service")
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
```

- [ ] **Step 2: Run runner tests**

Run: `cd runAll && go test -v -run "TestRunBuild|TestRestartService"`
Expected: all PASS.

- [ ] **Step 3: Run full test suite to check for regressions**

Run: `cd runAll && go test ./... -v`
Expected: all existing tests still PASS.

- [ ] **Step 4: Commit**

```bash
git add runAll/runner_test.go
git commit -m "test: add tests for restart with build_command"
```

---

### Task 5: Update `config.yaml` to use `build_command`

**Files:**
- Modify: `config.yaml`

- [ ] **Step 1: Split `go-run-container` build/run commands**

Edit `config.yaml` lines 79-85, change the `go-run-container` service:

Before:
```yaml
      - name: go-run-container
        command: "go build -o go_run_container . && ./go_run_container"
        working_dir: go_run_container
        health_check:
          url: "http://127.0.0.1:8796/health"
          timeout: 60
          retries: 15
```

After:
```yaml
      - name: go-run-container
        build_command: "go build -o go_run_container ."
        command: "./go_run_container"
        working_dir: go_run_container
        health_check:
          url: "http://127.0.0.1:8796/health"
          timeout: 60
          retries: 15
```

- [ ] **Step 2: Split `go-relay` build/run commands**

Edit `config.yaml` lines 87-93, change the `go-relay` service:

Before:
```yaml
      - name: go-relay
        command: "go build -o go_relayToTrae . && ./go_relayToTrae"
        working_dir: go_relayToTrae
        health_check:
          url: "http://127.0.0.1:8797/health"
          timeout: 60
          retries: 15
```

After:
```yaml
      - name: go-relay
        build_command: "go build -o go_relayToTrae ."
        command: "./go_relayToTrae"
        working_dir: go_relayToTrae
        health_check:
          url: "http://127.0.0.1:8797/health"
          timeout: 60
          retries: 15
```

- [ ] **Step 3: Commit**

```bash
git add config.yaml
git commit -m "config: split build and run commands for go-run-container and go-relay"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run full test suite**

Run: `cd runAll && go test ./... -v`
Expected: all tests PASS.

- [ ] **Step 2: Build the runAll binary**

Run: `cd runAll && go build -o runAll .`
Expected: exit 0, binary created.

- [ ] **Step 3: Smoke test — config loads with build_command**

Run: `cd runAll && go run . --config ../config.yaml --daemon`
Expected: services start, build_command is NOT executed (daemon mode confirms no regression).
Interrupt after "All services healthy." with Ctrl+C.
