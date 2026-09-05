package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// minBuildDiskBytes is the minimum free space required before running build_command.
const minBuildDiskBytes = int64(1 << 30) // 1 GiB

func availableBytesAtPath(path string) (int64, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

func formatBuildDiskPreflightError(workDir string, avail int64) error {
	dfOut, _ := exec.Command("df", "-h", workDir).CombinedOutput()
	dfLine := strings.TrimSpace(string(dfOut))
	if dfLine == "" {
		dfLine = "(df unavailable)"
	}
	return fmt.Errorf(
		"insufficient disk for build: need at least 1GiB free at %q (available=%s); %s; try: bash runAll/scripts/truncate-ram-work-logs.sh --all; truncate logs/ and taskGateway/logs/; remove old runAll binaries (taskEvents prune-legacy-binaries)",
		workDir,
		formatBytesHuman(avail),
		dfLine,
	)
}

func formatBytesHuman(n int64) string {
	const gib = int64(1 << 30)
	const mib = int64(1 << 20)
	if n >= gib {
		return fmt.Sprintf("%.2fGiB", float64(n)/float64(gib))
	}
	return fmt.Sprintf("%.2fMiB", float64(n)/float64(mib))
}

func checkBuildDiskSpace(workDir string) error {
	path := strings.TrimSpace(workDir)
	if path == "" {
		path = "."
	}
	avail, err := availableBytesAtPath(path)
	if err != nil {
		// 编译输出目录可能尚未创建（build_command 会自行 mkdir），
		// statfs ENOENT 不代表磁盘不足：回退到最近存在的祖先目录
		// 检查所在文件系统的可用空间，避免误阻断编译。
		fallback, ok := nearestExistingAncestor(path)
		if !ok {
			fallback = "."
		}
		avail, err = availableBytesAtPath(fallback)
		if err != nil {
			return fmt.Errorf("disk preflight statfs %q (fallback %q): %w", path, fallback, err)
		}
		path = fallback
	}
	if avail >= minBuildDiskBytes {
		return nil
	}
	return formatBuildDiskPreflightError(path, avail)
}

// nearestExistingAncestor 返回 path 及其逐级祖先中第一个存在的目录；全部不存在时返回 false。
func nearestExistingAncestor(path string) (string, bool) {
	cur := filepath.Clean(path)
	for {
		fi, err := os.Stat(cur)
		if err == nil && fi.IsDir() {
			return cur, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false
		}
		cur = parent
	}
}
