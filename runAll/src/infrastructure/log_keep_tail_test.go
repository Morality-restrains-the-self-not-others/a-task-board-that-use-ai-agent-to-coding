package infrastructure

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTruncateFileKeepTail_RetainsLastBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, runallConsoleLogBaseName)
	payload := bytes.Repeat([]byte("abcdefghij"), 1000) // 10_000 bytes
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	const keep = 1024
	if err := truncateFileKeepTail(path, keep); err != nil {
		t.Fatalf("truncateFileKeepTail: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(got)) != keep {
		t.Fatalf("len=%d, want %d", len(got), keep)
	}
	want := payload[len(payload)-keep:]
	if !bytes.Equal(got, want) {
		t.Fatalf("tail mismatch")
	}
}

func TestTruncateFileKeepTail_SmallFileUnchangedContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.log")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := truncateFileKeepTail(path, 1024); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestFileServiceLogSink_TruncateAll_KeepsConsoleTail(t *testing.T) {
	dir := t.TempDir()
	sink, err := NewFileServiceLogSink(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	console := filepath.Join(dir, runallConsoleLogBaseName)
	big := strings.Repeat("LINE-EXIT-REASON\n", 20000)
	if err := os.WriteFile(console, []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	svcPath := filepath.Join(dir, "svc-a.log")
	if err := os.WriteFile(svcPath, []byte("service noise\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := sink.TruncateAll(); err != nil {
		t.Fatalf("TruncateAll: %v", err)
	}
	svcData, _ := os.ReadFile(svcPath)
	if len(svcData) != 0 {
		t.Fatalf("svc-a.log should be empty, got %d bytes", len(svcData))
	}
	consoleData, err := os.ReadFile(console)
	if err != nil {
		t.Fatal(err)
	}
	if len(consoleData) == 0 {
		t.Fatal("runall-console.log must retain a live tail")
	}
	if int64(len(consoleData)) > runallConsoleKeepTailBytes {
		t.Fatalf("console len=%d exceeds keep=%d", len(consoleData), runallConsoleKeepTailBytes)
	}
	if !bytes.Contains(consoleData, []byte("LINE-EXIT-REASON")) {
		t.Fatal("console tail missing expected content")
	}
}
