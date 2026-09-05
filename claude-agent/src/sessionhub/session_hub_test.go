// Package sessionhub — 多智能体会话互知与冲突防护（agent-session-coordination v14/v66）
// 测试先行：注册表 / 仓库级锁(租约+心跳+窃取) / 死锁检测 / 暂停恢复 / Shadow Edit
package sessionhub

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ---- helpers ----

func newTestHub(t *testing.T) (*Hub, string) {
	t.Helper()
	root := t.TempDir()
	// 模拟 meta root：.gitmodules 标记
	if err := os.WriteFile(filepath.Join(root, ".gitmodules"), []byte("[submodule \"db\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	return h, root
}

func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "0.1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Process.Release()
	time.Sleep(300 * time.Millisecond) // 等它退出
	return cmd.Process.Pid
}

func alivePID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	return cmd.Process.Pid
}

func staleSession(h *Hub, sid string, pid int) {
	s, _ := h.get(sid)
	s.HeartbeatAt = time.Now().Add(-30 * time.Minute).Format(time.RFC3339)
	_ = h.put(s)
}

// ---- 注册表 ----

func TestRegisterAndList(t *testing.T) {
	h, _ := newTestHub(t)
	pid := alivePID(t)
	sid, err := h.Register(Session{Kind: "interactive", PID: pid, StartCwd: "/tmp/ram-work", Note: "test"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if sid == "" {
		t.Fatal("empty sid")
	}
	list, err := h.List(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].SessionID != sid || list[0].Status != "active" {
		t.Fatalf("list = %+v", list)
	}
}

func TestUnregister(t *testing.T) {
	h, _ := newTestHub(t)
	pid := alivePID(t)
	sid, _ := h.Register(Session{Kind: "headless", PID: pid})
	if err := h.Unregister(sid); err != nil {
		t.Fatal(err)
	}
	list, _ := h.List(false)
	if len(list) != 0 {
		t.Fatalf("expected empty after unregister, got %d", len(list))
	}
}

func TestListMarksDeadPID(t *testing.T) {
	h, _ := newTestHub(t)
	sid, _ := h.Register(Session{Kind: "headless", PID: deadPID(t)})
	time.Sleep(100 * time.Millisecond)
	list, err := h.List(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range list {
		if s.SessionID == sid {
			t.Fatal("dead session still listed as active")
		}
	}
}

// ---- 锁 ----

func TestAcquireFreeThenConflict(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	b, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})

	res, lock, err := h.Acquire("db", a, 0)
	if err != nil || res != Acquired || lock.Holder != a {
		t.Fatalf("acquire free: res=%v lock=%+v err=%v", res, lock, err)
	}

	res2, lock2, err := h.Acquire("db", b, 0)
	if err != nil || res2 != HeldBy || lock2.Holder != a {
		t.Fatalf("acquire conflict: res=%v lock=%+v err=%v", res2, lock2, err)
	}
}

func TestAcquireStaleDeadSteal(t *testing.T) {
	h, _ := newTestHub(t)
	dead := deadPID(t)
	a, _ := h.Register(Session{Kind: "headless", PID: dead})
	staleSession(h, a, dead)
	b, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})

	if _, _, err := h.Acquire("db", a, 0); err != nil {
		t.Fatal(err)
	}
	res, lock, err := h.Acquire("db", b, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res != Stolen {
		t.Fatalf("expected steal, got res=%v lock=%+v", res, lock)
	}
	if lock.Holder != b {
		t.Fatalf("lock should be re-held by %s, got %s", b, lock.Holder)
	}
}

func TestAcquireStaleAliveNoSteal(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	b, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	staleSession(h, a, 0) // pid 参数忽略 — 用真实 alive pid 场景
	s, _ := h.get(a)
	s.HeartbeatAt = time.Now().Add(-30 * time.Minute).Format(time.RFC3339)
	_ = h.put(s)

	_, _, _ = h.Acquire("db", a, 0)
	res, _, err := h.Acquire("db", b, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res != HeldBy {
		t.Fatalf("stale-but-alive must not be stolen, got %v", res)
	}
}

func TestOrphanLockSteal(t *testing.T) {
	h, _ := newTestHub(t)
	// 直接制造孤儿锁：holder 会话不存在
	orphan := "sess_orphan_dead"
	_ = h.writeLock("db", &Lock{Repo: "db", Holder: orphan, AcquiredAt: time.Now().Format(time.RFC3339), ExpiresAt: time.Now().Add(10 * time.Minute).Format(time.RFC3339)})
	b, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})

	res, lock, err := h.Acquire("db", b, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res != Stolen || lock.Holder != b {
		t.Fatalf("orphan lock must be stolen: res=%v lock=%+v", res, lock)
	}
}

func TestRelease(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	_, _, _ = h.Acquire("db", a, 0)
	if err := h.Release("db", a); err != nil {
		t.Fatal(err)
	}
	_, held, _ := h.Check("db")
	if held {
		t.Fatal("lock should be free after release")
	}
}

