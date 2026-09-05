package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 成功路径须写入 consumed-at 水位线，供自动扫描跳过「已部署但工作树仍脏」的重登记。
func TestPreciseRestart_SuccessWritesConsumedAtWatermark(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	cfg := &Config{
		Groups: []Group{{Services: []Service{{
			Name:        "svc-wm-a",
			Command:     "sleep 30",
			HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
		}}}},
	}
	runner, store := testPreciseRestartRunner(t, cfg)
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	store.Init([]string{"svc-wm-a"})
	store.Update("svc-wm-a", StatusStopped, "")

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"svc-wm-a"}); err != nil {
		t.Fatal(err)
	}
	before := time.Now().Unix()
	if !runner.TryBeginPreciseRestart("precise-restart-wm") {
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
		t.Fatalf("file should be cleared, got %v", after)
	}
	wmPath := preciseRestartConsumedAtFile(path)
	raw, err := os.ReadFile(wmPath)
	if err != nil {
		t.Fatalf("consumed-at watermark missing: %v", err)
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || ts < before {
		t.Fatalf("consumed-at = %q (parsed %d), want unix >= %d", raw, ts, before)
	}
}

// 运行中并发追加的新登记不得被成功收尾误删。
func TestRewriteRegistrationsAfterRun_PreservesConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "svc-a", State: RegistrationStatePending, RegisteredAt: 100},
		{Name: "svc-b", State: RegistrationStatePending, RegisteredAt: 200}, // 并发追加
	}); err != nil {
		t.Fatal(err)
	}
	original := map[string]bool{"svc-a": true}
	keep, err := rewriteRegistrationsAfterRun(path, original, nil, []string{"svc-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(keep) != 1 || keep[0] != "svc-b" {
		t.Fatalf("keep = %v, want [svc-b]", keep)
	}
	entries, _ := readRegistrationEntries(path)
	if len(entries) != 1 || entries[0].Name != "svc-b" || entries[0].State != RegistrationStatePending {
		t.Fatalf("entries = %+v, want pending svc-b", entries)
	}
}

// 分组编译成功后：仅移除本组已登记且编译成功的服务名；本组失败项、本组外服务与
// 并发追加的新登记一律保留（OPT-20260811-036）。
func TestTrimRegistrationsAfterGroupBuild_RemovesOnlyBuiltGroupServices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if err := writeRegistrationEntries(path, []RegistrationEntry{
		{Name: "svc-in-group-ok", State: RegistrationStatePending, RegisteredAt: 100},
		{Name: "svc-in-group-failed", State: RegistrationStateFailed, RegisteredAt: 110},
		{Name: "svc-other-group", State: RegistrationStatePending, RegisteredAt: 120},
		{Name: "svc-concurrent", State: RegistrationStatePending, RegisteredAt: 130},
	}); err != nil {
		t.Fatal(err)
	}
	if err := trimRegistrationsAfterGroupBuild(path, []string{"svc-in-group-ok"}); err != nil {
		t.Fatal(err)
	}
	entries, err := readRegistrationEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]RegistrationState{}
	for _, e := range entries {
		byName[e.Name] = e.State
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %+v, want 3 kept (only built group service removed)", entries)
	}
	if _, ok := byName["svc-in-group-ok"]; ok {
		t.Fatalf("successfully built group service must be removed, got %+v", entries)
	}
	if byName["svc-in-group-failed"] != RegistrationStateFailed {
		t.Fatalf("failed group service must stay failed, got %+v", entries)
	}
	if byName["svc-other-group"] != RegistrationStatePending {
		t.Fatalf("other-group service must stay pending, got %+v", entries)
	}
	if byName["svc-concurrent"] != RegistrationStatePending {
		t.Fatalf("concurrent registration must stay pending, got %+v", entries)
	}
}

// built 为空时裁剪为 no-op，登记文件原样保留。
func TestTrimRegistrationsAfterGroupBuild_EmptyBuiltNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if err := writeRegisteredServices(path, []string{"svc-a"}); err != nil {
		t.Fatal(err)
	}
	if err := trimRegistrationsAfterGroupBuild(path, nil); err != nil {
		t.Fatal(err)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 1 || after[0] != "svc-a" {
		t.Fatalf("after = %v, want [svc-a] unchanged", after)
	}
}

