package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------- 引擎层 ----------

// 未知名登记：整体拒绝该服务（不执行任何重启），失败项保留在文件中。
func TestPreciseRestart_UnknownServiceRetainedInFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"ghost-svc"}); err != nil {
		t.Fatal(err)
	}
	runner, _ := testPreciseRestartRunner(t, &Config{})

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 1 || keep[0] != "ghost-svc" {
		t.Fatalf("keep = %v, want [ghost-svc]", keep)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 1 || after[0] != "ghost-svc" {
		t.Fatalf("file after = %v, want [ghost-svc]", after)
	}
}

// 空登记：报错且不触碰文件。
func TestPreciseRestart_NoRegistrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	runner, _ := testPreciseRestartRunner(t, &Config{})
	if _, err := runner.PreciseRestart(context.Background(), "test-session"); err == nil {
		t.Fatal("expected error for empty registrations")
	}
}

// 成功路径：登记的两个服务被重启（依赖序：a 先于 b），文件被清空。
func TestPreciseRestart_SuccessClearsFile(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	cfg := &Config{
		Groups: []Group{{Services: []Service{
			{
				Name:        "svc-precise-a",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			},
			{
				Name:        "svc-precise-b",
				DependsOn:   []string{"svc-precise-a"},
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			},
		}}},
	}
	runner, store := testPreciseRestartRunner(t, cfg)
	// 健康检查端点为进程内 httptest server（测试二进制自身 PID 监听），
	// kill-first 重启的端口释放等待会撞上 30s×2 超时；缩短等待窗口加速测试。
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	store.Init([]string{"svc-precise-a", "svc-precise-b"})
	store.Update("svc-precise-a", StatusStopped, "")
	store.Update("svc-precise-b", StatusStopped, "")

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	// 逆依赖序登记（b 在 a 之前），验证引擎按依赖深度重新排序（a 先重启）。
	if err := writeRegisteredServices(path, []string{"svc-precise-b", "svc-precise-a"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-test") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 0 {
		t.Fatalf("keep = %v, want empty", keep)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 0 {
		t.Fatalf("file should be cleared after success, got %v", after)
	}

	// 登记的服务都应健康（未登记的服务不会被触碰）。
	deadline := time.Now().Add(10 * time.Second)
	for {
		stA := store.Get("svc-precise-a")
		stB := store.Get("svc-precise-b")
		if stA != nil && stA.Status == StatusHealthy && stB != nil && stB.Status == StatusHealthy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("services not healthy: a=%v b=%v", stA, stB)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// 别名归一化：svc-precise-x 与 svcPreciseX（同服务不同别名）只重启一次、只登记一次。
func TestPreciseRestart_AliasDedup(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	// WorkingDir 兼作别名归一化键（svc-precise-x ↔ svcPreciseX，按首段匹配）与
	// 启动 chdir 目标：kill-first 强制启动路径真实执行 cmd.Dir，须指向存在的目录
	// （相对测试 cwd 的旧写法在 forceFreshStart 下 chdir ENOENT，被旧 skip-start
	// 语义掩盖）。t.Chdir + Mkdir 两全：别名字符串不变，目录真实存在。
	workRoot := t.TempDir()
	t.Chdir(workRoot)
	if err := os.Mkdir("svcPreciseX", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeRegisteredServices(path, []string{"svc-precise-x", "svcPreciseX"}); err != nil {
		t.Fatal(err)
	}
	store := NewStatusStore()
	cfg := &Config{Groups: []Group{{Services: []Service{
		{
			Name:        "svc-precise-x",
			WorkingDir:  "svcPreciseX",
			Command:     "sleep 30",
			HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
		},
	}}}}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	// 与其它 precise-restart 测试一致：健康端点为进程内 httptest，端口监听者
	// 即测试进程自身（kill-first 的端口兜底终止会跳过自身 PID，见 terminateListenersByPort），
	// 故 stub 监听者查询，聚焦别名去重语义。
	stubNoPortListenersForTest(runner)
	// 健康端点为进程内 httptest（测试进程自身监听），端口释放等待必然 30s 超时；
	// 缩短窗口避免拖慢套件。
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	store.Init([]string{"svc-precise-x"})
	store.Update("svc-precise-x", StatusStopped, "")

	// 引擎不会对同一服务执行两次：第二次解析被别名去重跳过（total 应为 1）。
	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 0 {
		t.Fatalf("keep = %v, want empty", keep)
	}
	if st := store.Get("svc-precise-x"); st == nil || st.Status != StatusHealthy {
		t.Fatalf("service status = %v, want healthy", st)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 0 {
		t.Fatalf("file should be cleared after success, got %v", after)
	}
}
