// Package agent provides the ClaudeAgent implementation that wraps
// the Claude Code CLI via subprocess.
package agent

import (
	"context"
	"fmt"
	"time"

	"claudeAgent/src/config"
	"claudeAgent/src/process"
	"claudeAgent/src/trajectory"
)

// ClaudeAgent wraps the Claude Code CLI as an agent.
type ClaudeAgent struct {
	cfg     *config.ResolvedConfig
	pm      *process.Manager
	rec     *trajectory.Recorder
	task    string
	workDir string
}

// NewClaudeAgent creates a new ClaudeAgent instance.
func NewClaudeAgent(cfg *config.ResolvedConfig, rec *trajectory.Recorder) (*ClaudeAgent, error) {
	pm, err := process.NewManager(cfg.WorkingDir, nil)
	if err != nil {
		return nil, fmt.Errorf("init process manager: %w", err)
	}
	return &ClaudeAgent{
		cfg:     cfg,
		pm:      pm,
		rec:     rec,
		workDir: cfg.WorkingDir,
	}, nil
}

// SetWorkingDir updates the working directory.
func (a *ClaudeAgent) SetWorkingDir(dir string) {
	if dir != "" {
		a.workDir = dir
		var err error
		a.pm, err = process.NewManager(dir, nil)
		if err != nil {
			a.pm = nil
		}
	}
}

// Run executes a task synchronously via `claude -p`.
func (a *ClaudeAgent) Run(ctx context.Context, task string) (*Execution, error) {
	a.task = task
	execution := &Execution{
		Task:  task,
		Steps: make([]Step, 0),
	}
	startTime := time.Now()

	execution.State = StateRunning
	step := Step{StepNumber: 1, State: StepThinking}

	// Start trajectory recording
	if a.rec != nil {
		a.rec.Start(task, a.cfg.Provider, a.cfg.Model, a.cfg.MaxSteps)
	}

	result, runErr := a.pm.RunTask(ctx, task, &process.RunOptions{
		Model:           a.cfg.Model,
		MaxSteps:        a.cfg.MaxSteps,
		SkipPermissions: a.cfg.PermissionMode == "skip",
		APIKey:          a.cfg.APIKey,
		BaseURL:         a.cfg.BaseURL,
	})

	if runErr != nil && result == nil {
		execution.State = StateError
		execution.ErrorMessage = runErr.Error()
		execution.Success = false
		a.recordSession(execution, "", runErr.Error(), -1, 0)
		execution.ExecutionTime = time.Since(startTime).Seconds()
		return execution, nil
	}

	step.State = StepCompleted
	step.LLMResponse = truncateStr(result.Stdout, 1000)

	if result.ExitCode != 0 {
		step.Error = result.Stderr
		step.State = StepError
		execution.State = StateError
		execution.ErrorMessage = result.Stderr
		execution.Success = false
	} else {
		execution.State = StateCompleted
		execution.Success = true
		execution.FinalResult = result.Stdout
	}

	execution.Steps = append(execution.Steps, step)
	a.recordSession(execution, result.Stdout, result.Stderr, result.ExitCode, float64(result.DurationMs))
	execution.ExecutionTime = time.Since(startTime).Seconds()

	// Finalize trajectory
	if a.rec != nil {
		a.rec.Finalize(execution.Success, execution.FinalResult)
	}

	return execution, nil
}

// RunInteractive starts claude in interactive mode (foreground passthrough).
func (a *ClaudeAgent) RunInteractive() (int, error) {
	return a.pm.RunInteractive(&process.RunOptions{
		Model: a.cfg.Model,
	})
}

func (a *ClaudeAgent) recordSession(exec *Execution, stdout, stderr string, exitCode int, durationMs float64) {
	if a.rec == nil {
		return
	}
	sessionID := fmt.Sprintf("claude_%d", time.Now().Unix())
	a.rec.RecordSession(sessionID, stdout, stderr, exitCode, durationMs)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
