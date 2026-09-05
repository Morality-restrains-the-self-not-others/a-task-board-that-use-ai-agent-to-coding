// Package trajectory records Claude Agent session data to JSON files.
// Output format is compatible with the Python TrajectoryRecorder.
package trajectory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Trajectory represents the full trajectory data structure.
type Trajectory struct {
	Task          string    `json:"task"`
	StartTime     string    `json:"start_time"`
	EndTime       string    `json:"end_time"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	MaxSteps      int       `json:"max_steps"`
	Sessions      []Session `json:"sessions"`
	Success       bool      `json:"success"`
	FinalResult   *string   `json:"final_result"`
	ExecutionTime float64   `json:"execution_time"`
}

// Session records a single Claude Code CLI interaction.
type Session struct {
	SessionID  string    `json:"session_id"`
	Timestamp  string    `json:"timestamp"`
	Messages   []Message `json:"messages"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	ExitCode   int       `json:"exit_code"`
	DurationMs float64   `json:"duration_ms"`
}

// Message represents a user/assistant message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Recorder manages trajectory file writing.
type Recorder struct {
	path  string
	data  *Trajectory
	start time.Time
}

// New creates a new Recorder. If path is empty, auto-generates one under .trajectories/.
func New(path string) *Recorder {
	if path == "" {
		ts := time.Now().Format("20060102_150405")
		path = filepath.Join(".trajectories", "trajectory_"+ts+".json")
	}
	return &Recorder{
		path: path,
		data: &Trajectory{
			Sessions: make([]Session, 0),
		},
	}
}

// Path returns the trajectory file path.
func (r *Recorder) Path() string {
	return r.path
}

// Start begins recording a new task.
func (r *Recorder) Start(task, provider, model string, maxSteps int) {
	r.start = time.Now()
	r.data.Task = task
	r.data.StartTime = r.start.Format(time.RFC3339)
	r.data.Provider = provider
	r.data.Model = model
	r.data.MaxSteps = maxSteps
	r.save()
}

// RecordSession appends a Claude Code session result.
func (r *Recorder) RecordSession(sessionID, stdout, stderr string, exitCode int, durationMs float64) {
	session := Session{
		SessionID: sessionID,
		Timestamp: time.Now().Format(time.RFC3339),
		Messages: []Message{
			{Role: "user", Content: r.data.Task},
			{Role: "assistant", Content: truncate(stdout, 5000)},
		},
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}
	r.data.Sessions = append(r.data.Sessions, session)
	r.save()
}

// Finalize completes the recording.
func (r *Recorder) Finalize(success bool, finalResult string) {
	end := time.Now()
	r.data.EndTime = end.Format(time.RFC3339)
	r.data.Success = success
	if finalResult != "" {
		r.data.FinalResult = &finalResult
	}
	r.data.ExecutionTime = end.Sub(r.start).Seconds()
	r.save()
}

func (r *Recorder) save() {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create trajectory dir %s: %v\n", dir, err)
		return
	}

	f, err := os.Create(r.path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot write trajectory %s: %v\n", r.path, err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r.data); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to encode trajectory: %v\n", err)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
