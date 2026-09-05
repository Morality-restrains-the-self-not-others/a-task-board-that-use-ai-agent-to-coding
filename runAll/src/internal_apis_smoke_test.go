package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSummarizeSmokeOutput_PrefersFailLines(t *testing.T) {
	out := "" +
		"=== internal API live smoke ===\n" +
		"PASS  saas /api/health/ (200)\n" +
		"FAIL  feature-params/tenant — HTTP 410\n" +
		"PASS  taskbill pricing-packages (200)\n" +
		"=== summary: pass=7 fail=1 skip=0 ===\n"
	got := summarizeSmokeOutput(out)
	if !strings.Contains(got, "FAIL  feature-params/tenant — HTTP 410") {
		t.Fatalf("want FAIL detail in message, got %q", got)
	}
	if !strings.Contains(got, "=== summary: pass=7 fail=1 skip=0 ===") {
		t.Fatalf("want summary retained, got %q", got)
	}
}

func TestSummarizeSmokeOutput_SummaryOnlyWhenNoFails(t *testing.T) {
	out := "PASS  a\n=== summary: pass=1 fail=0 skip=0 ===\n"
	got := summarizeSmokeOutput(out)
	if got != "=== summary: pass=1 fail=0 skip=0 ===" {
		t.Fatalf("got %q", got)
	}
}

func TestVerifyInternalAPIsSmoke_UsesScriptHook(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc", WorkingDir: t.TempDir(), Command: "true", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	orig := runInternalAPIsSmokeScript
	runInternalAPIsSmokeScript = func(ctx context.Context, root, mode string) (int, string, error) {
		if mode != "required" {
			t.Fatalf("mode=%q, want required", mode)
		}
		return 0, "=== summary: pass=8 fail=0 skip=0 ===\n", nil
	}
	defer func() { runInternalAPIsSmokeScript = orig }()

	rep := runner.VerifyInternalAPIsSmoke(context.Background(), "manual")
	if rep.Status != internalAPIsSmokeStatusOK {
		t.Fatalf("status=%q want ok; msg=%q", rep.Status, rep.Message)
	}
	if !strings.Contains(rep.Message, "pass=8") {
		t.Fatalf("message=%q", rep.Message)
	}
	got := runner.GetInternalAPIsSmokeReport()
	if got.Status != internalAPIsSmokeStatusOK || got.Trigger != "manual" {
		t.Fatalf("stored=%+v", got)
	}
}

func TestAPIInternalAPIsSmokeVerify(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc", WorkingDir: t.TempDir(), Command: "true", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	orig := runInternalAPIsSmokeScript
	runInternalAPIsSmokeScript = func(ctx context.Context, root, mode string) (int, string, error) {
		return 1, "=== summary: pass=1 fail=2 skip=0 ===\n", nil
	}
	defer func() { runInternalAPIsSmokeScript = orig }()

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/smoke/internal-apis/verify", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var rep InternalAPIsSmokeReport
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
		t.Fatalf("json: %v", err)
	}
	if rep.Status != internalAPIsSmokeStatusFailed {
		t.Fatalf("status=%q", rep.Status)
	}

	reqStatus := httptest.NewRequest(http.MethodGet, "/api/smoke/internal-apis/status", nil)
	recStatus := httptest.NewRecorder()
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status endpoint=%d", recStatus.Code)
	}
}

func TestMaybeRunInternalAPIsSmokeAfterStartAll_SkipsOnFailure(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc", WorkingDir: t.TempDir(), Command: "true", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	called := false
	orig := runInternalAPIsSmokeScript
	runInternalAPIsSmokeScript = func(ctx context.Context, root, mode string) (int, string, error) {
		called = true
		return 0, "ok", nil
	}
	defer func() { runInternalAPIsSmokeScript = orig }()

	runner.maybeRunInternalAPIsSmokeAfterStartAll(1)
	time.Sleep(50 * time.Millisecond)
	if called {
		t.Fatal("expected smoke not to run when start-all had failures")
	}
}
