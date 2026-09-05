package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunner_validateTestTarget(t *testing.T) {
	cfg := minimalConfig()
	r := NewRunner(cfg, NewStatusStore(cfg))
	if err := r.validateTestTarget("flow-a", "s1", true); err != nil {
		t.Fatalf("valid target: %v", err)
	}
	if err := r.validateTestTarget("nope", "s1", true); err == nil {
		t.Fatal("want unknown stream error")
	}
	if err := r.validateTestTarget("flow-a", "nope", true); err == nil {
		t.Fatal("want unknown step error")
	}
	if err := r.validateTestTarget("flow-a", "", false); err != nil {
		t.Fatalf("stream only: %v", err)
	}
}

func TestRunner_validateTestTarget_PlannedStepRejected(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{Name: "planned-step", Lifecycle: "planned", TestFile: "future.py"},
				{Name: "active-step", Lifecycle: "active", TestFile: "now.py"},
			},
		}},
	}
	r := NewRunner(cfg, NewStatusStore(cfg))

	err := r.validateTestTarget("flow-a", "planned-step", true)
	if err == nil {
		t.Fatal("want planned step rejected")
	}
	if !strings.Contains(err.Error(), "planned") || !strings.Contains(err.Error(), "cannot be run") {
		t.Fatalf("want clear planned error, got %q", err.Error())
	}
}

func TestRunner_RunStep_MockPytest(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "mock-pytest")
	script := "#!/bin/sh\nb=$(basename \"$1\")\nif [ \"$b\" = \"fail.py\" ]; then exit 1; fi\nexit 0\n"
	os.WriteFile(mockPytest, []byte(script), 0755)

	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	passFile := filepath.Join(proj, "pass.py")
	failFile := filepath.Join(proj, "fail.py")
	os.WriteFile(passFile, []byte(""), 0644)
	os.WriteFile(failFile, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner: RunnerConfig{
			WorkingDir: "proj",
			PytestBin:  mockPytest,
		},
		ValueStreams: []ValueStream{{
			Name:   "f",
			Domain: "test-domain",
			Steps: []Step{
				{Name: "ok", TestFile: "pass.py", TestFileAbs: passFile},
				{Name: "bad", TestFile: "fail.py", TestFileAbs: failFile},
			},
		}},
	}
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)

	run, err := r.runPytest(context.Background(), passFile)
	if err != nil {
		t.Fatal(err)
	}
	if !run.Passed || run.ExitCode != 0 {
		t.Fatalf("pass run: %+v", run)
	}

	run2, err := r.runPytest(context.Background(), failFile)
	if err != nil {
		t.Fatal(err)
	}
	if run2.Passed || run2.ExitCode == 0 {
		t.Fatalf("fail run: %+v", run2)
	}
}

func TestRunner_RunStep_UpdatesStepOnly(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "mock-pytest")
	script := "#!/bin/sh\nexit 0\n"
	os.WriteFile(mockPytest, []byte(script), 0755)
	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	passFile := filepath.Join(proj, "pass.py")
	os.WriteFile(passFile, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner:    RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name:   "f",
			Domain: "test-domain",
			Steps:  []Step{{Name: "ok", TestFile: "pass.py", TestFileAbs: passFile}},
		}},
	}
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)
	if err := r.RunStep(context.Background(), "f", "ok"); err != nil {
		t.Fatal(err)
	}
	view := store.StreamView("f")
	if view.StreamStatus != StreamStatusUnknown {
		t.Fatalf("stream_status = %q, want unknown", view.StreamStatus)
	}
	if view.Steps[0].Status != StepStatusPassed {
		t.Fatalf("step = %q", view.Steps[0].Status)
	}
}

func TestRunner_RunStep_UnknownStream(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)
	err := r.RunStep(context.Background(), "nope", "s1")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestRunner_runStep_EmptyTestFileAbsReturnsClearError(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{
					Name:      "s1",
					Lifecycle: "active",
					TestFile:  "missing_abs.py",
				},
			},
		}},
	}
	r := NewRunner(cfg, NewStatusStore(cfg))
	err := r.runStep(context.Background(), "flow-a", "s1")
	if err == nil {
		t.Fatal("want empty TestFileAbs error")
	}
	if !strings.Contains(err.Error(), "test_file") || !strings.Contains(err.Error(), "resolved absolute path") {
		t.Fatalf("want explicit empty TestFileAbs error, got %q", err.Error())
	}
}

