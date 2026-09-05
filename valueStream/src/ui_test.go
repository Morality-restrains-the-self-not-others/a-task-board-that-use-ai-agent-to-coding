package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func configWithFields() *Config {
	return &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{{
				Name:     "s1",
				TestFile: "a.py",
				Fields: []StepField{{
					Name: "saas-backend.auth_user.email",
				}},
			}},
		}},
	}
}

func TestAPIStreams_IncludesFieldViews(t *testing.T) {
	cfg := configWithFields()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/streams", nil))
	var resp StreamsResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	fields := resp.Streams[0].Steps[0].Fields
	if len(fields) != 1 || fields[0].ProviderService != "saas-backend" {
		t.Fatalf("fields = %+v", fields)
	}
	if resp.Streams[0].Domain != "auth" {
		t.Fatalf("domain = %q, want auth", resp.Streams[0].Domain)
	}
}

func TestAPIStreams(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	r := NewRunner(cfg, store)
	registerUIHandlers(mux, store, r)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/streams", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp StreamsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Streams) != 1 {
		t.Fatalf("len = %d", len(resp.Streams))
	}
	if resp.Streams[0].ImpactStatus != "unknown" {
		t.Fatalf("impact_status = %q, want unknown", resp.Streams[0].ImpactStatus)
	}
}

func TestUIHomePage(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestAPITestStep_ReturnsStarted(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "mock-pytest")
	os.WriteFile(mockPytest, []byte("#!/bin/sh\nexit 0\n"), 0755)
	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	pass := filepath.Join(proj, "pass.py")
	os.WriteFile(pass, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner:    RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps:  []Step{{Name: "s1", TestFile: "pass.py", TestFileAbs: pass}},
		}},
	}
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	body := `{"stream":"flow-a","step":"s1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test/step", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	var result map[string]string
	json.NewDecoder(rec.Body).Decode(&result)
	if result["status"] != "started" {
		t.Fatalf("result = %v", result)
	}
}

func TestAPITestStream_UnknownStream400(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	body := `{"stream":"no-such-stream"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body.String())
	}
}

func TestAPITestStep_UnknownStep400(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	body := `{"stream":"flow-a","step":"no-such-step"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test/step", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body.String())
	}
}

func TestAPITestStep_PlannedStep400WithClearError(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{Name: "future", Lifecycle: "planned", TestFile: "future.py"},
				{Name: "now", Lifecycle: "active", TestFile: "now.py"},
			},
		}},
	}
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	body := `{"stream":"flow-a","step":"future"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test/step", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body.String())
	}
	var result map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	msg := result["error"]
	if !strings.Contains(msg, "planned") || !strings.Contains(msg, "cannot be run") {
		t.Fatalf("want clear planned-step error, got %q", msg)
	}
}

func TestAPITestStream_ConcurrentSecondRequest409(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "slow-pytest")
	os.WriteFile(mockPytest, []byte("#!/bin/sh\nsleep 1\nexit 0\n"), 0755)
	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	pass := filepath.Join(proj, "pass.py")
	os.WriteFile(pass, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner:    RunnerConfig{WorkingDir: "proj", PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps:  []Step{{Name: "s1", TestFile: "pass.py", TestFileAbs: pass}},
		}},
	}
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	post := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/test/stream", strings.NewReader(`{"stream":"flow-a"}`))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if c := post(); c != http.StatusOK {
			t.Errorf("first request status = %d, want 200", c)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	if c := post(); c != http.StatusConflict {
		t.Fatalf("second request status = %d, want 409", c)
	}
	wg.Wait()
}

