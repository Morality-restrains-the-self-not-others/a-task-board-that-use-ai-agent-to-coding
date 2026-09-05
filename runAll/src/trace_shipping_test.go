package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"runAll/src/domain"
)

type stubLokiClient struct {
	readyErr error
	counts   map[string]int
	countErr error
}

func (s *stubLokiClient) Ready(context.Context) error { return s.readyErr }

func (s *stubLokiClient) CountLogsByJob(context.Context, string, time.Duration) (map[string]int, error) {
	if s.countErr != nil {
		return nil, s.countErr
	}
	return s.counts, nil
}

func TestPollLokiForProbeCounts_WaitsUntilAllServicesVisible(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Observability: Observability{
			LokiURL: "http://127.0.0.1:3100",
		},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			{Name: "svc-b", Command: "echo b", HealthCheck: HealthCheck{URL: "http://127.0.0.1:2"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	calls := 0
	origVerify := verifyTraceShippingLoki
	verifyTraceShippingLoki = func(r *Runner, ctx context.Context, probeID string) (bool, string, map[string]int, error) {
		calls++
		if calls < 3 {
			return true, "", map[string]int{"svc-a": 1}, nil
		}
		return true, "", map[string]int{"svc-a": 1, "svc-b": 1}, nil
	}
	defer func() { verifyTraceShippingLoki = origVerify }()

	start := time.Now()
	ready, errStr, counts := pollLokiForProbeCounts(context.Background(), runner, "runall-ship-test", []string{"svc-a", "svc-b"}, 2*time.Second)
	elapsed := time.Since(start)
	if !ready || errStr != "" {
		t.Fatalf("ready=%v err=%q", ready, errStr)
	}
	if counts["svc-a"] != 1 || counts["svc-b"] != 1 {
		t.Fatalf("counts=%v", counts)
	}
	if calls < 3 {
		t.Fatalf("calls=%d, want polling until all services visible", calls)
	}
	if elapsed < 500*time.Millisecond {
		t.Fatalf("expected poll delay, elapsed=%v", elapsed)
	}
}

func TestParseTraceShippingWait(t *testing.T) {
	defaultWait := traceShippingDefaultWait
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"", defaultWait},
		{"0", 0},
		{"5", 5 * time.Second},
		{"30", 30 * time.Second},
		{"31", defaultWait},
		{"-1", defaultWait},
		{"abc", defaultWait},
	}
	for _, tc := range cases {
		got := parseTraceShippingWait(tc.raw, defaultWait)
		if got != tc.want {
			t.Fatalf("parseTraceShippingWait(%q)=%v want %v", tc.raw, got, tc.want)
		}
	}
}

func TestVerifyTraceShipping_SkipWaitCompletesQuickly(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Observability: Observability{
			GrafanaURL:        "http://127.0.0.1:3000",
			TraceDashboardUID: "trace-log-journey",
			LokiURL:           "http://127.0.0.1:3100",
		},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	origVerify := verifyTraceShippingLoki
	verifyTraceShippingLoki = func(r *Runner, ctx context.Context, probeID string) (bool, string, map[string]int, error) {
		return true, "", map[string]int{"svc-a": 1}, nil
	}
	defer func() { verifyTraceShippingLoki = origVerify }()

	start := time.Now()
	report, err := runner.VerifyTraceShipping(context.Background(), 0)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("VerifyTraceShipping: %v", err)
	}
	if elapsed >= 6*time.Second {
		t.Fatalf("expected skip wait to finish quickly, took %v", elapsed)
	}
	if report.PromtailReload == "" {
		t.Fatalf("expected promtail_reload to be set, got %q", report.PromtailReload)
	}
}

