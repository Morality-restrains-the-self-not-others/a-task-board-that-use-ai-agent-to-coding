package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// helper 须清空登记并写入 consumed-at 水位线。
func TestClearPreciseRestartRegistrationsAfterFullRebuild(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "precise_restart_services.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "svc-pending", State: RegistrationStatePending, RegisteredAt: 100},
		{Name: "svc-failed", State: RegistrationStateFailed, RegisteredAt: 110},
	}); err != nil {
		t.Fatal(err)
	}
	before := time.Now().Unix()
	if err := clearPreciseRestartRegistrationsAfterFullRebuild(""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	after, err := readRegistrationEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Fatalf("registrations = %+v, want empty", after)
	}
	raw, err := os.ReadFile(preciseRestartConsumedAtFile(path))
	if err != nil {
		t.Fatalf("consumed-at missing: %v", err)
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || ts < before {
		t.Fatalf("consumed-at = %q (parsed %d), want unix >= %d", raw, ts, before)
	}
}

// 页头全部重新编译正常完成后须消费精准编译登记（约束 42）。
func TestBuildAll_ClearsPreciseRestartRegistrations(t *testing.T) {
	dir := t.TempDir()
	store := NewStatusStore()
	store.Init([]string{"ba-clear-a"})
	store.Update("ba-clear-a", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "ba-clear",
			Services: []Service{{
				Name:         "ba-clear-a",
				Command:      "true",
				BuildCommand: "echo built",
				WorkingDir:   dir,
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "ba-clear-a", State: RegistrationStatePending, RegisteredAt: 100},
		{Name: "other-svc", State: RegistrationStateFailed, RegisteredAt: 110},
	}); err != nil {
		t.Fatal(err)
	}

	before := time.Now().Unix()
	result, err := runner.BuildAll(context.Background())
	if err != nil {
		t.Fatalf("BuildAll: %v", err)
	}
	if result.Built != 1 {
		t.Fatalf("built = %d, want 1", result.Built)
	}

	entries, err := readRegistrationEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("registrations after BuildAll = %+v, want empty", entries)
	}
	raw, err := os.ReadFile(preciseRestartConsumedAtFile(path))
	if err != nil {
		t.Fatalf("consumed-at missing after BuildAll: %v", err)
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || ts < before {
		t.Fatalf("consumed-at = %q (parsed %d), want unix >= %d", raw, ts, before)
	}
}

// 中断取消时不得清空登记，以便后续精准编译重启仍能消费剩余项。
func TestBuildAll_AbortKeepsPreciseRestartRegistrations(t *testing.T) {
	dir := t.TempDir()
	store := NewStatusStore()
	store.Init([]string{"ba-abort-a"})
	store.Update("ba-abort-a", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "ba-abort",
			Services: []Service{{
				Name:         "ba-abort-a",
				Command:      "true",
				BuildCommand: "echo built",
				WorkingDir:   dir,
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	want := []RegistrationEntry{
		{Name: "ba-abort-a", State: RegistrationStatePending, RegisteredAt: 100},
	}
	if err := writeRegistrationEntries(path, want); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = runner.BuildAll(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildAll err = %v, want context.Canceled", err)
	}

	entries, err := readRegistrationEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "ba-abort-a" {
		t.Fatalf("registrations after abort = %+v, want [ba-abort-a]", entries)
	}
}
