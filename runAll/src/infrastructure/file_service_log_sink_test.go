package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileServiceLogSink_AppendLine(t *testing.T) {
	root := t.TempDir()
	sink, err := NewFileServiceLogSink(root)
	if err != nil {
		t.Fatalf("NewFileServiceLogSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	if err := sink.AppendLine("saas-backend", "stdout", "hello trace"); err != nil {
		t.Fatalf("AppendLine: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "saas-backend.log"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "(stdout) hello trace") {
		t.Fatalf("unexpected log content: %q", content)
	}
}

func TestFileServiceLogSink_TruncateService(t *testing.T) {
	root := t.TempDir()
	sink, err := NewFileServiceLogSink(root)
	if err != nil {
		t.Fatalf("NewFileServiceLogSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	if err := sink.AppendLine("saas-backend", "stdout", "line-one"); err != nil {
		t.Fatalf("AppendLine: %v", err)
	}
	if err := sink.TruncateService("saas-backend"); err != nil {
		t.Fatalf("TruncateService: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "saas-backend.log"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty file after truncate, got %q", string(data))
	}

	if err := sink.AppendLine("saas-backend", "stdout", "line-two"); err != nil {
		t.Fatalf("AppendLine after truncate: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(root, "saas-backend.log"))
	if err != nil {
		t.Fatalf("ReadFile after append: %v", err)
	}
	if !strings.Contains(string(data), "line-two") {
		t.Fatalf("unexpected content after re-append: %q", string(data))
	}
}

func TestFileServiceLogSink_TruncateAll(t *testing.T) {
	root := t.TempDir()
	sink, err := NewFileServiceLogSink(root)
	if err != nil {
		t.Fatalf("NewFileServiceLogSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	_ = sink.AppendLine("task-auth", "stdout", "a")
	_ = sink.AppendLine("task-bill", "stderr", "b")

	count, err := sink.TruncateAll()
	if err != nil {
		t.Fatalf("TruncateAll: %v", err)
	}
	if count != 2 {
		t.Fatalf("truncated = %d, want 2", count)
	}
	for _, name := range []string{"task-auth.log", "task-bill.log"} {
		data, readErr := os.ReadFile(filepath.Join(root, name))
		if readErr != nil {
			t.Fatalf("ReadFile %s: %v", name, readErr)
		}
		if len(data) != 0 {
			t.Fatalf("%s not empty after TruncateAll", name)
		}
	}
}

func TestFileServiceLogSink_RequiresRoot(t *testing.T) {
	_, err := NewFileServiceLogSink("")
	if err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestFileServiceLogSink_ServiceIsolation(t *testing.T) {
	root := t.TempDir()
	sink, err := NewFileServiceLogSink(root)
	if err != nil {
		t.Fatalf("NewFileServiceLogSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	_ = sink.AppendLine("task-auth", "stderr", "err-a")
	_ = sink.AppendLine("task-auth", "stdout", "out-a")

	taskAuth, _ := os.ReadFile(filepath.Join(root, "task-auth.log"))
	if strings.Contains(string(taskAuth), "saas-backend") {
		t.Fatal("expected isolated log files per service")
	}
}

func TestFileServiceLogSink_ReopensAfterExternalUnlink(t *testing.T) {
	root := t.TempDir()
	sink, err := NewFileServiceLogSink(root)
	if err != nil {
		t.Fatalf("NewFileServiceLogSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	path := filepath.Join(root, "saas-backend.log")
	if err := sink.AppendLine("saas-backend", "stdout", "before-unlink"); err != nil {
		t.Fatalf("AppendLine before unlink: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// External writer recreates the path Promtail will tail (same failure mode as clear/rm + service restart).
	if err := os.WriteFile(path, []byte("recreated-by-service\n"), 0o644); err != nil {
		t.Fatalf("WriteFile recreate: %v", err)
	}

	if err := sink.AppendLine("saas-backend", "stdout", "after-reopen probe-xyz"); err != nil {
		t.Fatalf("AppendLine after unlink: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "after-reopen probe-xyz") {
		t.Fatalf("expected probe on recreated path, got %q", content)
	}
	if strings.Contains(content, "before-unlink") {
		t.Fatalf("stale inode content leaked into new path: %q", content)
	}
}

func TestFileServiceLogSink_TruncateAllRecoversDeletedOpenFiles(t *testing.T) {
	root := t.TempDir()
	sink, err := NewFileServiceLogSink(root)
	if err != nil {
		t.Fatalf("NewFileServiceLogSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	path := filepath.Join(root, "task-auth.log")
	if err := sink.AppendLine("task-auth", "stdout", "old"); err != nil {
		t.Fatalf("AppendLine: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := os.WriteFile(path, []byte("external\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := sink.TruncateAll(); err != nil {
		t.Fatalf("TruncateAll: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("path should be empty after TruncateAll, got %q", string(data))
	}
	if err := sink.AppendLine("task-auth", "stdout", "fresh"); err != nil {
		t.Fatalf("AppendLine after TruncateAll: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after append: %v", err)
	}
	if !strings.Contains(string(data), "fresh") {
		t.Fatalf("expected fresh line on path, got %q", string(data))
	}
}