func TestVerifyTraceShipping_LocalAndLoki(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Observability: Observability{
			GrafanaURL:        "http://127.0.0.1:3000",
			TraceDashboardUID: "trace-log-journey",
			LokiURL:           "http://127.0.0.1:3100",
		},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			{Name: "svc-b", Command: "echo b", HealthCheck: HealthCheck{URL: "http://127.0.0.1:2"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	origVerify := verifyTraceShippingLoki
	verifyTraceShippingLoki = func(r *Runner, ctx context.Context, probeID string) (bool, string, map[string]int, error) {
		return true, "", map[string]int{"svc-a": 1}, nil
	}
	defer func() { verifyTraceShippingLoki = origVerify }()

	report, err := runner.VerifyTraceShipping(context.Background(), 0)
	if err != nil {
		t.Fatalf("VerifyTraceShipping: %v", err)
	}
	if report.Summary.Total != 2 {
		t.Fatalf("total=%d", report.Summary.Total)
	}
	if report.Summary.ConfiguredTotal != 2 {
		t.Fatalf("configured_total=%d", report.Summary.ConfiguredTotal)
	}
	if report.Summary.LocalOK != 2 {
		t.Fatalf("local_ok=%d", report.Summary.LocalOK)
	}
	if report.Summary.LokiOK != 1 {
		t.Fatalf("loki_ok=%d", report.Summary.LokiOK)
	}
	if report.Summary.Failed != 1 {
		t.Fatalf("failed=%d", report.Summary.Failed)
	}
	if report.VerificationStatus != traceShippingStatusVerified {
		t.Fatalf("verification_status=%q", report.VerificationStatus)
	}
	foundB := false
	for _, svc := range report.Services {
		if svc.ServiceName == "svc-b" && svc.Issue != "" && !svc.LokiVisible {
			foundB = true
		}
	}
	if !foundB {
		t.Fatalf("expected svc-b issue, got %+v", report.Services)
	}
}

func TestTraceShippingReportForUI_AlignsNewConfiguredService(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Observability: Observability{
			GrafanaURL:        "http://127.0.0.1:3000",
			TraceDashboardUID: "trace-log-journey",
			LokiURL:           "http://127.0.0.1:3100",
		},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			{Name: "svc-b", Command: "echo b", HealthCheck: HealthCheck{URL: "http://127.0.0.1:2"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	runner.traceShippingMu.Lock()
	runner.traceShippingLast = &domain.TraceShippingReport{
		ProbeTraceID:       "runall-ship-test",
		VerifiedAt:         time.Now().UTC().Format(time.RFC3339),
		VerificationStatus: traceShippingStatusVerified,
		LokiReady:          true,
		Summary: domain.TraceShippingSummary{
			Total:           1,
			ConfiguredTotal: 1,
			LocalOK:         1,
			LokiOK:          1,
		},
		Services: []domain.TraceShippingServiceResult{
			{ServiceName: "svc-a", LocalProbeOK: true, LokiVisible: true, LokiLogCount: 1},
		},
	}
	runner.traceShippingMu.Unlock()

	report := runner.TraceShippingReportForUI()
	if report == nil {
		t.Fatal("expected aligned report")
	}
	if report.Summary.ConfiguredTotal != 2 {
		t.Fatalf("configured_total=%d", report.Summary.ConfiguredTotal)
	}
	if report.VerificationStatus != traceShippingStatusStale {
		t.Fatalf("verification_status=%q", report.VerificationStatus)
	}
	foundB := false
	for _, svc := range report.Services {
		if svc.ServiceName == "svc-b" && svc.Issue == traceShippingPendingIssue {
			foundB = true
		}
	}
	if !foundB {
		t.Fatalf("expected svc-b pending issue, got %+v", report.Services)
	}
}

func TestConfiguredServicesForTraceShipping_ReloadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runAll.yaml")
	if err := os.WriteFile(cfgPath, []byte(`version: "1"
groups:
  - name: g1
    services:
      - name: svc-a
        command: echo a
        health_check:
          url: http://127.0.0.1:1
      - name: svc-b
        command: echo b
        health_check:
          url: http://127.0.0.1:2
      - name: svc-c
        command: echo c
        health_check:
          url: http://127.0.0.1:3
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetConfigPath(cfgPath)

	services := runner.configuredServicesForTraceShipping()
	if len(services) != 3 {
		t.Fatalf("services=%d", len(services))
	}
}

func TestAPITraceShippingStatus_IncludesAllConfiguredServicesBeforeVerify(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			{Name: "svc-b", Command: "echo b", HealthCheck: HealthCheck{URL: "http://127.0.0.1:2"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/trace-shipping/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "pending" {
		t.Fatalf("status=%v", payload["status"])
	}
	summary, ok := payload["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary=%v", payload["summary"])
	}
	if int(summary["configured_total"].(float64)) != 2 {
		t.Fatalf("configured_total=%v", summary["configured_total"])
	}
	services, ok := payload["services"].([]interface{})
	if !ok || len(services) != 2 {
		t.Fatalf("services=%v", payload["services"])
	}
}

func TestAPITraceShippingVerify(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Observability: Observability{
			GrafanaURL:        "http://127.0.0.1:3000",
			TraceDashboardUID: "trace-log-journey",
			LokiURL:           "http://127.0.0.1:3100",
		},
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "saas-backend", Command: "echo", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	origVerify := verifyTraceShippingLoki
	verifyTraceShippingLoki = func(r *Runner, ctx context.Context, probeID string) (bool, string, map[string]int, error) {
		return true, "", map[string]int{"saas-backend": 1}, nil
	}
	defer func() { verifyTraceShippingLoki = origVerify }()

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/trace-shipping/verify?wait_seconds=0", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var report domain.TraceShippingReport
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(report.ProbeTraceID, "runall-ship-") {
		t.Fatalf("probe=%q", report.ProbeTraceID)
	}
	if report.Summary.LokiOK != 1 {
		t.Fatalf("loki_ok=%d", report.Summary.LokiOK)
	}
	if report.Summary.ConfiguredTotal != 1 {
		t.Fatalf("configured_total=%d", report.Summary.ConfiguredTotal)
	}
	if report.PromtailReload == "" {
		t.Fatalf("promtail_reload=%q", report.PromtailReload)
	}
}

func TestReloadPromtail_SkipsSharedContainerOnGoTestTempRoot(t *testing.T) {
	// A Runner whose FileRoot is a Go t.TempDir() must never reach the real
	// promtail reload shell — that would repoint the shared aimonitor-promtail
	// container at an empty temp dir and starve Loki (OPT-20260903-004).
	runner := &Runner{cfg: &Config{Logging: Logging{FileRoot: t.TempDir()}}}
	invoked := false
	origRun := reloadPromtailShellRun
	reloadPromtailShellRun = func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
		invoked = true
		return []byte("ok"), nil
	}
	defer func() { reloadPromtailShellRun = origRun }()

	if status := runner.reloadPromtailForTraceShipping(context.Background()); status != "skipped" {
		t.Fatalf("reloadPromtailForTraceShipping() = %q, want \"skipped\" for t.TempDir() log root", status)
	}
	if invoked {
		t.Fatal("promtail reload shell executed for t.TempDir() log root; shared aimonitor-promtail would be hijacked")
	}
}

func TestPromtailLogRoot_FlagsGoTestTempRoot(t *testing.T) {
	if _, prod := promtailLogRoot(&Runner{cfg: &Config{Logging: Logging{FileRoot: t.TempDir()}}}); prod {
		t.Fatal("promtailLogRoot(t.TempDir()) reported production=true")
	}
	realRoots := []string{
		"/tmp/ram-work/logs",
		"/home/ljy/bin/daydaymoney-deploy/logs",
		filepath.Join(t.TempDir(), "..", "..", "ram-work", "logs"), // resolves under /tmp but not a Test* dir
	}
	for _, root := range realRoots {
		if _, prod := promtailLogRoot(&Runner{cfg: &Config{Logging: Logging{FileRoot: root}}}); !prod {
			t.Fatalf("promtailLogRoot(%q) reported production=false for real log root", root)
		}
	}
	if _, prod := promtailLogRoot(&Runner{cfg: &Config{}}); prod {
		t.Fatal("promtailLogRoot(empty cfg) reported production=true")
	}
}
