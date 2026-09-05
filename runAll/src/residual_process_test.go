package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// OPT-20260817-001 回归：findResidualRunAllProcesses 只能找到「同二进制、非本
// PID、未监听 ui-port」的残留 runAll，daemon 实例（本就无 UI 端口）必须排除。
//
// 注意：TestHelperResidualSleep 以独立子进程形式运行（spawnResidualHelper 重新
// 执行本测试二进制），父测试在 /proc 中枚举它。若父测试二进制被外部 SIGKILL
// （夜间巡检并发下偶发 OOM），t.Cleanup 不会执行，子进程会成为孤儿——因此子进程
// 的睡眠时长必须**有界**（而非无界 30s），把孤儿存活窗口压到最小。

// helperDaemonFlag 使子测试二进制能接受 -daemon 参数而不被 flag.Parse 报错退出。
// 没有它，带 -daemon 的 helper 在 flag 解析阶段就崩溃退出，ExcludesDaemonChild
// 测例会「恒真通过」而非真正验证 daemon 排除逻辑。
var helperDaemonFlag = flag.Bool("daemon", false, "residual-scan helper daemon mode (no UI port by design)")

// helperSleep 是 helper 子进程的有界睡眠时长：足以覆盖父进程 spawn → 枚举 /proc
// → 断言的全过程，又不会在父进程被杀后长时间残留。
const helperSleep = 10 * time.Second

// TestHelperResidualSleep is the helper entry point for the residual scan tests:
// it sleeps long enough for the parent test to enumerate /proc. It is only active
// when RUNALL_HELPER_SLEEP=1 so it is a no-op during a normal test run.
func TestHelperResidualSleep(t *testing.T) {
	if os.Getenv("RUNALL_HELPER_SLEEP") != "1" {
		return
	}
	// 睡眠有界：父进程正常时会被 t.Cleanup 提前 SIGKILL；父进程异常被杀时，
	// helper 至多再存活 helperSleep，不会成为长时间残留的孤儿。
	time.Sleep(helperSleep)
}

func spawnResidualHelper(t *testing.T, extraArgs ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperResidualSleep")
	cmd.Env = append(os.Environ(), "RUNALL_HELPER_SLEEP=1")
	cmd.Args = append(cmd.Args, extraArgs...)
	// 独立进程组：避免 helper 被其所在组的组信号误伤，也让 t.Cleanup 能按组收割。
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			_, _ = cmd.Process.Wait()
		}
	})
	return cmd
}

// waitForHelperProc 轮询 /proc/PID/exe 直到子进程可被父进程的残留扫描读到。
// 用轮询替代固定 sleep：夜间巡检并发满载时 fork→exec 可能慢于固定延时，
// 轮询保证「子进程已可见」这一扫描前置条件成立，而不是靠运气。
func waitForHelperProc(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
			// 子进程与父进程同二进制（os.Args[0] 重执行），exe 可解析即说明
			// findResidualRunAllProcesses 的 exe 匹配前置条件已满足。
			if strings.Contains(filepath.Base(exe), ".test") {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("helper PID %d never became visible in /proc", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestFindResidualRunAllProcesses_FindsSameBinaryChild(t *testing.T) {
	cmd := spawnResidualHelper(t)
	selfExe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	// Give the child a moment to enter its sleep syscall so /proc/PID/exe resolves.
	waitForHelperProc(t, cmd.Process.Pid)

	residuals := findResidualRunAllProcesses(selfExe, "59999", os.Getpid())
	for _, r := range residuals {
		if r.PID == cmd.Process.Pid {
			return
		}
	}
	t.Fatalf("residual scan did not find same-binary child PID %d; got %#v", cmd.Process.Pid, residuals)
}

func TestFindResidualRunAllProcesses_ExcludesDaemonChild(t *testing.T) {
	cmd := spawnResidualHelper(t, "-daemon")
	selfExe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	waitForHelperProc(t, cmd.Process.Pid)

	residuals := findResidualRunAllProcesses(selfExe, "59999", os.Getpid())
	for _, r := range residuals {
		if r.PID == cmd.Process.Pid {
			t.Fatal("daemon-mode child must be excluded from the residual scan")
		}
	}
}

func TestFindResidualRunAllProcesses_ExcludesSelf(t *testing.T) {
	selfExe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	residuals := findResidualRunAllProcesses(selfExe, "59999", os.Getpid())
	for _, r := range residuals {
		if r.PID == os.Getpid() {
			t.Fatalf("self PID %d must not appear in residuals", os.Getpid())
		}
	}
}
