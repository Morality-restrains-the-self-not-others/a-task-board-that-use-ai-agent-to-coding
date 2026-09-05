package main

// 进程组停止与 compile-then-swap 行为测试：
//   1. stopProcess 必须等待进程组消亡（而非组长进程退出），SIGKILL 升级必须真正生效；
//   2. ADR-0027：带 compile-then-swap 的重启先编译再停；编译失败保留旧进程为 healthy；
//   3. startAndCheck 在 restart 上下文（forceFreshStart）下不得「跳过启动」，
//      必须等待端口释放后启动新二进制。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestStopProcess_WaitsForGroupExitAndEscalates 复现 12:15 缺陷场景：
// start_command 为 bash 包装（bash 组长 + 忽略 SIGTERM 的服务子进程）。
// 旧实现：bash 组长收到 SIGTERM 立即退出 → cmd.Wait() 快速返回 → 打 "stopped for
// restart" → 5s SIGKILL 升级被短路 → 服务子进程存活、端口继续被监听。
// 新实现：必须等待进程组消亡；组长已死但组内仍有进程 → 5s 后 SIGKILL → 组消亡。
func TestStopProcess_WaitsForGroupExitAndEscalates(t *testing.T) {
	// READY 握手确保子进程完成 SIG_IGN/SIGHUP_IGN 注册后再触发 stopProcess
	// （否则 SIGTERM 落在注册窗口前，子进程按默认动作死亡，测不出缺陷）。
	// `& wait` 复合命令阻止 bash 的单命令 exec 优化，组长必须是 bash。
	py := "import signal,time,sys; signal.signal(signal.SIGTERM, signal.SIG_IGN); signal.signal(signal.SIGHUP, signal.SIG_IGN); print('READY', flush=True); time.sleep(60)"
	cmd := exec.Command("/bin/bash", "-c", `python3 -c "`+py+`" & wait`)
	var outBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start bash wrapper: %v", err)
	}
	pgid := cmd.Process.Pid
	t.Cleanup(func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	})
	deadline := time.Now().Add(3 * time.Second)
	for !strings.Contains(outBuf.String(), "READY") {
		if time.Now().After(deadline) {
			t.Fatalf("child never signaled READY; stdout=%q", outBuf.String())
		}
		time.Sleep(20 * time.Millisecond)
	}

	runner := &Runner{}
	runner.mu.Lock()
	runner.processes = map[string]*exec.Cmd{"grp-svc": cmd}
	runner.mu.Unlock()

	start := time.Now()
	ok := runner.stopProcess("grp-svc")
	elapsed := time.Since(start)
	if !ok {
		t.Fatal("stopProcess should return true")
	}
	if elapsed > 15*time.Second {
		t.Fatalf("stopProcess took %v, expected SIGKILL escalation within ~5s", elapsed)
	}
	// 进程组必须已消亡（kill(-pgid, 0) 返回 ESRCH）。
	if err := syscall.Kill(-pgid, 0); err == nil {
		t.Error("process group still alive after stopProcess — SIGKILL escalation was short-circuited")
	}
}

// TestStopProcess_FastExit 正常进程（SIGTERM 即死）不应引入额外延迟。
func TestStopProcess_FastExit(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pgid := cmd.Process.Pid
	t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })

	runner := &Runner{}
	runner.mu.Lock()
	runner.processes = map[string]*exec.Cmd{"fast-svc": cmd}
	runner.mu.Unlock()

	start := time.Now()
	if !runner.stopProcess("fast-svc") {
		t.Fatal("stopProcess should return true")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("stopProcess took %v for a SIGTERM-responsive process", elapsed)
	}
	if err := syscall.Kill(-pgid, 0); err == nil {
		t.Error("process group still alive after stopProcess")
	}
}

// TestRestartService_CompileThenSwap_BuildFailureKeepsProcess ADR-0027：
// 精准编译路径先编后切；编译失败时旧进程必须仍存活，状态保持 Healthy。
func TestRestartService_CompileThenSwap_BuildFailureKeepsProcess(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "compile-swap",
					BuildCommand: "exit 9",
					Command:      "sleep 60",
					HealthCheck: HealthCheck{
						URL:           healthServer.URL,
						Timeout:       2,
						Retries:       2,
						CheckInterval: 1,
						Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
					},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("compile-swap", StatusHealthy, "")

	proc := exec.Command("sleep", "60")
	proc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := proc.Start(); err != nil {
		t.Fatalf("start old process: %v", err)
	}
	procPgid := proc.Process.Pid
	t.Cleanup(func() {
		runner.stopMonitoring("compile-swap")
		_ = syscall.Kill(-procPgid, syscall.SIGKILL)
	})
	runner.mu.Lock()
	runner.processes["compile-swap"] = proc
	runner.mu.Unlock()

	err = runner.RestartService(withCompileThenSwap(context.Background()), "compile-swap")
	if err == nil {
		t.Fatal("expected build failure error")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Errorf("error should mention build failed, got: %v", err)
	}

	if perr := syscall.Kill(proc.Process.Pid, 0); perr != nil {
		t.Errorf("old process must stay alive after failed compile-then-swap: %v", perr)
	}
	runner.mu.Lock()
	_, exists := runner.processes["compile-swap"]
	runner.mu.Unlock()
	if !exists {
		t.Error("process handle must be kept when compile fails before stop")
	}

	status := store.Get("compile-swap")
	if status == nil {
		t.Fatal("status should exist")
	}
	if status.Status != StatusHealthy {
		t.Errorf("status = %s, want %s (last-good process still running)", status.Status, StatusHealthy)
	}
}

