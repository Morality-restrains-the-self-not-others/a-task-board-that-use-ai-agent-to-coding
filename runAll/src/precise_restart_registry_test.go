package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- OPT-20260807-024/025/026 UI 支撑 ----------

// OPT-20260807-026：过期登记重新登记时刷新时间戳并重置为 pending，按钮恢复可用。
func TestAppendRegisteredServices_ExpiredEntryRefreshed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	old := time.Now().Add(-48 * time.Hour).Unix()
	// 写一条 48h 前登记的条目（已超过 24h TTL）。
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "task-auth", State: RegistrationStatePending, RegisteredAt: old},
	}); err != nil {
		t.Fatal(err)
	}
	merged, err := appendRegisteredServices(path, []string{"task-auth"})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0] != "task-auth" {
		t.Fatalf("merged = %v, want [task-auth]", merged)
	}
	entries, _ := readRegistrationEntries(path)
	if len(entries) != 1 {
		t.Fatalf("entries = %v, want 1", entries)
	}
	if entries[0].State != RegistrationStatePending {
		t.Fatalf("state = %q, want pending (expired re-register resets state)", entries[0].State)
	}
	if entries[0].RegisteredAt <= old {
		t.Fatalf("registered_at not refreshed: old=%d new=%d", old, entries[0].RegisteredAt)
	}
	if registrationExpired(entries[0], time.Now()) {
		t.Fatal("re-registered entry should not be expired")
	}
}

// registrationExpired 对无时间戳条目仍返回 expired（防御性；读取路径会先自动补戳）。
func TestRegistrationExpired_LegacyNoTimestampRowIsExpired(t *testing.T) {
	if !registrationExpired(RegistrationEntry{Name: "task-auth", State: RegistrationStatePending}, time.Now()) {
		t.Fatal("unparsed legacy row (RegisteredAt=0) should be treated as expired")
	}
}

// 旧格式纯服务名文件在 readRegistrationEntries 时自动补戳并写回三列格式。
func TestReadRegistrationEntries_MigratesLegacyNameOnlyRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if err := os.WriteFile(path, []byte("taskFE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := time.Now().Unix()
	entries, err := readRegistrationEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "taskFE" {
		t.Fatalf("entries = %+v, want taskFE", entries)
	}
	if entries[0].RegisteredAt < before {
		t.Fatalf("registered_at = %d, want >= %d", entries[0].RegisteredAt, before)
	}
	if registrationExpired(entries[0], time.Now()) {
		t.Fatal("migrated legacy row should not be expired")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "taskFE\tpending\t") {
		t.Fatalf("file not migrated to tab format: %q", string(raw))
	}
	// 二次读取不应再次改写时间戳
	entries2, err := readRegistrationEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if entries2[0].RegisteredAt != entries[0].RegisteredAt {
		t.Fatalf("second read changed timestamp: %d -> %d", entries[0].RegisteredAt, entries2[0].RegisteredAt)
	}
}

func TestAppendRegisteredServices_LegacyRowRefreshedOnReregister(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "task-auth", State: RegistrationStatePending}, // RegisteredAt=0 → 旧格式
	}); err != nil {
		t.Fatal(err)
	}
	merged, err := appendRegisteredServices(path, []string{"task-auth"})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0] != "task-auth" {
		t.Fatalf("merged = %v, want [task-auth]", merged)
	}
	entries, _ := readRegistrationEntries(path)
	if len(entries) != 1 {
		t.Fatalf("entries = %v, want 1", entries)
	}
	if entries[0].State != RegistrationStatePending || entries[0].RegisteredAt <= 0 {
		t.Fatalf("legacy row should be refreshed to pending+timestamp, got %+v", entries[0])
	}
}

// OPT-20260807-026：未过期的重复登记保留原状态与时间戳（含失败待重试项）。
func TestAppendRegisteredServices_NonExpiredDuplicateKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	ts := time.Now().Add(-1 * time.Hour).Unix()
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "task-bill", State: RegistrationStateFailed, RegisteredAt: ts},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := appendRegisteredServices(path, []string{"task-bill"}); err != nil {
		t.Fatal(err)
	}
	entries, _ := readRegistrationEntries(path)
	if len(entries) != 1 {
		t.Fatalf("entries = %v, want 1", entries)
	}
	if entries[0].State != RegistrationStateFailed || entries[0].RegisteredAt != ts {
		t.Fatalf("duplicate should be kept untouched, got %+v", entries[0])
	}
}

// OPT-20260807-024/025/026：registrations 接口返回结构化 entries，
// 区分 failed/pending、标注 buildable（无编译直接重启）与 expired（TTL 置灰）。
func TestAPIPreciseRestartRegistrations_StructuredEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	// 一条 failed（新近失败，未过期）+ 一条 pending 且超 24h（过期）+
	// 一条 pending 且不可编译（无 build_command）+ 一条未知服务（不可解析）。
	stale := time.Now().Add(-48 * time.Hour).Unix()
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "svc-a", State: RegistrationStateFailed, RegisteredAt: time.Now().Unix()},
		{Name: "svc-b", State: RegistrationStatePending, RegisteredAt: stale},
		{Name: "svc-c", State: RegistrationStatePending, RegisteredAt: time.Now().Unix()},
		{Name: "ghost-svc", State: RegistrationStatePending, RegisteredAt: time.Now().Unix()},
	}); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{
		{Name: "svc-a", BuildCommand: "go build -o app ."},
		{Name: "svc-b", BuildCommand: "go build -o app ."},
		{Name: "svc-c"}, // 无 build_command，也不存在 build.sh/go.mod 工作目录 → 不可编译
	}}}}
	runner, _ := testPreciseRestartRunner(t, cfg)
	mux := http.NewServeMux()
	registerPreciseRestartHandlers(mux, runner)

	req := httptest.NewRequest(http.MethodGet, "/api/precise-restart/registrations", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []struct {
			Name         string `json:"name"`
			State        string `json:"state"`
			RegisteredAt int64  `json:"registered_at"`
			Expired      bool   `json:"expired"`
			Resolvable   bool   `json:"resolvable"`
			Buildable    bool   `json:"buildable"`
		} `json:"entries"`
		Services []string `json:"services"`
		TTLHours int      `json:"ttl_hours"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(body.Entries))
	}
	if len(body.Services) != 4 || body.TTLHours != 24 {
		t.Fatalf("services=%v ttl_hours=%d, want 4 entries & 24h ttl", body.Services, body.TTLHours)
	}
	byName := map[string]struct {
		state      string
		expired    bool
		resolvable bool
		buildable  bool
	}{}
	for _, e := range body.Entries {
		byName[e.Name] = struct {
			state      string
			expired    bool
			resolvable bool
			buildable  bool
		}{e.State, e.Expired, e.Resolvable, e.Buildable}
	}
	if got := byName["svc-a"]; got.state != "failed" || got.expired || !got.resolvable || !got.buildable {
		t.Fatalf("svc-a = %+v, want failed/not-expired/resolvable/buildable", got)
	}
	if got := byName["svc-b"]; got.state != "pending" || !got.expired {
		t.Fatalf("svc-b = %+v, want pending/expired", got)
	}
	if got := byName["svc-c"]; got.state != "pending" || got.expired || !got.resolvable || got.buildable {
		t.Fatalf("svc-c = %+v, want pending/not-expired/resolvable/not-buildable", got)
	}
	if got := byName["ghost-svc"]; got.resolvable {
		t.Fatalf("ghost-svc = %+v, want unresolvable", got)
	}
}
