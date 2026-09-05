package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var errBusy = errors.New("a test is already running")

const logTailMaxLines = 80

type Runner struct {
	cfg     *Config
	store   *StatusStore
	testCtx context.Context

	inFlight sync.WaitGroup
}

func NewRunner(cfg *Config, store *StatusStore) *Runner {
	return &Runner{
		cfg:     cfg,
		store:   store,
		testCtx: context.Background(),
	}
}

func (r *Runner) SetTestContext(ctx context.Context) {
	if ctx == nil {
		r.testCtx = context.Background()
		return
	}
	r.testCtx = ctx
}

func (r *Runner) testContext() context.Context {
	if r.testCtx != nil {
		return r.testCtx
	}
	return context.Background()
}

func (r *Runner) runAsync(fn func()) {
	r.inFlight.Add(1)
	go func() {
		defer r.inFlight.Done()
		fn()
	}()
}

func (r *Runner) WaitIdle(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		r.inFlight.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	override := make(map[string]bool, len(extra))
	for k := range extra {
		override[k] = true
	}
	filtered := make([]string, 0, len(base)+len(extra))
	for _, e := range base {
		key, _, ok := strings.Cut(e, "=")
		if ok && override[key] {
			continue
		}
		filtered = append(filtered, e)
	}
	for k, v := range extra {
		filtered = append(filtered, k+"="+v)
	}
	return filtered
}

func tailLines(s string, n int) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func (r *Runner) runPytest(ctx context.Context, testFileAbs string) (TestRun, error) {
	start := time.Now()
	args := append([]string{}, r.cfg.Runner.PytestArgs...)
	args = append(args, testFileAbs)
	cmd := exec.CommandContext(ctx, r.cfg.Runner.PytestBin, args...)
	cmd.Dir = r.cfg.runnerWorkingDirAbs()
	cmd.Env = mergeEnv(os.Environ(), r.cfg.Runner.Env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := stdout.String() + stderr.String()
	if ctx.Err() != nil {
		return TestRun{}, ctx.Err()
	}
	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			return TestRun{}, err
		}
	}
	return TestRun{
		StartedAt:  start.Format(time.RFC3339),
		DurationMs: time.Since(start).Milliseconds(),
		ExitCode:   exitCode,
		Passed:     exitCode == 0,
		LogTail:    tailLines(combined, logTailMaxLines),
	}, nil
}

func (r *Runner) validateTestTarget(streamName, stepName string, isStep bool) error {
	vs, err := r.cfg.StreamByName(streamName)
	if err != nil {
		return err
	}
	if isStep {
		st, err := vs.StepByName(stepName)
		if err != nil {
			return err
		}
		if st.IsPlanned() || st.IsDeprecated() {
			return fmt.Errorf("step %q is planned or deprecated and cannot be run", stepName)
		}
	}
	return nil
}

func (r *Runner) RunStep(ctx context.Context, streamName, stepName string) error {
	if !r.store.TryAcquire() {
		return errBusy
	}
	defer r.store.Release()
	return r.runStep(ctx, streamName, stepName)
}

func (r *Runner) runStep(ctx context.Context, streamName, stepName string) error {
	vs, err := r.cfg.StreamByName(streamName)
	if err != nil {
		return err
	}
	st, err := vs.StepByName(stepName)
	if err != nil {
		return err
	}
	if st.IsPlanned() {
		return fmt.Errorf("step %q is planned and cannot be run", stepName)
	}
	if st.TestFileAbs == "" {
		return fmt.Errorf("step %q test_file has no resolved absolute path", stepName)
	}

	r.store.SetStepRunning(streamName, stepName)
	run, err := r.runPytest(ctx, st.TestFileAbs)
	if err != nil {
		r.store.FinishStep(streamName, stepName, TestRun{
			StartedAt: time.Now().Format(time.RFC3339),
			ExitCode:  -1,
			Passed:    false,
			LogTail:   err.Error(),
		})
		return nil
	}
	r.store.FinishStep(streamName, stepName, run)
	return nil
}

func (r *Runner) RunStream(ctx context.Context, streamName string) error {
	if !r.store.TryAcquire() {
		return errBusy
	}
	defer r.store.Release()
	return r.runStream(ctx, streamName)
}

func (r *Runner) runStream(ctx context.Context, streamName string) error {
	vs, err := r.cfg.StreamByName(streamName)
	if err != nil {
		return err
	}

	r.store.BeginStreamRun(streamName)
	for _, st := range vs.Steps {
		if st.IsPlanned() || st.IsDeprecated() {
			continue
		}
		if err := ctx.Err(); err != nil {
			r.store.EndStreamRun(streamName)
			return err
		}
		r.store.SetStepRunning(streamName, st.Name)
		run, err := r.runPytest(ctx, st.TestFileAbs)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				r.store.FinishStep(streamName, st.Name, TestRun{
					StartedAt: time.Now().Format(time.RFC3339),
					ExitCode:  -1,
					Passed:    false,
					LogTail:   err.Error(),
				})
				r.store.EndStreamRun(streamName)
				return err
			}
			run = TestRun{
				StartedAt: time.Now().Format(time.RFC3339),
				ExitCode:  -1,
				Passed:    false,
				LogTail:   err.Error(),
			}
		}
		r.store.FinishStep(streamName, st.Name, run)
	}
	r.store.EndStreamRun(streamName)
	return nil
}