func TestCheckAfterUnregisterReleasesLocks(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	_, _, _ = h.Acquire("db", a, 0)
	_, _, _ = h.Acquire("taskAuth", a, 0)
	if err := h.Unregister(a); err != nil {
		t.Fatal(err)
	}
	_, held, _ := h.Check("db")
	if held {
		t.Fatal("db lock should be released on unregister")
	}
	_, held2, _ := h.Check("taskAuth")
	if held2 {
		t.Fatal("taskAuth lock should be released on unregister")
	}
}

// ---- 心跳 ----

func TestHeartbeat(t *testing.T) {
	h, _ := newTestHub(t)
	pid := alivePID(t)
	sid, _ := h.Register(Session{Kind: "interactive", PID: pid})
	staleSession(h, sid, pid)
	if err := h.Heartbeat(sid, pid); err != nil {
		t.Fatal(err)
	}
	s, _ := h.get(sid)
	ts, _ := time.Parse(time.RFC3339, s.HeartbeatAt)
	if time.Since(ts) > 2*time.Second {
		t.Fatalf("heartbeat not refreshed: %s", s.HeartbeatAt)
	}
}

func TestHeartbeatUnknownSession(t *testing.T) {
	h, _ := newTestHub(t)
	if err := h.Heartbeat("nope", os.Getpid()); err == nil {
		t.Fatal("expected error for unknown session")
	}
}

// ---- 心跳守护（OPT-20260806-003） ----

func TestHeartbeatDaemonRefreshesAndStops(t *testing.T) {
	h, _ := newTestHub(t)
	pid := alivePID(t)
	sid, _ := h.Register(Session{Kind: "interactive", PID: pid})
	staleSession(h, sid, pid)

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- h.HeartbeatDaemon(sid, pid, 100*time.Millisecond, stop) }()

	// 守护无事件驱动也持续刷新（严格租约核心：长思考期间心跳不过期）
	deadline := time.Now().Add(2 * time.Second)
	for {
		s, _ := h.get(sid)
		ts, _ := time.Parse(time.RFC3339, s.HeartbeatAt)
		if time.Since(ts) < time.Second {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("daemon did not refresh heartbeat: %s", s.HeartbeatAt)
		}
		time.Sleep(50 * time.Millisecond)
	}

	close(stop)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("daemon exit err: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("daemon did not stop after stop channel close")
	}
}

func TestHeartbeatDaemonExitsOnDeadPid(t *testing.T) {
	h, _ := newTestHub(t)
	pid := deadPID(t)
	sid, _ := h.Register(Session{Kind: "interactive", PID: pid})
	if err := h.HeartbeatDaemon(sid, pid, 50*time.Millisecond, nil); err == nil {
		t.Fatal("expected error for dead pid")
	}
}

func TestHeartbeatDaemonExitsAfterMonitoredPidDies(t *testing.T) {
	h, _ := newTestHub(t)
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	pid := cmd.Process.Pid
	sid, _ := h.Register(Session{Kind: "interactive", PID: pid})

	done := make(chan error, 1)
	go func() { done <- h.HeartbeatDaemon(sid, pid, 100*time.Millisecond, nil) }()

	time.Sleep(250 * time.Millisecond) // 守护已开始周期刷新
	_ = cmd.Process.Kill()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil exit after pid death, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not exit after monitored pid died")
	}
}

// ---- 死锁 ----

func TestDeadlockDetection(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	b, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	_, _, _ = h.Acquire("a", a, 0)
	_, _, _ = h.Acquire("b", b, 0)
	// a 等 b 持有; b 等 a 持有 → 环
	_ = h.SetHoldingWait(a, "b")
	_ = h.SetHoldingWait(b, "a")
	cycle, ok := h.DetectDeadlock(a)
	if !ok {
		t.Fatal("expected deadlock detected")
	}
	if len(cycle) != 2 {
		t.Fatalf("cycle = %v", cycle)
	}
}

func TestNoDeadlock(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	_, _, _ = h.Acquire("a", a, 0)
	_, ok := h.DetectDeadlock(a)
	if ok {
		t.Fatal("no cycle should be deadlock-free")
	}
}

// ---- 字典序加锁 ----

