package main

// OPT-20260902-026：精准编译重启后服务若带病存活（进程在、健康检查失败）——
// 此前 runAll 只对已退出（zombie）进程自动重启，带病存活进程要等外部 5 分钟
// cron watchdog。此测试锁定「进程存活但健康失败也会触发自动重启、且受 per-session
// 上限约束」的行为。

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func aliveServiceCfg(name, healthURL string) *Config {
	return &Config{Groups: []Group{{Services: []Service{{
		Name:    name,
		Command: "sleep 300",
		HealthCheck: HealthCheck{
			URL:                healthURL,
			Timeout:            1,
			Retries:            2,
			CheckInterval:      1,
			UnhealthyThreshold: 2,
			Backoff:            Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
		},
	}}}}}
}

func TestFailedServiceAutoRestart_AliveButUnhealthyProcessIsRestarted(t *testing.T) {
	// 健康端点保持可用：预置 store=Failed + 仍存活的托管进程后，自动重启应能
	// 完整跑通「杀旧 → 起新 → 健康」并回到 Healthy。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	const name = "svc-alive-unhealthy"
	cfg := aliveServiceCfg(name, srv.URL)
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{name})
	store.Update(name, StatusFailed, "precondition: health check failed while process still alive")

	// 伪造一个仍存活的托管进程（复现「磁盘二进制已换、旧 pid 仍占端口」场景）。
	alive := exec.Command("sleep", "300")
	if err := alive.Start(); err != nil {
		t.Fatalf("start fake tracked process: %v", err)
	}
	t.Cleanup(func() { _ = alive.Process.Kill() })
	runner.mu.Lock()
	runner.processes[name] = alive
	runner.mu.Unlock()
	if exited, _ := processExitInfo(alive.Process); exited {
		t.Fatal("fake tracked process exited unexpectedly")
	}

	var callsMu sync.Mutex
	calls := 0
	prevHook := restartServiceTestHook
	restartServiceTestHook = func(n string) {
		if n == name {
			callsMu.Lock()
			calls++
			callsMu.Unlock()
		}
	}
	t.Cleanup(func() { restartServiceTestHook = prevHook })

	runner.tryAutoRestartFailedService(cfg.Groups[0].Services[0])

	waitFor := func(desc string, d time.Duration, cond func() bool) bool {
		deadline := time.Now().Add(d)
		for time.Now().Before(deadline) {
			if cond() {
				return true
			}
			time.Sleep(50 * time.Millisecond)
		}
		return cond()
	}

	called := waitFor("restartService triggered", 5*time.Second, func() bool {
		callsMu.Lock()
		defer callsMu.Unlock()
		return calls >= 1
	})
	if !called {
		t.Fatal("restartService was not triggered for an alive-but-unhealthy tracked process")
	}

	// 端点恢复健康后，重启应成功并把服务带回 Healthy（证明是完整自动重启而非只标记失败）。
	recovered := waitFor("service healthy after auto-restart", 8*time.Second, func() bool {
		st := store.Get(name)
		return st != nil && st.Status == StatusHealthy
	})
	if !recovered {
		t.Fatalf("service did not recover after alive-but-unhealthy auto-restart; status=%+v", store.Get(name))
	}

	// 清理：结束重启后新起的进程及其 monitor。
	_ = runner.stopProcess(name)
}

func TestFailedServiceAutoRestart_RespectsSessionCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	const name = "svc-capped"
	cfg := aliveServiceCfg(name, srv.URL)
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{name})
	store.Update(name, StatusFailed, "precondition")

	alive := exec.Command("sleep", "300")
	if err := alive.Start(); err != nil {
		t.Fatalf("start fake tracked process: %v", err)
	}
	t.Cleanup(func() { _ = alive.Process.Kill() })
	runner.mu.Lock()
	runner.processes[name] = alive
	runner.mu.Unlock()

	runner.zombieRestartMu.Lock()
	runner.zombieRestartCount[name] = maxZombieRestarts
	runner.zombieRestartMu.Unlock()

	calls := 0
	prevHook := restartServiceTestHook
	restartServiceTestHook = func(n string) {
		if n == name {
			calls++
		}
	}
	t.Cleanup(func() { restartServiceTestHook = prevHook })

	runner.tryAutoRestartFailedService(cfg.Groups[0].Services[0])
	if calls != 0 {
		t.Fatalf("auto-restart attempted despite reaching session cap (calls=%d)", calls)
	}
}

func TestFailedServiceAutoRestart_NoTrackedProcessSkips(t *testing.T) {
	const name = "svc-external"
	cfg := aliveServiceCfg(name, "http://127.0.0.1:1/health")
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{name})
	store.Update(name, StatusFailed, "precondition")

	calls := 0
	prevHook := restartServiceTestHook
	restartServiceTestHook = func(n string) {
		if n == name {
			calls++
		}
	}
	t.Cleanup(func() { restartServiceTestHook = prevHook })

	runner.tryAutoRestartFailedService(cfg.Groups[0].Services[0])
	if calls != 0 {
		t.Fatalf("external service without a runAll process handle must not be auto-restarted (calls=%d)", calls)
	}
}
