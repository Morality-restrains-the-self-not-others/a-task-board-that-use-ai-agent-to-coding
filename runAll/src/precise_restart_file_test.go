package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- 登记文件路径 ----------

func TestPreciseRestartFile_RepoRootFromCfgPath(t *testing.T) {
	root := t.TempDir()
	// 仓库根标记：.ai.md + conf/runAll.yaml 两级结构
	if err := os.MkdirAll(filepath.Join(root, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ai.md"), []byte("# marker"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := preciseRestartFile(filepath.Join(root, "conf", "runAll.yaml"))
	want := filepath.Join(root, ".runall", "precise_restart_services.txt")
	if got != want {
		t.Fatalf("preciseRestartFile = %q, want %q", got, want)
	}
}

func TestPreciseRestartFile_EnvOverride(t *testing.T) {
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", "/tmp/custom-reg.txt")
	if got := preciseRestartFile("/x/conf/runAll.yaml"); got != "/tmp/custom-reg.txt" {
		t.Fatalf("env override: got %q", got)
	}
}

// ---------- 登记文件读写 ----------

func TestReadRegisteredServices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	content := "# 注释行\n\ntask-auth\ntaskBill\ntask-auth # 重复\n  带空格\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readRegisteredServices(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"task-auth", "taskBill", "带空格"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReadRegisteredServices_MissingFileIsEmpty(t *testing.T) {
	got, err := readRegisteredServices(filepath.Join(t.TempDir(), "nope.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestAppendAndClearRegisteredServices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if _, err := appendRegisteredServices(path, []string{"task-auth", "task-bill"}); err != nil {
		t.Fatal(err)
	}
	merged, err := appendRegisteredServices(path, []string{"task-bill", "taskFE"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"task-auth", "task-bill", "taskFE"}
	if strings.Join(merged, ",") != strings.Join(want, ",") {
		t.Fatalf("merged = %v, want %v", merged, want)
	}
	if err := clearRegisteredServices(path); err != nil {
		t.Fatal(err)
	}
	after, err := readRegisteredServices(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Fatalf("expected cleared, got %v", after)
	}
}