func TestMultiLockOrdering(t *testing.T) {
	h, _ := newTestHub(t)
	a, _ := h.Register(Session{Kind: "headless", PID: alivePID(t)})
	results, err := h.AcquireMany([]string{"zz", "aa", "mm"}, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[0].Repo != "aa" || results[1].Repo != "mm" || results[2].Repo != "zz" {
		t.Fatalf("wrong order: %+v", results)
	}
	// 全部 Acquired
	for _, r := range results {
		if r.Result != Acquired {
			t.Fatalf("expected all acquired: %+v", r)
		}
	}
}

// ---- Shadow Edit ----

func TestShadowBeginApplyUnchanged(t *testing.T) {
	h, root := newTestHub(t)
	sid, _ := h.Register(Session{Kind: "interactive", PID: alivePID(t)})
	// 构造目标文件（repo 内）
	repoDir := filepath.Join(root, "db")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(repoDir, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := "package main\nfunc main() {}\n"
	if err := os.WriteFile(target, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}

	shadowPaths, err := h.ShadowBegin(sid, "db", []string{"src/main.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(shadowPaths) != 1 {
		t.Fatalf("shadowPaths = %v", shadowPaths)
	}
	// 编辑 shadow 副本
	if err := os.WriteFile(shadowPaths[0], []byte(orig+"// edited in shadow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := h.ShadowApply(sid, "db", nil, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), "edited in shadow") {
		t.Fatalf("shadow apply did not copy back:\n%s", got)
	}
	// manifest 清理
	if _, err := os.Stat(h.shadowManifest(sid)); !os.IsNotExist(err) {
		t.Fatal("manifest should be cleaned after apply")
	}
}

func TestShadowApplyConflict(t *testing.T) {
	h, root := newTestHub(t)
	sid, _ := h.Register(Session{Kind: "interactive", PID: alivePID(t)})
	repoDir := filepath.Join(root, "db")
	target := filepath.Join(repoDir, "a.txt")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = h.ShadowBegin(sid, "db", []string{"a.txt"})
	// 他人改了原文件
	if err := os.WriteFile(target, []byte("v2-concurrent"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := h.ShadowApply(sid, "db", nil, false)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	ce, ok := err.(*ConflictError)
	if !ok || len(ce.Files) != 1 {
		t.Fatalf("expected ConflictError with 1 file, got %#v", err)
	}
	// 原文件未被覆盖
	got, _ := os.ReadFile(target)
	if string(got) != "v2-concurrent" {
		t.Fatal("original must remain untouched on conflict")
	}
	// force 覆盖
	if err := h.ShadowApply(sid, "db", nil, true); err != nil {
		t.Fatal(err)
	}
}

func TestShadowAbort(t *testing.T) {
	h, _ := newTestHub(t)
	sid, _ := h.Register(Session{Kind: "interactive", PID: alivePID(t)})
	if _, err := h.ShadowBegin(sid, "db", []string{"x.go"}); err != nil {
		t.Fatal(err)
	}
	if err := h.ShadowAbort(sid); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(h.shadowDir(sid)); !os.IsNotExist(err) {
		t.Fatal("shadow dir should be removed on abort")
	}
}

// ---- 安全：repo 名净化 ----

func TestRepoSanitize(t *testing.T) {
	h, _ := newTestHub(t)
	for _, bad := range []string{"../evil", "a/b/../../c", "", "/abs/path", "a b"} {
		if _, _, err := h.Acquire(bad, "s1", 0); err == nil {
			t.Fatalf("repo %q should be rejected", bad)
		}
	}
	if got, err := sanitizeRepo("taskAuth/sub"); err != nil || got != "taskAuth/sub" {
		t.Fatalf("sanitize: %q %v", got, err)
	}
}

// ---- 审计 ----

func TestAuditLog(t *testing.T) {
	h, _ := newTestHub(t)
	if err := h.Audit("s1", "acquire", "db ok"); err != nil {
		t.Fatal(err)
	}
	// 今日审计文件存在且包含记录
	logs, err := filepath.Glob(filepath.Join(h.dir, "audit", "*.log"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("audit log missing: %v %v", logs, err)
	}
	b, _ := os.ReadFile(logs[0])
	if !strings.Contains(string(b), "acquire") {
		t.Fatalf("audit content: %s", b)
	}
}

// ---- 暂停/恢复 ----

func procState(pid int) string {
	b, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(pid), "stat"))
	if err != nil {
		return ""
	}
	s := string(b)
	if i := strings.LastIndexByte(s, ')'); i >= 0 {
		tail := strings.TrimSpace(s[i+1:])
		if len(tail) > 0 {
			return string(tail[0])
		}
	}
	return ""
}

func waitState(t *testing.T, pid int, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if procState(pid) == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("proc %d state=%q want %q", pid, procState(pid), want)
}

func TestPauseResume(t *testing.T) {
	// 子进程独立进程组，避免信号波及测试进程自身
	cmd := exec.Command("sh", "-c", "while true; do sleep 1; done")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	pid := cmd.Process.Pid

	s := &Session{SessionID: "test", PID: pid, Kind: "headless"}
	if err := PauseSession(s); err != nil {
		t.Fatal(err)
	}
	waitState(t, pid, "T", 2*time.Second) // T = stopped

	if err := ResumeSession(s); err != nil {
		t.Fatal(err)
	}
	waitState(t, pid, "S", 2*time.Second) // S = sleeping/running
}

func TestProcessHasAncestor(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	if !ProcessHasAncestor(cmd.Process.Pid, os.Getpid()) {
		t.Fatal("child should have test process as ancestor")
	}
	if ProcessHasAncestor(os.Getpid(), 999999) {
		t.Fatal("unrelated pid must not match")
	}
}

// ---- 校验和辅助 ----

func TestSHA256Helper(t *testing.T) {
	b := sha256.Sum256([]byte("hello"))
	if hex.EncodeToString(b[:])[:16] != "2cf24dba5fb0a30e" {
		t.Fatal("sha256 mismatch")
	}
}
