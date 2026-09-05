package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"runAll/src/domain"
)

func TestClassifyStdioFileLine(t *testing.T) {
	stream, msg := classifyStdioFileLine("2026-08-29T12:00:00Z (stderr) TabError: boom")
	if stream != domain.StreamStderr || msg != "TabError: boom" {
		t.Fatalf("prefixed stderr: stream=%q msg=%q", stream, msg)
	}
	stream, msg = classifyStdioFileLine("2026-08-29T12:00:00Z (stdout) hello")
	if stream != domain.StreamStdout || msg != "hello" {
		t.Fatalf("prefixed stdout: stream=%q msg=%q", stream, msg)
	}
	stream, msg = classifyStdioFileLine(`{"msg":"http_request"}`)
	if stream != domain.StreamStderr || msg != `{"msg":"http_request"}` {
		t.Fatalf("raw child line: stream=%q msg=%q", stream, msg)
	}
}

func TestAttachManagedServiceStdio_FileKeepsWriterAliveAfterParentClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "child.log")
	file, err := openManagedServiceStdioFile(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cmd := exec.Command(getBashPath(), "-c", "while true; do echo tick; sleep 0.05; done")
	cmd.SysProcAttr = managedServiceSysProcAttr()
	attachManagedServiceStdio(cmd, file)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	if err := closeParentStdioFile(file); err != nil {
		t.Fatalf("close parent fd: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var size int64
	for time.Now().Before(deadline) {
		if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
			t.Fatalf("child pid %d died after parent closed stdio file: %v", cmd.Process.Pid, err)
		}
		st, err := os.Stat(path)
		if err == nil && st.Size() > 0 {
			size = st.Size()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if size == 0 {
		t.Fatal("expected child to keep writing after parent closed its log FD copy")
	}

	time.Sleep(150 * time.Millisecond)
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() <= size {
		t.Fatalf("log should keep growing after parent close, size stayed %d", st.Size())
	}
}

func TestPipeStdout_ChildDiesWhenParentClosesReadEnd(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	cmd := exec.Command(getBashPath(), "-c", "while true; do echo tick; sleep 0.05; done")
	cmd.SysProcAttr = managedServiceSysProcAttr()
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close write: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close read: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case waitErr := <-done:
		if waitErr == nil {
			t.Fatal("pipe child should exit after read end closed (SIGPIPE)")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("pipe child still alive 3s after parent closed the pipe — SIGPIPE regression detector failed")
	}
}
