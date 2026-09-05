package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// OPT-20260812-041：全量编译在内存紧张主机上被 OOM Kill（exit 137 / signal: killed）
// 不可复现失败；runBuild 须在 MemAvailable 过低时等待回收，并对 OOM 自动降并发重试。
// 以下为对应单元与集成测试。

func TestIsOOMExit(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("signal: killed"), true},
		{fmt.Errorf("exit status 137"), true},
		{fmt.Errorf("[taskFE] build failed: exit status 137"), true},
		{fmt.Errorf("[taskFE] build failed: exit status 1"), false},
		{errors.New("command not found"), false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isOOMExit(c.err); got != c.want {
			t.Errorf("isOOMExit(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestMergeEnv(t *testing.T) {
	base := []string{"PATH=/bin", "GOGC=100", "FOO=bar"}
	got := mergeEnv(base, map[string]string{"GOGC": "40", "GOMAXPROCS": "2"})
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4 (replace GOGC, append GOMAXPROCS): %v", len(got), got)
	}
	seenGOGC, seenGOMAX, seenPath := false, false, false
	for _, e := range got {
		switch {
		case e == "GOGC=40":
			seenGOGC = true
		case e == "GOMAXPROCS=2":
			seenGOMAX = true
		case e == "PATH=/bin":
			seenPath = true
		}
	}
	if !seenGOGC || !seenGOMAX || !seenPath {
		t.Errorf("mergeEnv missing expected keys: got %v", got)
	}
	if strings.Contains(strings.Join(got, "\n"), "GOGC=100") {
		t.Errorf("mergeEnv should replace GOGC=100, got %v", got)
	}
	// nil overrides returns original slice unchanged
	if merged := mergeEnv(base, nil); len(merged) != len(base) {
		t.Errorf("nil overrides should not change env, got %v", merged)
	}
}

func TestReadMemAvailableKB(t *testing.T) {
	kb, err := readMemAvailableKB()
	if err != nil {
		t.Skipf("no /proc/meminfo: %v", err)
	}
	if kb <= 0 {
		t.Fatalf("MemAvailable should be positive, got %d", kb)
	}
}

// TestRunBuild_OOMRetrySucceedsWithReducedEnv：首次构建被 OOM Kill，自动降并发重试成功。
func TestRunBuild_OOMRetrySucceedsWithReducedEnv(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "oom-once")
	out := filepath.Join(dir, "env-out.txt")
	buildCmd := fmt.Sprintf(
		`if [ ! -f %[1]q ]; then touch %[1]q; kill -9 $$; fi; printf '%%s' "GOGC=$GOGC GOMAXPROCS=$GOMAXPROCS" > %[2]q; rm -f %[1]q; exit 0`,
		marker, out)

	runner := &Runner{}
	svc := &Service{Name: "test-oom-retry", BuildCommand: buildCmd}
	if err := runner.runBuild(context.Background(), svc, buildCmd); err != nil {
		t.Fatalf("OOM retry should succeed, got: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read env marker: %v", err)
	}
	if !strings.Contains(string(data), "GOGC=40") {
		t.Errorf("retry env should set GOGC=40, got %q", string(data))
	}
	if !strings.Contains(string(data), "GOMAXPROCS=2") {
		t.Errorf("retry env should set GOMAXPROCS=2, got %q", string(data))
	}
}

// TestRunBuild_OOMRetryAlsoFails：OOM 重试仍失败时，错误需明确标注 OOM 而非笼统 build failed。
func TestRunBuild_OOMRetryAlsoFails(t *testing.T) {
	runner := &Runner{}
	svc := &Service{Name: "test-oom-fail", BuildCommand: "kill -9 $$"}
	err := runner.runBuild(context.Background(), svc, svc.BuildCommand)
	if err == nil {
		t.Fatal("expected error for persistent OOM build")
	}
	if !strings.Contains(err.Error(), "OOM") {
		t.Errorf("error should mention OOM, got: %v", err)
	}
}
