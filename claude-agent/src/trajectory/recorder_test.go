package trajectory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRecorder(t *testing.T) {
	rec := New("")
	if rec == nil {
		t.Fatal("New() returned nil")
	}
	if rec.Path() == "" {
		t.Error("path should not be empty")
	}
}

func TestRecorderStart(t *testing.T) {
	tmpDir := t.TempDir()
	trajPath := filepath.Join(tmpDir, "test_trajectory.json")
	rec := New(trajPath)

	rec.Start("test task", "anthropic", "claude-sonnet-4-20250514", 100)

	if rec.data.Task != "test task" {
		t.Errorf("task = %q, want 'test task'", rec.data.Task)
	}
	if rec.data.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", rec.data.Provider)
	}
}

func TestRecorderRecordAndFinalize(t *testing.T) {
	tmpDir := t.TempDir()
	trajPath := filepath.Join(tmpDir, "test_trajectory.json")
	rec := New(trajPath)

	rec.Start("test task", "anthropic", "claude-sonnet-4-20250514", 100)
	rec.RecordSession("session_1", "hello world", "", 0, 1234.5)
	rec.Finalize(true, "task completed successfully")

	// Verify file was written
	data, err := os.ReadFile(trajPath)
	if err != nil {
		t.Fatalf("failed to read trajectory file: %v", err)
	}

	var traj Trajectory
	if err := json.Unmarshal(data, &traj); err != nil {
		t.Fatalf("failed to parse trajectory JSON: %v", err)
	}

	if traj.Task != "test task" {
		t.Errorf("task = %q, want 'test task'", traj.Task)
	}
	if len(traj.Sessions) != 1 {
		t.Errorf("sessions count = %d, want 1", len(traj.Sessions))
	}
	if traj.Sessions[0].SessionID != "session_1" {
		t.Errorf("session_id = %q, want session_1", traj.Sessions[0].SessionID)
	}
	if !traj.Success {
		t.Error("success should be true")
	}
	if traj.FinalResult == nil || *traj.FinalResult != "task completed successfully" {
		t.Error("final_result mismatch")
	}
}

func TestRecorderFailure(t *testing.T) {
	tmpDir := t.TempDir()
	trajPath := filepath.Join(tmpDir, "test_trajectory_fail.json")
	rec := New(trajPath)

	rec.Start("failing task", "anthropic", "claude-sonnet-4-20250514", 100)
	rec.RecordSession("session_1", "", "error occurred", 1, 500.0)
	rec.Finalize(false, "")

	data, err := os.ReadFile(trajPath)
	if err != nil {
		t.Fatalf("failed to read trajectory file: %v", err)
	}

	var traj Trajectory
	if err := json.Unmarshal(data, &traj); err != nil {
		t.Fatalf("failed to parse trajectory JSON: %v", err)
	}

	if traj.Success {
		t.Error("success should be false")
	}
	if traj.Sessions[0].ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", traj.Sessions[0].ExitCode)
	}
}
