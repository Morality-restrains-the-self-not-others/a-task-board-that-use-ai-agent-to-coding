package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// hotReplaceScript 生成 start/stop/reload 三标记脚本，供热替换重启测试断言哪些
// 生命周期命令被调用（OPT-20260820-004）。
func hotReplaceScript(dir string) string {
	startMarker := filepath.Join(dir, "started.marker")
	stopMarker := filepath.Join(dir, "stopped.marker")
	reloadMarker := filepath.Join(dir, "reloaded.marker")
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  start) touch %q; exec sleep 5 ;;
  stop) touch %q ;;
  reload) touch %q ;;
esac
`, startMarker, stopMarker, reloadMarker)
}

// TestRunner_RestartService_SkipStopOnRestart_ReusesLiveContainer 验证热替换重启：
// 服务 Healthy + skip_stop_on_restart + detach 时，restartService 跳过 stop 阶段
// （不跑 stop_command、不做端口兜底终止）、复用在线容器（不跑 start_command），
// 并在最后执行 restart_reload_command（如 nginx -s reload）。
func TestRunner_RestartService_SkipStopOnRestart_ReusesLiveContainer(t *testing.T) {
	dir := t.TempDir()
	startMarker := filepath.Join(dir, "started.marker")
	stopMarker := filepath.Join(dir, "stopped.marker")
	reloadMarker := filepath.Join(dir, "reloaded.marker")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	runSh := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(runSh, []byte(hotReplaceScript(dir)), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:                 "svc",
				StartCommand:         "bash run.sh start",
				StopCommand:          "bash run.sh stop",
				RestartReloadCommand: "bash run.sh reload",
				SkipStopOnRestart:    true,
				LaunchMode:           "detach",
				WorkingDir:           dir,
				HealthCheck: HealthCheck{
					URL:           srv.URL,
					Timeout:       5,
					Retries:       5,
					CheckInterval: 1,
					Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	// 模拟在线容器：健康端口上有活跃监听（forceFreshStart=false → 跳过启动复用）。
	runner.listenerPIDsFn = func(string) ([]int, error) { return []int{12345}, nil }

	store.Update("svc", StatusHealthy, "")
	if err := runner.RestartService(context.Background(), "svc"); err != nil {
		t.Fatalf("RestartService: %v", err)
	}
	if _, err := os.Stat(stopMarker); err == nil {
		t.Fatal("hot-replace restart must NOT run stop_command")
	}
	if _, err := os.Stat(startMarker); err == nil {
		t.Fatal("hot-replace restart must NOT run start_command (reuses live container)")
	}
	if _, err := os.Stat(reloadMarker); err != nil {
		t.Fatalf("expected restart_reload_command after hot replace: %v", err)
	}
	got := store.Get("svc")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", got)
	}
}

// TestRunner_RestartService_SkipStopOnRestart_StartsWhenContainerDown 验证热替换重启
// 在容器已不在线（无端口监听）时仍能恢复：跳过 stop、跑 start_command 拉起容器。
func TestRunner_RestartService_SkipStopOnRestart_StartsWhenContainerDown(t *testing.T) {
	dir := t.TempDir()
	startMarker := filepath.Join(dir, "started.marker")
	stopMarker := filepath.Join(dir, "stopped.marker")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	runSh := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(runSh, []byte(hotReplaceScript(dir)), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:              "svc",
				StartCommand:      "bash run.sh start",
				StopCommand:       "bash run.sh stop",
				SkipStopOnRestart: true,
				LaunchMode:        "detach",
				WorkingDir:        dir,
				HealthCheck: HealthCheck{
					URL:           srv.URL,
					Timeout:       5,
					Retries:       5,
					CheckInterval: 1,
					Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	// 容器不在线：健康端口无监听 → startAndCheck 应执行 start_command 拉起。
	runner.listenerPIDsFn = func(string) ([]int, error) { return nil, nil }

	store.Update("svc", StatusHealthy, "")
	if err := runner.RestartService(context.Background(), "svc"); err != nil {
		t.Fatalf("RestartService: %v", err)
	}
	if _, err := os.Stat(stopMarker); err == nil {
		t.Fatal("hot-replace restart must NOT run stop_command")
	}
	if _, err := os.Stat(startMarker); err != nil {
		t.Fatalf("expected start_command to recover downed container: %v", err)
	}
	got := store.Get("svc")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", got)
	}
	t.Cleanup(func() {
		_ = runner.StopService(context.Background(), "svc")
	})
}

// TestRunner_RestartService_SkipStopOnRestart_NotEligibleWhenNotHealthy 验证守卫：
// skip_stop_on_restart 仅当服务当前 Healthy 时启用；Failed/Stopped 服务仍走
// 完整停 → 启动（普通重启不编译；精准编译见 compile-then-swap 测例）。
func TestRunner_RestartService_SkipStopOnRestart_NotEligibleWhenNotHealthy(t *testing.T) {
	dir := t.TempDir()
	startMarker := filepath.Join(dir, "started.marker")
	stopMarker := filepath.Join(dir, "stopped.marker")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	runSh := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(runSh, []byte(hotReplaceScript(dir)), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:              "svc",
				StartCommand:      "bash run.sh start",
				StopCommand:       "bash run.sh stop",
				SkipStopOnRestart: true,
				LaunchMode:        "detach",
				WorkingDir:        dir,
				HealthCheck: HealthCheck{
					URL:           srv.URL,
					Timeout:       5,
					Retries:       5,
					CheckInterval: 1,
					Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) { return nil, nil }
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })

	store.Update("svc", StatusStopped, "")
	if err := runner.RestartService(context.Background(), "svc"); err != nil {
		t.Fatalf("RestartService: %v", err)
	}
	if _, err := os.Stat(stopMarker); err != nil {
		t.Fatalf("non-healthy service must still run stop_command: %v", err)
	}
	if _, err := os.Stat(startMarker); err != nil {
		t.Fatalf("expected start_command after stop: %v", err)
	}
	got := store.Get("svc")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", got)
	}
	t.Cleanup(func() {
		_ = runner.StopService(context.Background(), "svc")
	})
}

func TestRunner_RestartService_StopThenStart(t *testing.T) {
	dir := t.TempDir()
	stopMarker := filepath.Join(dir, "stopped.marker")
	startMarker := filepath.Join(dir, "started.marker")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  start) touch %q; exec sleep 5 ;;
  stop) touch %q ;;
esac
`, startMarker, stopMarker)
	runSh := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(runSh, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:         "svc",
				StartCommand: "bash run.sh start",
				StopCommand:  "bash run.sh stop",
				WorkingDir:   dir,
				HealthCheck: HealthCheck{
					URL:           srv.URL,
					Timeout:       5,
					Retries:       5,
					CheckInterval: 1,
					Backoff:       Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.2},
				},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) { return nil, nil }
	// 健康端点为进程内 httptest（测试进程自身监听），kill-first 重启的端口释放
	// 等待必然 30s 超时；缩短窗口加速测试。
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })

	store.Update("svc", StatusStopped, "")
	if err := runner.RestartService(context.Background(), "svc"); err != nil {
		t.Fatalf("RestartService: %v", err)
	}
	if _, err := os.Stat(stopMarker); err != nil {
		t.Fatalf("expected stop_command before start: %v", err)
	}
	if _, err := os.Stat(startMarker); err != nil {
		t.Fatalf("expected start_command after stop: %v", err)
	}
	got := store.Get("svc")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("status = %#v, want healthy", got)
	}
	t.Cleanup(func() {
		_ = runner.StopService(context.Background(), "svc")
	})
}
