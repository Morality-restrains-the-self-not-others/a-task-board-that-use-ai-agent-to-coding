package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// OPT-20260810-042：bulk 操作起止写/删本地锁文件，供 watchdog 离线感知。
func TestBulkOpLockFile_WrittenOnBeginRemovedOnEnd(t *testing.T) {
	regDir := t.TempDir()
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(regDir, "reg.txt"))
	runner, _ := testPreciseRestartRunner(t, &Config{})
	lockPath := runner.bulkOpLockFilePath()

	if !runner.TryBeginBulk("start-all") {
		t.Fatal("TryBeginBulk should succeed")
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock file should exist after begin: %v", err)
	}
	var lock bulkOpLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatalf("lock file should be valid JSON: %v", err)
	}
	if lock.Kind != "start-all" {
		t.Fatalf("lock kind = %q, want start-all", lock.Kind)
	}
	if lock.Ts <= 0 {
		t.Fatalf("lock ts = %d, want > 0", lock.Ts)
	}

	runner.EndBulk("start-all")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock file should be removed after end, stat err=%v", err)
	}
}

// 释放非持有 op 不得误删锁（防并发释放）。
func TestBulkOpLockFile_WrongOpEndKeepsLock(t *testing.T) {
	regDir := t.TempDir()
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(regDir, "reg.txt"))
	runner, _ := testPreciseRestartRunner(t, &Config{})
	lockPath := runner.bulkOpLockFilePath()

	if !runner.TryBeginBulk("stop-all") {
		t.Fatal("TryBeginBulk should succeed")
	}
	runner.EndBulk("restart-all") // 错误 op：不得释放 stop-all 的锁
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock should persist after wrong-op EndBulk, err=%v", err)
	}
	runner.EndBulk("stop-all")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock should be removed after correct EndBulk")
	}
}

// 锁文件路径在注入登记文件时落临时目录（不污染真实仓库 .runall）。
func TestBulkOpLockFilePath_EnvIsolated(t *testing.T) {
	regDir := t.TempDir()
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(regDir, "reg.txt"))
	runner, _ := testPreciseRestartRunner(t, &Config{})
	p := runner.bulkOpLockFilePath()
	if !strings.HasPrefix(p, regDir) {
		t.Fatalf("lock path = %q, want under %q", p, regDir)
	}
	if !strings.HasSuffix(p, "bulk_op.lock") {
		t.Fatalf("lock path = %q, want suffix bulk_op.lock", p)
	}
}