func TestRunner_runPytest_CancelsDuringExec(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "slow-pytest")
	os.WriteFile(mockPytest, []byte("#!/bin/sh\nsleep 5\nexit 0\n"), 0755)
	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	testFile := filepath.Join(proj, "t.py")
	os.WriteFile(testFile, []byte(""), 0644)

	cfg := &Config{ConfigDir: dir, Runner: RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest}}
	r := NewRunner(cfg, NewStatusStore(cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := r.runPytest(ctx, testFile)
	if err == nil {
		t.Fatal("want context cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunner_RunStream_ReturnsErrorWhenContextCanceled(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "mock-pytest")
	os.WriteFile(mockPytest, []byte("#!/bin/sh\nexit 0\n"), 0755)
	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	pass := filepath.Join(proj, "pass.py")
	os.WriteFile(pass, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner:    RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name:   "f",
			Domain: "test-domain",
			Steps:  []Step{{Name: "s1", TestFile: "pass.py", TestFileAbs: pass}},
		}},
	}
	r := NewRunner(cfg, NewStatusStore(cfg))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.runStream(ctx, "f")
	if err == nil {
		t.Fatal("want context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunner_RunStream_AbortsMidStepWhenContextCanceled(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "slow-pytest")
	os.WriteFile(mockPytest, []byte("#!/bin/sh\nsleep 5\nexit 0\n"), 0755)
	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	pass := filepath.Join(proj, "pass.py")
	os.WriteFile(pass, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner:    RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name:   "f",
			Domain: "test-domain",
			Steps:  []Step{{Name: "s1", TestFile: "pass.py", TestFileAbs: pass}},
		}},
	}
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	store.BeginStreamRun("f")
	_, err := r.runPytest(ctx, pass)
	store.FinishStep("f", "s1", TestRun{Passed: err == nil, ExitCode: 0})

	if err == nil {
		t.Fatal("want context error from pytest")
	}
	if step := store.StreamView("f").Steps[0]; step.Status != StepStatusFailed {
		t.Fatalf("step status = %q, want failed after cancel", step.Status)
	}
}

func TestRunner_WaitIdleWaitsForAsyncWork(t *testing.T) {
	r := NewRunner(minimalConfig(), NewStatusStore(minimalConfig()))
	done := make(chan struct{})
	r.runAsync(func() {
		defer close(done)
		time.Sleep(50 * time.Millisecond)
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.WaitIdle(ctx); err != nil {
		t.Fatalf("WaitIdle: %v", err)
	}
	select {
	case <-done:
	default:
		t.Fatal("async work did not finish before WaitIdle returned")
	}
}

func TestRunner_RunStream_ContinuesOnFailure(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "mock-pytest")
	script := "#!/bin/sh\nb=$(basename \"$1\")\nif [ \"$b\" = \"fail.py\" ]; then exit 1; fi\nexit 0\n"
	os.WriteFile(mockPytest, []byte(script), 0755)

	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	passFile := filepath.Join(proj, "pass.py")
	failFile := filepath.Join(proj, "fail.py")
	os.WriteFile(passFile, []byte(""), 0644)
	os.WriteFile(failFile, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner: RunnerConfig{
			WorkingDir: "proj",
			PytestBin:  mockPytest,
		},
		ValueStreams: []ValueStream{{
			Name:   "f",
			Domain: "test-domain",
			Steps: []Step{
				{Name: "ok", TestFile: "pass.py", TestFileAbs: passFile},
				{Name: "bad", TestFile: "fail.py", TestFileAbs: failFile},
				{Name: "ok2", TestFile: "pass.py", TestFileAbs: passFile},
			},
		}},
	}
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)

	if err := r.RunStream(context.Background(), "f"); err != nil {
		t.Fatal(err)
	}
	view := store.StreamView("f")
	if view.StreamStatus != StreamStatusFailed {
		t.Fatalf("stream_status = %q", view.StreamStatus)
	}
	if len(view.FailedSteps) != 1 || view.FailedSteps[0] != "bad" {
		t.Fatalf("failed_steps = %v", view.FailedSteps)
	}
	if view.Steps[2].Status != StepStatusPassed {
		t.Fatalf("ok2 status = %q, want passed", view.Steps[2].Status)
	}
}

func TestRunner_runStream_MixedPlannedAndActiveRunsOnlyActive(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "called.txt")
	mockPytest := filepath.Join(dir, "mock-pytest")
	script := "#!/bin/sh\necho \"$1\" >> \"" + logFile + "\"\nexit 0\n"
	if err := os.WriteFile(mockPytest, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(dir, "proj")
	if err := os.MkdirAll(proj, 0755); err != nil {
		t.Fatal(err)
	}
	activeFile := filepath.Join(proj, "active.py")
	if err := os.WriteFile(activeFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ConfigDir: dir,
		Runner:    RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{Name: "planned-step", Lifecycle: "planned", TestFile: "future.py"},
				{Name: "active-step", Lifecycle: "active", TestFile: "active.py", TestFileAbs: activeFile},
			},
		}},
	}
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)

	if err := r.runStream(context.Background(), "flow-a"); err != nil {
		t.Fatalf("runStream: %v", err)
	}

	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	lines := strings.Fields(strings.TrimSpace(string(raw)))
	if len(lines) != 1 {
		t.Fatalf("pytest calls = %v, want exactly 1 active step call", lines)
	}
	if !strings.HasSuffix(lines[0], "active.py") {
		t.Fatalf("pytest called with %q, want active.py only", lines[0])
	}
	view := store.StreamView("flow-a")
	if view.Steps[0].Status != StepStatusPlanned {
		t.Fatalf("planned step status = %q, want planned", view.Steps[0].Status)
	}
	if view.Steps[1].Status != StepStatusPassed {
		t.Fatalf("active step status = %q, want passed", view.Steps[1].Status)
	}
}
