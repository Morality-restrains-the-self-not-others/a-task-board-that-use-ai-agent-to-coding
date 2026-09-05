package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// OPT-20260816-045：start-all / start-group / 单服务 start 入口需从磁盘重载 runAll.yaml，
// 使进程启动后才写入 YAML 的服务可被 UI 拉起（与 PreciseRestart 相同的失败保留内存语义）。

func TestStartGroup_ReloadsDiskOnlyService(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, store := testPreciseRestartRunner(t, &Config{})
	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-group-svc", health.URL)
	runner.SetConfigPath(yamlPath)

	// 内存缺名、磁盘 YAML 有名：StartGroup 必须通过磁盘重载找到该服务并启动
	if err := runner.StartGroup(context.Background(), "g"); err != nil {
		t.Fatalf("StartGroup: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		st := store.Get("disk-group-svc")
		if st != nil && st.Status == StatusHealthy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("disk-group-svc not healthy after StartGroup reload: %+v", st)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestStartService_ReloadsDiskOnlyService(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, store := testPreciseRestartRunner(t, &Config{})
	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-svc-single", health.URL)
	runner.SetConfigPath(yamlPath)

	if err := runner.StartService(context.Background(), "disk-svc-single"); err != nil {
		t.Fatalf("StartService: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		st := store.Get("disk-svc-single")
		if st != nil && st.Status == StatusHealthy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("disk-svc-single not healthy after StartService reload: %+v", st)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// OPT-20260816-046：GET 精准编译重启登记时，对 unresolved 名做一次磁盘重载，
// 使 UI 不再把可热加载的新服务显示成「未知服务」。
func TestPreciseRestartRegistrations_ReloadsUnresolved(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, _ := testPreciseRestartRunner(t, &Config{})
	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-new-svc", health.URL)
	runner.SetConfigPath(yamlPath)

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"disk-new-svc"}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/precise-restart/registrations", nil)
	rec := httptest.NewRecorder()
	handlePreciseRestartRegistrations(rec, req, runner)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET registrations status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Entries []preciseRestartRegistrationView `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(body.Entries))
	}
	if body.Entries[0].Name != "disk-new-svc" {
		t.Fatalf("entry name = %q, want disk-new-svc", body.Entries[0].Name)
	}
	if !body.Entries[0].Resolvable {
		t.Fatal("resolvable should be true after disk reload (in-memory config lacked the service)")
	}
}

func TestStartGroup_ReloadFailureKeepsMemoryAndFailsPlanning(t *testing.T) {
	runner, _ := testPreciseRestartRunner(t, &Config{Groups: []Group{{Services: []Service{
		{Name: "mem-svc", Command: "true"},
	}}}})
	bad := filepath.Join(t.TempDir(), "runAll.yaml")
	if err := os.WriteFile(bad, []byte("{{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner.SetConfigPath(bad)

	// 重载失败必须保留内存配置，且 group 名不存在时 StartGroup 返回明确错误而非 panic
	if err := runner.StartGroup(context.Background(), "g"); err == nil {
		t.Fatal("StartGroup on missing group should error")
	}
	if svc := runner.resolveRegisteredService("mem-svc"); svc == nil {
		t.Fatal("invalid YAML must keep in-memory config")
	}
}
