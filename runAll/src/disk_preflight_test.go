package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatBuildDiskPreflightError(t *testing.T) {
	err := formatBuildDiskPreflightError("/tmp/ram-work", 512<<20)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{
		"insufficient disk for build",
		"1GiB",
		"truncate-ram-work-logs.sh",
		"logs/",
		"taskGateway/logs/",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q: %s", want, msg)
		}
	}
}

func TestFormatBytesHuman(t *testing.T) {
	if got := formatBytesHuman(2 << 30); got != "2.00GiB" {
		t.Fatalf("got %q", got)
	}
	if got := formatBytesHuman(512 << 20); got != "512.00MiB" {
		t.Fatalf("got %q", got)
	}
}

func TestCheckBuildDiskSpaceAllowsLargeTmpfs(t *testing.T) {
	// Typical dev tmpfs has >>1GiB free; this guards against regressions on CI runners.
	if err := checkBuildDiskSpace("."); err != nil {
		t.Fatalf("unexpected preflight failure in workspace: %v", err)
	}
}

func TestCheckBuildDiskSpaceFallsBackToAncestorWhenDirMissing(t *testing.T) {
	// 编译输出目录可能尚未创建（编译命令会自行创建目录）；磁盘预检应回退到
	// 最近存在的祖先目录检查所在文件系统空间，而不是因 ENOENT 直接阻断编译。
	missing := filepath.Join(t.TempDir(), "not-yet-created", "nested")
	if err := checkBuildDiskSpace(missing); err != nil {
		t.Fatalf("expected fallback to existing ancestor, got: %v", err)
	}
}