// TestStartAndCheck_ForceFreshStartNeverSkipsStart restart 上下文（forceFreshStart=true）：
// 端口被残留进程占用时不得「跳过启动」——终止残留、等待释放后启动新进程。
func TestStartAndCheck_ForceFreshStartNeverSkipsStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "fresh-svc",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:           srv.URL,
						Timeout:       5,
						Retries:       5,
						CheckInterval: 1,
						Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
					},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	// 模拟残留监听：首轮探测有 PID，terminate 后端口释放（不再强制占用启动）。
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	calls := 0
	runner.listenerPIDsFn = func(string) ([]int, error) {
		calls++
		if calls <= 2 { // 启动前检查 + terminate 列举
			return []int{2410097}, nil
		}
		return nil, nil
	}
	t.Cleanup(func() { runner.stopProcess("fresh-svc") })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	node := &ServiceNode{Service: *runner.findService("fresh-svc"), forceFreshStart: true}
	err = runner.startAndCheck(ctx, node)
	if err != nil {
		t.Fatalf("startAndCheck with forceFreshStart: %v", err)
	}

	// 必须真正启动了新进程（非跳过）。
	runner.mu.Lock()
	_, started := runner.processes["fresh-svc"]
	runner.mu.Unlock()
	if !started {
		t.Fatal("forceFreshStart must launch the new process, not skip start")
	}
	status := store.Get("fresh-svc")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %+v, want healthy after fresh start", status)
	}
}

// TestStartAndCheck_ForceFreshStartFailsWhenPortStuck 端口终止后仍占用：
// 必须失败返回，禁止强行启动导致 EADDRINUSE。
func TestStartAndCheck_ForceFreshStartFailsWhenPortStuck(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "stuck-port",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:           "http://127.0.0.1:18999/health",
						Timeout:       2,
						Retries:       1,
						CheckInterval: 1,
					},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 200 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	// 始终报告占用：terminate 跳过自身 PID，等待必超时。
	self := []int{os.Getpid()}
	runner.listenerPIDsFn = func(string) ([]int, error) { return self, nil }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := &ServiceNode{Service: *runner.findService("stuck-port"), forceFreshStart: true}
	err = runner.startAndCheck(ctx, node)
	if err == nil {
		t.Fatal("expected failure when port cannot be freed")
	}
	if !strings.Contains(err.Error(), "still occupied") {
		t.Fatalf("error should mention port still occupied, got: %v", err)
	}
	runner.mu.Lock()
	_, started := runner.processes["stuck-port"]
	runner.mu.Unlock()
	if started {
		t.Fatal("must not launch new process when port remains occupied")
	}
	status := store.Get("stuck-port")
	if status == nil || status.Status != StatusFailed {
		t.Fatalf("status = %+v, want failed", status)
	}
}

// TestStartAndCheck_HealthyPortSkipsWithoutForce 对照：普通 start（非 restart 上下文）
// 端口占用且健康时保持既有「跳过启动」语义（收养/防重复启动），行为不得回归。
func TestStartAndCheck_HealthyPortSkipsWithoutForce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "skip-svc",
					Command: "sleep 30",
					HealthCheck: HealthCheck{
						URL:           srv.URL,
						Timeout:       5,
						Retries:       5,
						CheckInterval: 1,
						Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
					},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) { return []int{2410097}, nil }

	node := &ServiceNode{Service: *runner.findService("skip-svc")}
	if err := runner.startAndCheck(context.Background(), node); err != nil {
		t.Fatalf("startAndCheck without force: %v", err)
	}

	// 跳过启动：不得产生新进程句柄，状态为 healthy（接管既有监听者）。
	runner.mu.Lock()
	_, started := runner.processes["skip-svc"]
	runner.mu.Unlock()
	if started {
		t.Fatal("non-restart start must keep skip semantics, no new process expected")
	}
	status := store.Get("skip-svc")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("status = %+v, want healthy via skip", status)
	}
}
