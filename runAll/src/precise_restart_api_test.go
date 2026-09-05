package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- API 层 ----------

func testPreciseRestartRunner(t *testing.T, cfg *Config) (*Runner, *StatusStore) {
	t.Helper()
	store := NewStatusStore()
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	stubNoPortListenersForTest(runner)
	return runner, store
}

// OPT-20260807-052：precise-restart 与 start/stop/build/restart-all 统一 bulk 互斥。
// precise-restart 进行中时其余 bulk 入口一律 409；反之 start-all 等占锁时
// precise-restart 拒绝启动。
//
// OPT-20260810-042：bulk 起止会写/删 .runall/bulk_op.lock（供 watchdog 离线感知），
// 测试注入临时登记文件路径隔离，避免触碰真实仓库的运行时锁。
func TestBulkOpMutualExclusion_PreciseRestartBlocksOthers(t *testing.T) {
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(t.TempDir(), "reg.txt"))
	runner, _ := testPreciseRestartRunner(t, &Config{})

	if !runner.TryBeginPreciseRestart("pr-1") {
		t.Fatal("first TryBeginPreciseRestart should succeed")
	}
	defer runner.endPreciseRestart()

	if !runner.IsBulkActive() {
		t.Fatal("IsBulkActive should be true while precise-restart active")
	}
	if got := runner.ActiveBulkOp(); got != "precise-restart" {
		t.Fatalf("ActiveBulkOp = %q, want precise-restart", got)
	}
	for _, op := range []string{"start-all", "stop-all", "build-all", "restart-all"} {
		if runner.TryBeginBulk(op) {
			t.Fatalf("TryBeginBulk(%q) should fail while precise-restart active", op)
		}
	}
	if runner.TryBeginPreciseRestart("pr-2") {
		t.Fatal("second TryBeginPreciseRestart should fail")
	}

	runner.endPreciseRestart()
	if runner.IsBulkActive() {
		t.Fatal("IsBulkActive should be false after endPreciseRestart")
	}
	if !runner.TryBeginBulk("start-all") {
		t.Fatal("TryBeginBulk(start-all) should succeed after precise-restart released")
	}
	runner.EndBulk("start-all")
}

func TestBulkOpMutualExclusion_BulkOpBlocksPreciseRestart(t *testing.T) {
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(t.TempDir(), "reg.txt"))
	runner, _ := testPreciseRestartRunner(t, &Config{})

	if !runner.TryBeginBulk("stop-all") {
		t.Fatal("TryBeginBulk(stop-all) should succeed")
	}
	defer runner.EndBulk("stop-all")

	if runner.TryBeginPreciseRestart("pr-1") {
		t.Fatal("TryBeginPreciseRestart should fail while stop-all holds bulk lock")
	}
	if !runner.IsBulkActive() {
		t.Fatal("IsBulkActive should be true")
	}
	if got := runner.ActiveBulkOp(); got != "stop-all" {
		t.Fatalf("ActiveBulkOp = %q, want stop-all", got)
	}
}

// 跨 entry 的互斥同时成立：restart-all 占锁时 build-all 入口（TryBeginBuildAllRun）拒绝。
func TestBulkOpMutualExclusion_RestartAllBlocksBuildAll(t *testing.T) {
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", filepath.Join(t.TempDir(), "reg.txt"))
	runner, _ := testPreciseRestartRunner(t, &Config{})

	if !runner.TryBeginRestartAll() {
		t.Fatal("TryBeginRestartAll should succeed")
	}
	defer runner.endRestartAll()

	if _, ok := runner.TryBeginBuildAllRun(); ok {
		t.Fatal("TryBeginBuildAllRun should fail while restart-all active")
	}
	if runner.TryBeginBulk("start-all") {
		t.Fatal("TryBeginBulk(start-all) should fail while restart-all active")
	}
}