func TestAPITestStream_InvalidJSON400(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test/stream", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAPITestStream_MethodNotAllowed(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/test/stream", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestUIHomePageContainsFieldTableHeaders(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	for _, want := range []string{"valueStream", "数据字段", "postTest", "测试整条流"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestUIHomePageContainsImpactMarkers(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	for _, want := range []string{"影响：", "受影响", "未受影响", "未知", "受影响环节：", "impact_status", "impacted_steps", "step.impacted"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestUIHomePageContainsGroupingAndCollapseScripts(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	for _, want := range []string{"groupByDomainAndStatus", "toggleCollapse", "业务域：", "状态：", "折叠业务域", "折叠状态", "拖拽调整业务域顺序", "/api/domain-order", "domainDragInProgress", "if (domainDragInProgress) return;"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestUIHomePageCollapseKeysDoNotDependOnStatus(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()

	for _, want := range []string{
		"keyOf('stream', domain, stream.name)",
		"keyOf('step', domain, stream.name, step.name)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing collapse key pattern %q", want)
		}
	}
}

func TestUIHomePageRendersRightSideLogPanel(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`id="logPanel"`,
		`id="logPanelMeta"`,
		`id="logPanelBody"`,
		"selectStepLog(",
		"在右侧查看日志",
		"position: fixed; top: 0;",
		"updateLogPanelFixedLayout()",
		"window.addEventListener('resize', updateLogPanelFixedLayout)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing right-log-panel marker %q", want)
		}
	}
}

func TestUIHomePagePersistsLogScrollPositionAcrossRefresh(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()

	for _, want := range []string{
		"panelScrollTopByKey",
		"rememberPanelScroll()",
		"restorePanelScroll()",
		"logPanelBodyEl.addEventListener('scroll'",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing log-scroll persistence marker %q", want)
		}
	}
}

func TestUIHomePageLogPanelPreventsScrollChaining(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()

	for _, want := range []string{
		".log-panel-body",
		"overscroll-behavior: contain;",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing scroll-chain guard marker %q", want)
		}
	}
}

func TestUIHomePageSupportsPaneSplitterResize(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`id="paneSplitter"`,
		`id="leftPane"`,
		"startPaneResize(",
		"onPaneResizeMove(",
		"paneSplitterEl.addEventListener('pointerdown', startPaneResize)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing pane splitter marker %q", want)
		}
	}
}

func TestAPITestStream_Busy409(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	if !store.TryAcquire() {
		t.Fatal("expected to acquire")
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	body := `{"stream":"flow-a"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestAPIDomainOrder_ReordersAndPersists(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "value-stream.yaml")
	projDir := filepath.Join(dir, "proj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "a.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "b.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("version: \"1\"\nvalue_streams: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Version:    "1",
		ConfigDir:  dir,
		ConfigPath: configPath,
		Runner: RunnerConfig{
			WorkingDir: "proj",
		},
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "domain-a", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
			{Name: "flow-b", Domain: "domain-b", Steps: []Step{{Name: "s1", TestFile: "b.py"}}},
		},
	}
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/domain-order", strings.NewReader(`{"domain_order":["domain-b","domain-a"]}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp StreamsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Streams) != 2 || resp.Streams[0].Name != "flow-b" {
		t.Fatalf("unexpected stream order: %+v", resp.Streams)
	}
	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(saved)
	if strings.Index(text, "name: flow-b") > strings.Index(text, "name: flow-a") {
		t.Fatalf("yaml not reordered: %s", text)
	}
}

func TestAPIDomainOrder_BusyReturns409(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "value-stream.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"1\"\nvalue_streams: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Version:    "1",
		ConfigDir:  dir,
		ConfigPath: configPath,
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "domain-a", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
		},
	}
	store := NewStatusStore(cfg)
	if !store.TryAcquire() {
		t.Fatal("expected to acquire lock")
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/domain-order", strings.NewReader(`{"domain_order":["domain-a"]}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIDomainOrder_UnknownDomainReturns400(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "value-stream.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"1\"\nvalue_streams: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Version:    "1",
		ConfigDir:  dir,
		ConfigPath: configPath,
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "domain-a", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
		},
	}
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/domain-order", strings.NewReader(`{"domain_order":["domain-x"]}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIDomainOrder_PersistFailureReturns400(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "missing-dir", "value-stream.yaml")
	cfg := &Config{
		Version:    "1",
		ConfigDir:  dir,
		ConfigPath: configPath,
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "domain-a", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
		},
	}
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/domain-order", strings.NewReader(`{"domain_order":["domain-a"]}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIDomainOrder_MethodNotAllowed(t *testing.T) {
	cfg := minimalConfig()
	cfg.ConfigPath = filepath.Join(t.TempDir(), "value-stream.yaml")
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/domain-order", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestAPIDomainOrder_InvalidJSONReturns400(t *testing.T) {
	cfg := minimalConfig()
	cfg.ConfigPath = filepath.Join(t.TempDir(), "value-stream.yaml")
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, NewRunner(cfg, store))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/domain-order", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