// 失败项保留为 failed；成功项清除；本批未出现在当前文件中的失败名仍写回。
func TestRewriteRegistrationsAfterRun_RetainsFailed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.txt")
	if err := writeRegisteredServices(path, []string{"svc-ok", "svc-bad"}); err != nil {
		t.Fatal(err)
	}
	original := map[string]bool{"svc-ok": true, "svc-bad": true}
	keep, err := rewriteRegistrationsAfterRun(path, original, []string{"svc-bad"}, []string{"svc-ok"})
	if err != nil {
		t.Fatal(err)
	}
	if len(keep) != 1 || keep[0] != "svc-bad" {
		t.Fatalf("keep = %v, want [svc-bad]", keep)
	}
	entries, _ := readRegistrationEntries(path)
	if len(entries) != 1 || entries[0].Name != "svc-bad" || entries[0].State != RegistrationStateFailed {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestKeepEntriesOnAbort_DropsSucceeded(t *testing.T) {
	originalTs := map[string]int64{"svc-a": 11, "svc-b": 22}
	entries := keepEntriesOnAbort(
		[]string{"svc-a"}, // succeeded
		[]string{"svc-b"}, // still pending
		[]string{"ghost"}, // failed/unknown
		originalTs,
		100,
	)
	byName := map[string]RegistrationEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if _, ok := byName["svc-a"]; ok {
		t.Fatalf("succeeded svc-a must be dropped, got %+v", entries)
	}
	if byName["svc-b"].State != RegistrationStatePending || byName["svc-b"].RegisteredAt != 22 {
		t.Fatalf("svc-b = %+v", byName["svc-b"])
	}
	if byName["ghost"].State != RegistrationStateFailed || byName["ghost"].RegisteredAt != 100 {
		t.Fatalf("ghost = %+v", byName["ghost"])
	}
}

// 中断时不得把已成功重启的服务写回 pending。
func TestPreciseRestart_AbortKeepsOnlyUnprocessedAndFailed(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	cfg := &Config{
		Groups: []Group{{Services: []Service{
			{
				Name:        "svc-abort-a",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			},
			{
				Name:        "svc-abort-b",
				DependsOn:   []string{"svc-abort-a"},
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			},
		}}},
	}
	runner, store := testPreciseRestartRunner(t, cfg)
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	store.Init([]string{"svc-abort-a", "svc-abort-b"})
	store.Update("svc-abort-a", StatusStopped, "")
	store.Update("svc-abort-b", StatusStopped, "")

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"svc-abort-a", "svc-abort-b", "ghost-abort"}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			st := store.Get("svc-abort-a")
			if st != nil && st.Status == StatusHealthy {
				cancel()
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	if !runner.TryBeginPreciseRestart("precise-restart-abort") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	keep, err := runner.PreciseRestart(ctx, "test-session")
	// 取消可能命中循环入口（返回 ctx.Err）或命中 B 的健康检查失败（err==nil、b 记入 failed）。
	_ = err
	keepSet := map[string]bool{}
	for _, k := range keep {
		keepSet[k] = true
	}
	if keepSet["svc-abort-a"] {
		t.Fatalf("successful svc-abort-a must not be kept, keep=%v", keep)
	}
	if !keepSet["svc-abort-b"] || !keepSet["ghost-abort"] {
		t.Fatalf("keep = %v, want svc-abort-b and ghost-abort", keep)
	}
	entries, _ := readRegistrationEntries(path)
	byName := map[string]RegistrationState{}
	for _, e := range entries {
		byName[e.Name] = e.State
	}
	if byName["svc-abort-a"] != "" {
		t.Fatalf("svc-abort-a should be removed, entries=%v", entries)
	}
	// b 可能是 pending（循环入口中断）或 failed（重启中被取消）；均不可丢。
	if byName["svc-abort-b"] != RegistrationStatePending && byName["svc-abort-b"] != RegistrationStateFailed {
		t.Fatalf("svc-abort-b state = %q, want pending or failed", byName["svc-abort-b"])
	}
	if byName["ghost-abort"] != RegistrationStateFailed {
		t.Fatalf("ghost-abort state = %q, want failed", byName["ghost-abort"])
	}
}
