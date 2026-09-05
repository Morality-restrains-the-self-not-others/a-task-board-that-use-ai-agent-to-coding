package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"runAll/src/domain"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func TestFileDevToolLogRecorder_AppendAndTail(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	if err := rec.Append(domain.ToolDbClear, "line 1"); err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if err := rec.Append(domain.ToolDbClear, "line 2"); err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if err := rec.Append(domain.ToolDbClear, "line 3"); err != nil {
		t.Fatalf("Append error: %v", err)
	}

	lines, err := rec.Tail(domain.ToolDbClear, 2)
	if err != nil {
		t.Fatalf("Tail error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("Tail returned %d lines, want 2", len(lines))
	}
	if !strings.Contains(lines[0], "line 2") {
		t.Errorf("first tail line should be 'line 2', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "line 3") {
		t.Errorf("second tail line should be 'line 3', got %q", lines[1])
	}
}

func TestFileDevToolLogRecorder_TailEmptyFile(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	// File doesn't exist yet — should return empty without error
	lines, err := rec.Tail(domain.ToolConfSync, 100)
	if err != nil {
		t.Fatalf("Tail on non-existent file should not error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("Tail on non-existent file returned %d lines, want 0", len(lines))
	}
}

func TestFileDevToolLogRecorder_Clear(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	rec.Append(domain.ToolDbInit, "some log")
	rec.Append(domain.ToolDbInit, "more log")

	if err := rec.Clear(domain.ToolDbInit); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	lines, _ := rec.Tail(domain.ToolDbInit, 10)
	if len(lines) != 0 {
		t.Errorf("after Clear, Tail returned %d lines, want 0", len(lines))
	}
}

func TestFileDevToolLogRecorder_ClearAll(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	for _, tool := range domain.AllDevTools {
		rec.Append(tool, "test")
	}

	if err := rec.ClearAll(); err != nil {
		t.Fatalf("ClearAll error: %v", err)
	}

	for _, tool := range domain.AllDevTools {
		lines, _ := rec.Tail(tool, 10)
		if len(lines) != 0 {
			t.Errorf("after ClearAll, Tail(%q) returned %d lines, want 0", tool, len(lines))
		}
	}
}

func TestFileDevToolLogRecorder_InvalidTool(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	if err := rec.Append("bad-tool", "msg"); err == nil {
		t.Error("Append with invalid tool should error")
	}
	if _, err := rec.Tail("bad-tool", 10); err == nil {
		t.Error("Tail with invalid tool should error")
	}
	if err := rec.Clear("bad-tool"); err == nil {
		t.Error("Clear with invalid tool should error")
	}
}

func TestFileDevToolLogRecorder_LogDirCreation(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	// Remove the default logs directory
	logDir := filepath.Join(root, "logs")
	os.RemoveAll(logDir)

	if err := rec.Append(domain.ToolConfSync, "auto-create dir"); err != nil {
		t.Fatalf("Append should auto-create logs dir: %v", err)
	}

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("logs directory was not auto-created")
	}
}

func TestFileDevToolLogRecorder_ConcurrentAppend(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			rec.Append(domain.ToolObservabilityClear, "concurrent write")
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	lines, err := rec.Tail(domain.ToolObservabilityClear, 100)
	if err != nil {
		t.Fatalf("Tail after concurrent writes: %v", err)
	}
	if len(lines) != 5 {
		t.Errorf("concurrent Append: got %d lines, want 5", len(lines))
	}
}

func TestFileDevToolLogRecorder_ClearNonExistent(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	// Clear on a file that doesn't exist should not error
	if err := rec.Clear(domain.ToolConfSyncRemote); err != nil {
		t.Errorf("Clear on non-existent file should not error: %v", err)
	}
}

func TestFileDevToolLogRecorder_TailManyLines(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	// Write 500 lines — more than the buffer can hold without looping
	for i := 0; i < 500; i++ {
		rec.Append(domain.ToolDbClear, fmt.Sprintf("line %04d", i))
	}

	// Request fewer lines than written
	lines, err := rec.Tail(domain.ToolDbClear, 10)
	if err != nil {
		t.Fatalf("Tail error: %v", err)
	}
	if len(lines) != 10 {
		t.Fatalf("Tail returned %d lines, want 10", len(lines))
	}
	if !strings.Contains(lines[0], "line 0490") {
		t.Errorf("first line should contain 'line 0490', got %q", lines[0])
	}
	if !strings.Contains(lines[9], "line 0499") {
		t.Errorf("last line should contain 'line 0499', got %q", lines[9])
	}
}

func TestFileDevToolLogRecorder_TailMoreLinesThanFile(t *testing.T) {
	root := tempDir(t)
	rec := NewFileDevToolLogRecorder(root)

	// Write only 3 lines
	rec.Append(domain.ToolDbInit, "a")
	rec.Append(domain.ToolDbInit, "b")
	rec.Append(domain.ToolDbInit, "c")

	// Request more lines than exist
	lines, err := rec.Tail(domain.ToolDbInit, 100)
	if err != nil {
		t.Fatalf("Tail error: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("Tail returned %d lines, want 3 (all lines)", len(lines))
	}
}