func TestAPIPreciseRestartRegistrations_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	runner, _ := testPreciseRestartRunner(t, &Config{})
	mux := http.NewServeMux()
	registerPreciseRestartHandlers(mux, runner)

	req := httptest.NewRequest(http.MethodGet, "/api/precise-restart/registrations", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Services []string `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Services) != 0 {
		t.Fatalf("expected empty services, got %v", body.Services)
	}
}

func TestAPIPreciseRestartRegister_ValidAndUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	cfg := &Config{Groups: []Group{{Services: []Service{
		{Name: "task-auth", WorkingDir: "taskAuth"},
	}}}}
	runner, _ := testPreciseRestartRunner(t, cfg)
	mux := http.NewServeMux()
	registerPreciseRestartHandlers(mux, runner)

	// 未知服务 → 400
	req := httptest.NewRequest(http.MethodPost, "/api/precise-restart/register",
		strings.NewReader(`{"services":["nope"]}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown service: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "nope") {
		t.Fatalf("error should name unknown service, got %s", rec.Body.String())
	}

	// 合法登记（含目录名别名）→ 200 + merged
	req2 := httptest.NewRequest(http.MethodPost, "/api/precise-restart/register",
		strings.NewReader(`{"services":["task-auth","taskAuth"]}`))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("valid register: status = %d, body=%s", rec2.Code, rec2.Body.String())
	}
	var body struct {
		Services []string `json:"services"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if strings.Join(body.Services, ",") != "task-auth" {
		t.Fatalf("merged = %v, want [task-auth] (dedup)", body.Services)
	}
}

func TestAPIPreciseRestartAction_NoRegistrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	runner, _ := testPreciseRestartRunner(t, &Config{})
	mux := http.NewServeMux()
	registerPreciseRestartHandlers(mux, runner)

	req := httptest.NewRequest(http.MethodPost, "/api/precise-restart", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 empty no-op, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Status   string `json:"status"`
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "empty" {
		t.Fatalf("status = %q, want empty; body=%s", body.Status, rec.Body.String())
	}
	if !strings.Contains(body.FilePath, filepath.Base(path)) {
		t.Fatalf("file_path = %q, want to name %s", body.FilePath, path)
	}
	if runner.IsPreciseRestartActive() {
		t.Fatal("empty registry must not start a precise-restart bulk op")
	}
}

func TestAPIPreciseRestartRegistrations_FillFromScan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	called := 0
	old := fillPreciseRestartFromScanFn
	fillPreciseRestartFromScanFn = func(string) {
		called++
		if err := writeRegisteredServices(path, []string{"task-auth"}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { fillPreciseRestartFromScanFn = old })

	cfg := &Config{Groups: []Group{{Services: []Service{{Name: "task-auth", WorkingDir: "taskAuth"}}}}}
	runner, _ := testPreciseRestartRunner(t, cfg)
	mux := http.NewServeMux()
	registerPreciseRestartHandlers(mux, runner)

	req := httptest.NewRequest(http.MethodGet, "/api/precise-restart/registrations", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if called != 0 {
		t.Fatalf("poller GET must not scan, called=%d", called)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/precise-restart/registrations?fill_from_scan=1", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec2.Code, rec2.Body.String())
	}
	if called != 1 {
		t.Fatalf("fill_from_scan GET must scan once, called=%d", called)
	}
	var body struct {
		Services []string `json:"services"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if strings.Join(body.Services, ",") != "task-auth" {
		t.Fatalf("after scan services = %v, want [task-auth]", body.Services)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/precise-restart/registrations?fill_from_scan=1", nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if called != 1 {
		t.Fatalf("non-empty registry must not scan again, called=%d", called)
	}
}

func TestAPIPreciseRestartAction_ConflictWhenActive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"task-auth"}); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{
		{Name: "task-auth", WorkingDir: "taskAuth"},
	}}}}
	runner, _ := testPreciseRestartRunner(t, cfg)
	mux := http.NewServeMux()
	registerPreciseRestartHandlers(mux, runner)

	// 先手动占位激活（模拟一次进行中的精准重启）
	runner.TryBeginPreciseRestart("precise-restart-test")

	req := httptest.NewRequest(http.MethodPost, "/api/precise-restart", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body=%s", rec.Code, rec.Body.String())
	}
	runner.endPreciseRestart()
}
