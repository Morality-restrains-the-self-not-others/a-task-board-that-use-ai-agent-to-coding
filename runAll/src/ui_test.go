package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func assertLifecycleAccepted(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["status"] != "accepted" {
		t.Fatalf("status body = %#v, want status=accepted", response)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func waitForServiceStatus(t *testing.T, store *StatusStore, name string, want Status, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := store.Get(name)
		if status != nil && status.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	status := store.Get(name)
	t.Fatalf("service %s status = %#v, want %q within %s", name, status, want, timeout)
}

func TestAPIStatus(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"redis", "kafka"})
	store.Update("redis", StatusHealthy, "")
	store.Update("kafka", StatusStarting, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var wrapper struct {
		Services []*ServiceStatus `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&wrapper); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	result := wrapper.Services
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestAPIStatus_IncludesPollIntervalMs(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"redis"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "redis", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:6379"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if got, ok := payload["poll_interval_ms"].(float64); !ok || int(got) != statusPollIntervalMs {
		t.Fatalf("poll_interval_ms = %#v, want %d", payload["poll_interval_ms"], statusPollIntervalMs)
	}
}

func TestAPIStatus_IncludesPortFields(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-port",
					Command:     "npm run dev --port 3000",
					HealthCheck: HealthCheck{URL: "http://localhost:8080/health"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if runner == nil {
		t.Fatal("runner is nil")
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var wrapper struct {
		Services []*ServiceStatus `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&wrapper); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	result := wrapper.Services
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].HealthPort != "8080" {
		t.Fatalf("health_port = %q, want 8080", result[0].HealthPort)
	}
	if result[0].CommandPort != "3000" {
		t.Fatalf("command_port = %q, want 3000", result[0].CommandPort)
	}
	if result[0].Group != "g1" {
		t.Fatalf("group = %q, want g1", result[0].Group)
	}
}

func TestAPIStatus_IncludesBuildableAndLanguage(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "svc-buildable",
					Command:      "npm run dev",
					BuildCommand: "npm run build",
					HealthCheck:  HealthCheck{URL: "http://localhost:4000/health"},
				},
				{
					Name:        "svc-no-build",
					Command:     "bash run.sh",
					Language:    "Go",
					HealthCheck: HealthCheck{URL: "http://localhost:8003/api/health/"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var wrapper struct {
		Services []map[string]any `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&wrapper); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	result := wrapper.Services
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}

	byName := map[string]map[string]any{}
	for _, row := range result {
		name, _ := row["name"].(string)
		byName[name] = row
	}

	if byName["svc-buildable"]["buildable"] != true {
		t.Fatalf("svc-buildable buildable = %#v, want true", byName["svc-buildable"]["buildable"])
	}
	if byName["svc-buildable"]["language"] != "JavaScript" {
		t.Fatalf("svc-buildable language = %#v, want JavaScript", byName["svc-buildable"]["language"])
	}
	if byName["svc-no-build"]["buildable"] != false {
		t.Fatalf("svc-no-build buildable = %#v, want false", byName["svc-no-build"]["buildable"])
	}
	if byName["svc-no-build"]["language"] != "Go" {
		t.Fatalf("svc-no-build language = %#v, want Go", byName["svc-no-build"]["language"])
	}
}

func TestUIIncludesFailureFields(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-failure-ui",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:8081/health"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.SetPID("svc-failure-ui", 12345)
	if err := runner.TakeoverService("svc-failure-ui", "session-ui-owner"); err != nil {
		t.Fatalf("TakeoverService: %v", err)
	}
	store.RecordFailure("svc-failure-ui", "readiness", "READINESS_TIMEOUT", "waiting on /health")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var wrapper struct {
		Services []map[string]any `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&wrapper); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	result := wrapper.Services
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0]["failure_phase"] != "readiness" {
		t.Fatalf("failure_phase = %#v, want readiness", result[0]["failure_phase"])
	}
	if result[0]["failure_code"] != "READINESS_TIMEOUT" {
		t.Fatalf("failure_code = %#v, want READINESS_TIMEOUT", result[0]["failure_code"])
	}
	if result[0]["hint"] != "[READINESS_TIMEOUT] waiting on /health" {
		t.Fatalf("hint = %#v, want [READINESS_TIMEOUT] waiting on /health", result[0]["hint"])
	}
	if result[0]["session_id"] != "session-ui-owner" {
		t.Fatalf("session_id = %#v, want session-ui-owner", result[0]["session_id"])
	}
}

func TestAPIStatus_RegressionMatrixPayload(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "matrix", Services: []Service{
				{
					Name:        "matrix-port-conflict",
					Command:     "python3 -m http.server 28180",
					HealthCheck: HealthCheck{URL: "http://127.0.0.1:28180/health"},
				},
				{
					Name:        "matrix-runtime-prereq-blocked",
					Command:     "docker compose up",
					HealthCheck: HealthCheck{URL: "http://127.0.0.1:28181/health"},
				},
				{
					Name:        "matrix-prereq-repaired-healthy",
					Command:     "docker compose up",
					HealthCheck: HealthCheck{URL: "http://127.0.0.1:28182/health"},
				},
				{
					Name:        "matrix-non-owner-rejected",
					Command:     "echo worker",
					HealthCheck: HealthCheck{URL: "http://127.0.0.1:28183/health"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.RecordPreflightFailure(
		"matrix-port-conflict",
		domain.ServiceFailureCodePortConflict,
		"PRECHECK_PORT_CONFLICT: foreign listener on 28180",
	)
	store.RecordPreflightFailure(
		"matrix-runtime-prereq-blocked",
		domain.ServiceFailureCodeRuntimePrereq,
		"PRECHECK_RUNTIME_PREREQ_FAILED: docker daemon unavailable",
	)
	store.Update("matrix-prereq-repaired-healthy", StatusHealthy, "")

	ownership, err := domain.NewServiceOwnership(
		"matrix-non-owner-rejected",
		"owner-session",
		4321,
		"config-hash",
		"http://127.0.0.1:28183/health",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}
	store.Update("matrix-non-owner-rejected", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var wrapper struct {
		Services []map[string]any `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&wrapper); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	payload := wrapper.Services
	if len(payload) != 4 {
		t.Fatalf("len = %d, want 4", len(payload))
	}

	byName := make(map[string]map[string]any, len(payload))
	for _, item := range payload {
		name, _ := item["name"].(string)
		byName[name] = item
	}
	for _, name := range []string{
		"matrix-port-conflict",
		"matrix-runtime-prereq-blocked",
		"matrix-prereq-repaired-healthy",
		"matrix-non-owner-rejected",
	} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("status payload missing service %q", name)
		}
	}

	if byName["matrix-port-conflict"]["failure_code"] != domain.ServiceFailureCodePortConflict {
		t.Fatalf("port conflict failure_code = %#v", byName["matrix-port-conflict"]["failure_code"])
	}
	if !strings.Contains(byName["matrix-port-conflict"]["hint"].(string), domain.ServiceFailureCodePortConflict) {
		t.Fatalf("port conflict hint should contain failure code, got %#v", byName["matrix-port-conflict"]["hint"])
	}

	if byName["matrix-runtime-prereq-blocked"]["failure_code"] != domain.ServiceFailureCodeRuntimePrereq {
		t.Fatalf("runtime prereq failure_code = %#v", byName["matrix-runtime-prereq-blocked"]["failure_code"])
	}
	if !strings.Contains(byName["matrix-runtime-prereq-blocked"]["hint"].(string), domain.ServiceFailureCodeRuntimePrereq) {
		t.Fatalf("runtime prereq hint should contain failure code, got %#v", byName["matrix-runtime-prereq-blocked"]["hint"])
	}

	if byName["matrix-prereq-repaired-healthy"]["status"] != string(StatusHealthy) {
		t.Fatalf("prereq repaired status = %#v, want healthy", byName["matrix-prereq-repaired-healthy"]["status"])
	}
	if gotCode := byName["matrix-prereq-repaired-healthy"]["failure_code"]; gotCode != nil && gotCode != "" {
		t.Fatalf("prereq repaired failure_code = %#v, want empty", gotCode)
	}

	if byName["matrix-non-owner-rejected"]["session_id"] != "owner-session" {
		t.Fatalf("non-owner case session_id = %#v, want owner-session", byName["matrix-non-owner-rejected"]["session_id"])
	}
}

func TestUIHomePage(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
	body := rec.Body.String()
	requiredSnippets := []string{
		`runAll Status`,
		`const dotClass = {`,
		`stopped: 'gray'`,
		`function fetchStatusData() {`,
		`function isServiceStartable(status) {`,
		`function getServiceStatus(name, data) {`,
		`const toggleAction = startable ? 'start' : 'stop';`,
		`data-action="stop-group"`,
		`data-action="build"`,
		`data-action="restart"`,
		`data-action="logs"`,
		`data-action="clear-logs"`,
		`id="logs-panel-loki"`,
		`id="logs-panel-grafana"`,
		`id="logs-panel-refresh"`,
		`id="logs-panel-copy"`,
		`function openGrafanaTraceLogs()`,
		`function extractTraceIdFromLogRows(rows)`,
		`id="observability-bar"`,
		`loadObservabilityBar`,
		`/api/observability`,
		`/api/observability/clear-all`,
		`function clearAllObservability()`,
		`id="obs-clear-all"`,
		`清空 Grafana 可观测数据`,
		`id="dev-tools-bar"`,
		`id="dev-generate-conf"`,
		`function generateConfReplica()`,
		`/api/conf/sync`,
		`生成配置副本`,
		`id="dev-catalog-link"`,
		`function openServiceUrlCatalog()`,
		`service-url-api-catalog`,
		`id="dev-clear-databases"`,
		`id="dev-init-databases"`,
		`id="dev-init-db-pending-label"`,
		`function refreshMigratePendingStatus(`,
		`/api/dev/migrate-status`,
		`function clearAllDatabases()`,
		`function initAllDatabases()`,
		`/api/dev/clear-databases`,
		`/api/dev/init-databases`,
		`/api/dev/bootstrap-admin-email`,
		`请先设置超级管理员邮箱`,
		`清空全部数据库（开发）`,
		`初始化全部数据库（开发）`,
		`function openLokiExplore()`,
		`/api/observability/grafana-trace`,
		`async function copyLogsToClipboard()`,
		`navigator.clipboard.writeText`,
		`id="logs-resize-handle"`,
		`function openLogsPanel(name)`,
		`function closeLogsPanel()`,
		`function postGroupAction(url, group, label)`,
		`function startAllServices()`,
		`id="start-all-btn"`,
		`全部启动`,
		`id="restart-all-btn"`,
		`全部重启`,
		`function restartAllServices()`,
		`/api/restart-all`,
		`/api/restart-all/cancel`,
		`/api/restart-all/progress`,
		`function connectRestartAllSSE()`,
		`全部重启进度`,
		`JSON.stringify({group: group, session_id: selectBulkActorSessionID()})`,
		`function selectBulkActorSessionID()`,
		`function lookupOwningSessionID(serviceName)`,
		`selectActorSessionID(explicitSessionID, name)`,
		`/api/build`,
		`/api/stop`,
		`/api/start`,
		`/api/start-all`,
		`/api/start-all/cancel`,
		`/api/stop-all/cancel`,
		`/api/build-all`,
		`/api/build-all/cancel`,
		`/api/build-all/progress`,
		`function connectBuildAllSSE()`,
		`function buildAllServices()`,
		`function buildGroup(group)`,
		`全部重新编译进度`,
		`分组重新编译进度`,
		`已有全部/分组重新编译进行中`,
		`function resumeActiveBulkProgress(data)`,
		`active_bulk_progress`,
		`_handledProgressDoneKey`,
		`progressDoneKey`,
		`shouldReconnectBulkProgressSSE`,
		`snap.event && snap.event.done`,
		`async function cancelProgressOperation(`,
		`id="prog-cancel-btn"`,
		`id="prog-copy-logs-btn"`,
		`id="prog-clear-logs-btn"`,
		`function dismissProgressPanel(`,
		`function copyProgressLogsToClipboard()`,
		`function buildProgressLogsText()`,
		`progress-header-actions`,
		`复制日志`,
		`/api/stop-group`,
		`/api/logs`,
		`/api/logs/clear`,
		`function normalizePortValue(port)`,
		`function escAttr(s)`,
		`function resolveHealthHref(url, fallbackPort)`,
		`function normalizePortHref(href)`,
		`event.target.closest('a.port-link')`,
		`target="_blank"`,
		`rel="noopener noreferrer"`,
		`class="ports"`,
		`class="service-extra"`,
		`grid-template-areas`,
		`service-group`,
		`group-toggle`,
		`toggle-group`,
		`function groupHealthDotClass(services)`,
		`function toggleGroupCollapsed(groupKey)`,
		`class="language"`,
		`svc.buildable === true`,
		`is-disabled`,
		`未配置 build_command，不可编译`,
		`id="dev-view-logs"`,
		`id="dev-log-select"`,
		`id="logs-panel-clear-dev"`,
		`function openDevLogsPanel(tool)`,
		`function closeDevLogsPanel()`,
		`function fetchDevLogsOnce()`,
		`function renderDevLogs(payload)`,
		`function clearDevLogs()`,
		`devToolOptions`,
		`devLogState`,
		`/api/dev/logs`,
		`/api/dev/logs/clear`,
		`function currentStatusRefreshMs()`,
		`function applyServerPollInterval(data)`,
		`function restartStatusRefreshTimer()`,
		`statusRefreshLogsOpenMultiplier`,
		`poll_interval_ms`,
		`restartStatusRefreshTimer();`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("home page missing required snippet %q", snippet)
		}
	}
}

func TestUIHomePage_LogCopyFailureFallbackPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, snippet := range []string{
		"function copyLogsToClipboard()",
		"updateLogsMeta('copy failed",
		"copied at",
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing log copy snippet %q", snippet)
		}
	}
}

func TestUIHomePage_PortLinkBoundaryGuardsPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		`function normalizePortValue(port)`,
		`function escAttr(s)`,
		`function resolveHealthHref(url, fallbackPort)`,
		`function httpEndpointLabel(href)`,
		`function portLink(href, fallbackPort`,
		`function commandPortCell(svc)`,
		`function healthPortCell(svc)`,
		`event.target.closest('a.port-link')`,
		`return '-'`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing port boundary snippet %q", snippet)
		}
	}
}

func TestUIHomePage_PortLinkSchemeGuardsPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		`function normalizePortHref(href)`,
		`http:' && parsed.protocol !== 'https:'`,
		`new URL(`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing port href scheme guard snippet %q", snippet)
		}
	}
}

func TestAPIBuild_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "svc-build",
					Command:      "echo running",
					BuildCommand: "echo built",
					HealthCheck:  HealthCheck{URL: "http://localhost:9999"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-build", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build", strings.NewReader(`{"name":"svc-build"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("status body = %#v, want status=ok", response)
	}
}

func TestAPIBuild_BadRequest(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "svc-build-bad",
					Command:      "echo running",
					BuildCommand: "echo built",
					HealthCheck:  HealthCheck{URL: "http://localhost:9998"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-build-bad", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing name", body: `{}`},
		{name: "unknown service", body: `{"name":"missing-service"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/build", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertJSONErrorResponse(t, rec, http.StatusBadRequest)
		})
	}
}

func TestAPIBuildGroup_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "bg-test", Services: []Service{
				{
					Name:         "bg-api-a",
					BuildCommand: "echo built-a",
					Command:      "echo running-a",
					HealthCheck:  HealthCheck{URL: "http://localhost:9801"},
				},
				{
					Name:         "bg-api-b",
					BuildCommand: "echo built-b",
					Command:      "echo running-b",
					HealthCheck:  HealthCheck{URL: "http://localhost:9802"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("bg-api-a", StatusHealthy, "")
	store.Update("bg-api-b", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build-group", strings.NewReader(`{"group":"bg-test"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["status"] != "accepted" {
		t.Fatalf("status = %v, want accepted", response["status"])
	}
	if response["run_id"] == "" {
		t.Fatalf("expected run_id, got %#v", response)
	}
	if response["group"] != "bg-test" {
		t.Fatalf("group = %q, want bg-test", response["group"])
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runner.GetActiveBuildAllRunID() == "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("build-group did not finish clearing active run id")
}

func TestAPIBuildGroup_GroupNotFound(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc",
					Command:     "echo svc",
					HealthCheck: HealthCheck{URL: "http://localhost:9803"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build-group", strings.NewReader(`{"group":"no-such-group"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertJSONErrorResponse(t, rec, http.StatusBadRequest)
}

func TestAPIBuildGroup_MissingGroup(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups:  []Group{},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing group", body: `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/build-group", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertJSONErrorResponse(t, rec, http.StatusBadRequest)
		})
	}
}

func TestAPIBuildGroup_MethodNotAllowed(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups:  []Group{},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/build-group", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertJSONErrorResponse(t, rec, http.StatusMethodNotAllowed)
}

func TestAPILogs_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-logs",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9997"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	entry1, err := domain.NewLogEntry(time.Now(), "svc-logs", domain.StreamStdout, "line-1")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	entry2, err := domain.NewLogEntry(time.Now(), "svc-logs", domain.StreamStderr, "line-2")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	entry3, err := domain.NewLogEntry(time.Now(), "svc-logs", domain.StreamStdout, "line-3")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	runner.logRepository.Append("svc-logs", entry1)
	runner.logRepository.Append("svc-logs", entry2)
	runner.logRepository.Append("svc-logs", entry3)
	store.Update("svc-logs", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/logs?name=svc-logs&lines=2", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Name  string            `json:"name"`
		Lines []domain.LogEntry `json:"lines"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response.Name != "svc-logs" {
		t.Fatalf("name = %q, want svc-logs", response.Name)
	}
	if len(response.Lines) != 2 {
		t.Fatalf("lines count = %d, want 2", len(response.Lines))
	}
	if response.Lines[0].Message != "line-2" || response.Lines[1].Message != "line-3" {
		t.Fatalf("unexpected log lines: %#v", response.Lines)
	}
}

func TestAPILogs_PendingServiceIncludesLifecycleDiagnostics(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-pending",
					Command:     "sleep 30",
					HealthCheck: HealthCheck{URL: "http://localhost:9997"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if st := store.Get("svc-pending"); st == nil || st.Status != StatusPending {
		t.Fatalf("svc-pending status = %#v, want pending", st)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/logs?name=svc-pending&lines=20", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Name  string            `json:"name"`
		Lines []domain.LogEntry `json:"lines"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(response.Lines) == 0 {
		t.Fatal("expected lifecycle diagnostics for pending service")
	}
	joined := ""
	for _, line := range response.Lines {
		joined += line.Message + "\n"
	}
	if !strings.Contains(joined, "not started") {
		t.Fatalf("expected not-started explanation in logs, got: %s", joined)
	}
}

func TestNewRunner_SeedsPendingLifecycleLog(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-seed",
					Command:     "sleep 1",
					HealthCheck: HealthCheck{URL: "http://localhost:9997"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	lines := runner.logRepository.Tail("svc-seed", 5)
	if len(lines) == 0 {
		t.Fatal("expected seeded lifecycle log for pending service")
	}
	if !strings.Contains(lines[0].Message, "default state: not started") {
		t.Fatalf("seed message = %q", lines[0].Message)
	}
}

func TestBuildStatusPayload_HidesPendingStatus(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-idle"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-idle", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	payload := buildStatusPayload(store, runner)
	if len(payload) != 1 {
		t.Fatalf("payload len = %d, want 1", len(payload))
	}
	if payload[0].Status != "" {
		t.Fatalf("display status = %q, want empty for default not-started", payload[0].Status)
	}
	if store.Get("svc-idle").Status != StatusPending {
		t.Fatalf("internal status = %q, want pending", store.Get("svc-idle").Status)
	}
}

func TestBuildStatusPayload_SkipsPortProbeForHealthyServices(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-healthy", "svc-stopped"})

	probeCalls := 0
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-healthy", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:18001"}},
			{Name: "svc-stopped", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:18002"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-healthy", StatusHealthy, "")
	store.Update("svc-stopped", StatusStopped, "")
	runner.listenerPIDsFn = func(string) ([]int, error) {
		probeCalls++
		return nil, nil
	}

	payload := buildStatusPayload(store, runner)
	if len(payload) != 2 {
		t.Fatalf("payload len = %d, want 2", len(payload))
	}
	for _, item := range payload {
		if item.Name == "svc-healthy" {
			if got := store.Get("svc-healthy").Status; got != StatusHealthy {
				t.Fatalf("svc-healthy internal status = %q, want healthy", got)
			}
			if shouldProbeListenPortForStatus(store.Get("svc-healthy").Status) {
				t.Fatalf("healthy service should not require port probe")
			}
		}
	}
	if probeCalls != 0 {
		t.Fatalf("per-port lsof calls = %d, want 0 (batch snapshot handles startable services)", probeCalls)
	}
	for _, item := range payload {
		if item.Name == "svc-healthy" && item.ListenPortActive {
			t.Fatalf("healthy service should not probe listen_port_active")
		}
		if item.Name == "svc-stopped" && item.ListenPortActive {
			t.Fatalf("stopped service with free port should report listen_port_active=false")
		}
	}
}

func TestAPIObservability_IncludesLoki(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: "/tmp/runall-logs"},
		Observability: Observability{
			GrafanaURL:        "http://127.0.0.1:3000",
			LokiURL:           "http://127.0.0.1:3100",
			TraceDashboardUID: "trace-log-journey",
			LocalPromtail:     true,
		},
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "ai-monitor",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9995"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/observability", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response["loki_url"] != "http://127.0.0.1:3100" {
		t.Fatalf("loki_url = %q", response["loki_url"])
	}
	if response["log_file_root"] != "/tmp/runall-logs" {
		t.Fatalf("log_file_root = %q", response["log_file_root"])
	}
	if !strings.Contains(response["grafana_loki_explore"], "/explore") {
		t.Fatalf("grafana_loki_explore = %q", response["grafana_loki_explore"])
	}
	if !strings.Contains(response["grafana_tempo_explore"], "/explore") {
		t.Fatalf("grafana_tempo_explore = %q", response["grafana_tempo_explore"])
	}
	if !strings.Contains(response["grafana_tempo_explore"], "tempo") {
		t.Fatalf("grafana_tempo_explore = %q", response["grafana_tempo_explore"])
	}
	if response["log_shipping"] != "local_promtail_remote_loki" {
		t.Fatalf("log_shipping = %q", response["log_shipping"])
	}
	if response["loki_push_url"] != "http://127.0.0.1:3100/loki/api/v1/push" {
		t.Fatalf("loki_push_url = %q", response["loki_push_url"])
	}
}

func TestAPIObservabilityGrafanaTrace_FromServiceLogs(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Observability: Observability{
			GrafanaURL:        "http://127.0.0.1:3000",
			TraceDashboardUID: "trace-log-journey",
		},
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "saas-backend",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9995"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	entry, err := domain.NewLogEntry(
		time.Now(),
		"saas-backend",
		domain.StreamStdout,
		`{"trace_id":"trace-api-test1234","msg":"relay accepted"}`,
	)
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	runner.logRepository.Append("saas-backend", entry)

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/observability/grafana-trace?service=saas-backend", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		TraceID string `json:"trace_id"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response.TraceID != "trace-api-test1234" {
		t.Fatalf("trace_id = %q", response.TraceID)
	}
	if !strings.Contains(response.URL, "var-trace_id=trace-api-test1234") {
		t.Fatalf("url = %q", response.URL)
	}
}

func TestAPIObservabilityClearAll(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: t.TempDir()},
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-obs-clear",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9979"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	entry, err := domain.NewLogEntry(time.Now(), "svc-obs-clear", domain.StreamStdout, "line")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	runner.logRepository.Append("svc-obs-clear", entry)

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/observability/clear-all", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["memory_services_cleared"].(float64) != 1 {
		t.Fatalf("memory_services_cleared = %#v", response["memory_services_cleared"])
	}
	if got := runner.logRepository.Tail("svc-obs-clear", 10); len(got) != 0 {
		t.Fatalf("remaining logs = %d, want 0", len(got))
	}
}

func TestAPILogs_BadRequest(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-logs-bad",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9996"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	tests := []struct {
		name string
		url  string
	}{
		{name: "missing name", url: "/api/logs?lines=10"},
		{name: "missing lines", url: "/api/logs?name=svc-logs-bad"},
		{name: "invalid lines", url: "/api/logs?name=svc-logs-bad&lines=abc"},
		{name: "non-positive lines", url: "/api/logs?name=svc-logs-bad&lines=0"},
		{name: "unknown service", url: "/api/logs?name=missing-service&lines=10"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertJSONErrorResponse(t, rec, http.StatusBadRequest)
		})
	}
}

func TestAPILogsClear_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-logs-clear",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9981"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	entry, err := domain.NewLogEntry(time.Now(), "svc-logs-clear", domain.StreamStdout, "line-before-clear")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	runner.logRepository.Append("svc-logs-clear", entry)

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{"name":"svc-logs-clear"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("status body = %#v, want status=ok", response)
	}
	if got := runner.logRepository.Tail("svc-logs-clear", 10); len(got) != 0 {
		t.Fatalf("remaining logs = %d, want 0", len(got))
	}
}

func TestAPILogsClearAll_TruncatesFilesWithoutRm(t *testing.T) {
	root := t.TempDir()
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Logging: Logging{FileRoot: root},
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "svc-a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				{Name: "svc-b", Command: "echo b", HealthCheck: HealthCheck{URL: "http://127.0.0.1:2"}},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	for _, name := range []string{"svc-a", "svc-b"} {
		entry, err := domain.NewLogEntry(time.Now(), name, domain.StreamStdout, "keep-inode-"+name)
		if err != nil {
			t.Fatalf("NewLogEntry: %v", err)
		}
		runner.logRepository.Append(name, entry)
	}
	pathA := filepath.Join(root, "svc-a.log")
	infoBefore, err := os.Stat(pathA)
	if err != nil {
		t.Fatalf("Stat before clear: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/logs/clear-all", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("response=%#v", response)
	}
	if response["method"] != "truncate" {
		t.Fatalf("method=%v, want truncate", response["method"])
	}
	if response["memory_services_cleared"] != float64(2) {
		t.Fatalf("memory_services_cleared=%v", response["memory_services_cleared"])
	}
	if got := runner.logRepository.Tail("svc-a", 10); len(got) != 0 {
		t.Fatalf("memory not cleared: %#v", got)
	}
	infoAfter, err := os.Stat(pathA)
	if err != nil {
		t.Fatalf("Stat after clear (file must still exist, not rm): %v", err)
	}
	if !os.SameFile(infoBefore, infoAfter) {
		t.Fatal("clear-all must truncate in place (same inode), not replace via rm")
	}
	data, err := os.ReadFile(pathA)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty file after truncate, got %q", string(data))
	}

	// Subsequent append must land on the same path Promtail tails.
	entry, err := domain.NewLogEntry(time.Now(), "svc-a", domain.StreamStdout, "after-clear-all probe")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	runner.logRepository.Append("svc-a", entry)
	data, err = os.ReadFile(pathA)
	if err != nil {
		t.Fatalf("ReadFile after append: %v", err)
	}
	if !strings.Contains(string(data), "after-clear-all probe") {
		t.Fatalf("expected probe on truncated path, got %q", string(data))
	}
}

func TestAPILogsClear_BadRequest(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-logs-clear-bad",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9980"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/logs/clear", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusMethodNotAllowed, "method not allowed")
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "invalid json")
	})

	t.Run("missing name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{"name":"   "}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "name is required")
	})

	t.Run("unknown service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{"name":"missing-service"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, `service "missing-service" not found`)
	})

	t.Run("runner nil", func(t *testing.T) {
		muxWithNilRunner := http.NewServeMux()
		registerUIHandlers(muxWithNilRunner, store, nil, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{"name":"svc-logs-clear-bad"}`))
		rec := httptest.NewRecorder()
		muxWithNilRunner.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "runner is required")
	})

	t.Run("log repository nil", func(t *testing.T) {
		runnerWithNilRepo, err := NewRunner(&Config{
			Version: "1",
			Groups: []Group{
				{Name: "g1", Services: []Service{
					{
						Name:        "svc-logs-clear-bad",
						Command:     "echo running",
						HealthCheck: HealthCheck{URL: "http://localhost:9980"},
					},
				}},
			},
		}, store)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}
		runnerWithNilRepo.logRepository = nil

		muxWithNilRepo := http.NewServeMux()
		registerUIHandlers(muxWithNilRepo, store, runnerWithNilRepo, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{"name":"svc-logs-clear-bad"}`))
		rec := httptest.NewRecorder()
		muxWithNilRepo.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "log repository is required")
	})
}

func TestAPIBuild_WhitespaceNameReturnsRequired(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "svc-build-space",
					Command:      "echo running",
					BuildCommand: "echo built",
					HealthCheck:  HealthCheck{URL: "http://localhost:9985"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build", strings.NewReader(`{"name":"   "}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertJSONErrorMessage(t, rec, http.StatusBadRequest, "name is required")
}

func TestAPILogs_WhitespaceNameReturnsRequired(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-logs-space",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9984"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/logs?name=%20%20%20&lines=50", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertJSONErrorMessage(t, rec, http.StatusBadRequest, "name is required")
}

func TestAPILogs_TooLargeLinesRejected(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-logs-cap",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9983"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/logs?name=svc-logs-cap&lines=5001", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertJSONErrorMessage(t, rec, http.StatusBadRequest, "lines must be <= 2000")
}

func TestUILogsPanel_CloseAndRefreshGuardsPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		"function closeLogsPanel()",
		"function stopLogsAutoRefresh()",
		"function fetchLogsOnce()",
		"function clearLogs(name)",
		"function handleLogsResizePointerDown(event)",
		"function handleLogsResizePointerMove(event)",
		"function stopLogsResize(event)",
		"setLogsPanelWidth(workspace.clientWidth * 0.38)",
		"position: absolute",
		".logs-resize-handle {",
		"addEventListener('pointerdown', handleLogsResizePointerDown)",
		"apiFetch('/api/logs/clear', {",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing logs modal guard snippet %q", snippet)
		}
	}
}

func TestUIHomePage_LayoutFlexColumnPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		"flex-direction: column",
		"flex: 1 1 auto",
		"scrollbar-gutter: stable",
		"findVerticalScrollableAncestor",
		"document.addEventListener('wheel'",
		"scrollbar-width: thin",
		`class="services-pane">`,
		`id="observability-bar"`,
		`id="dev-tools-bar"`,
	}
	forbiddenSnippets := []string{
		"calc(100vh - 120px)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing layout flex snippet %q", snippet)
		}
	}
	obsIdx := strings.Index(body, `id="observability-bar"`)
	paneIdx := strings.Index(body, `class="services-pane">`)
	if obsIdx < 0 || paneIdx < 0 || obsIdx < paneIdx {
		t.Fatalf("observability-bar must be nested inside .services-pane (pane=%d obs=%d)", paneIdx, obsIdx)
	}
	for _, snippet := range forbiddenSnippets {
		if strings.Contains(body, snippet) {
			t.Fatalf("status.html still contains deprecated layout snippet %q", snippet)
		}
	}
}

func TestUIHomePage_LogsContentScrollLayout(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		".logs-panel-header { display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid #1e293b; font-size: 13px; flex-shrink: 0; }",
		".logs-content { margin: 0; padding: 12px 14px; font-size: 12px; line-height: 1.45; white-space: pre-wrap; overflow: auto; color: #dbeafe; min-height: 0; flex: 1 1 auto; background: #020617; }",
	}
	forbiddenSnippets := []string{
		"min-height: 280px",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing logs content scroll layout snippet %q", snippet)
		}
	}
	for _, snippet := range forbiddenSnippets {
		if strings.Contains(body, snippet) {
			t.Fatalf("status.html still contains deprecated logs content snippet %q", snippet)
		}
	}
}

func TestUIHomePage_ColorSystemPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		"runAll UI Color System",
		"--ra-btn-bg-clicked",
		".action-btn.is-clicked",
		".restart-btn.is-clicked",
		".logs-btn.is-clicked",
		".clear-logs-btn.is-clicked",
		"clickFeedbackDurationMs = 300",
		"function pulseClickFeedback(buttonEl)",
		"pulseClickFeedback(event.currentTarget)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing color system snippet %q", snippet)
		}
	}
}

func TestUIHomePage_TraceShippingPanelPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, snippet := range []string{
		"id=\"trace-shipping-bar\"",
		"runTraceShippingVerify",
		"/api/trace-shipping/verify",
		"/api/trace-shipping/status",
		"id=\"internal-apis-smoke-bar\"",
		"runInternalAPIsSmokeVerify",
		"/api/smoke/internal-apis/verify",
		"/api/smoke/internal-apis/status",
		"extractInternalAPIsSmokeFailReasons",
		"ts-fail-detail",
		"原因:",
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing health smoke snippet %q", snippet)
		}
	}
}

func TestAPIBuild_StateConflictReturnsBadRequest(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "svc-build-conflict",
					Command:      "echo running",
					BuildCommand: "echo built",
					HealthCheck:  HealthCheck{URL: "http://localhost:9995"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-build-conflict", StatusBuilding, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build", strings.NewReader(`{"name":"svc-build-conflict"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertJSONErrorResponse(t, rec, http.StatusBadRequest)
}

func TestAPIMethodNotAllowed_ReturnsJSON(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:         "svc-method",
					Command:      "echo running",
					BuildCommand: "echo built",
					HealthCheck:  HealthCheck{URL: "http://localhost:9982"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "restart get", method: http.MethodGet, path: "/api/restart"},
		{name: "build get", method: http.MethodGet, path: "/api/build"},
		{name: "logs post", method: http.MethodPost, path: "/api/logs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertJSONErrorMessage(t, rec, http.StatusMethodNotAllowed, "method not allowed")
		})
	}
}

func TestAPIStopService(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-stop",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9986"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-stop", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stop", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusMethodNotAllowed, "method not allowed")
	})

	t.Run("missing name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"   "}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "name is required")
	})

	t.Run("missing session id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"svc-stop"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "session_id is required")
	})

	t.Run("runner nil", func(t *testing.T) {
		muxWithNilRunner := http.NewServeMux()
		registerUIHandlers(muxWithNilRunner, store, nil, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"svc-stop","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		muxWithNilRunner.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "runner is required")
	})

	t.Run("runner error passthrough", func(t *testing.T) {
		expectedErr := runner.StopServiceWithActor(context.Background(), "missing-service", "owner-session")
		if expectedErr == nil {
			t.Fatal("expected StopService error for missing service")
		}

		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"missing-service","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, expectedErr.Error())
	})

	t.Run("non owner rejected when cascade false", func(t *testing.T) {
		store.SetPID("svc-stop", 1234)
		ownership, err := domain.NewServiceOwnership(
			"svc-stop",
			"owner-session",
			1234,
			"config-hash",
			"http://localhost:9986",
			time.Now(),
		)
		if err != nil {
			t.Fatalf("NewServiceOwnership: %v", err)
		}
		if err := runner.ownershipRepo.Save(ownership); err != nil {
			t.Fatalf("Save ownership: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"svc-stop","session_id":"other-session","cascade":false}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertLifecycleAccepted(t, rec)
		// Non-owner is delegated to registered owner; stop should succeed.
		waitForServiceStatus(t, store, "svc-stop", StatusStopped, 2*time.Second)
	})

	t.Run("success", func(t *testing.T) {
		store.Update("svc-stop", StatusHealthy, "")

		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"svc-stop","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertLifecycleAccepted(t, rec)
		waitForServiceStatus(t, store, "svc-stop", StatusStopped, 2*time.Second)
	})
}

func TestAPIStartService(t *testing.T) {
	store := NewStatusStore()
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthServer.Close()

	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-start",
					Command:     "echo running",
					LaunchMode:  "detach", // short-lived launch; health is served externally
					HealthCheck: HealthCheck{URL: healthServer.URL},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("svc-start", StatusStopped, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/start", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusMethodNotAllowed, "method not allowed")
	})

	t.Run("missing name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/start", strings.NewReader(`{"name":"   "}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "name is required")
	})

	t.Run("missing session id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/start", strings.NewReader(`{"name":"svc-start"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "session_id is required")
	})

	t.Run("runner nil", func(t *testing.T) {
		muxWithNilRunner := http.NewServeMux()
		registerUIHandlers(muxWithNilRunner, store, nil, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/start", strings.NewReader(`{"name":"svc-start","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		muxWithNilRunner.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "runner is required")
	})

	t.Run("runner error passthrough", func(t *testing.T) {
		expectedErr := runner.StartServiceWithActor(context.Background(), "missing-service", "owner-session")
		if expectedErr == nil {
			t.Fatal("expected StartService error for missing service")
		}

		req := httptest.NewRequest(http.MethodPost, "/api/start", strings.NewReader(`{"name":"missing-service","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, expectedErr.Error())
	})

	t.Run("success", func(t *testing.T) {
		store.Update("svc-start", StatusStopped, "")

		req := httptest.NewRequest(http.MethodPost, "/api/start", strings.NewReader(`{"name":"svc-start","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertLifecycleAccepted(t, rec)
		waitForServiceStatus(t, store, "svc-start", StatusHealthy, 5*time.Second)
	})

	t.Run("on_failure skip startup failure returns error", func(t *testing.T) {
		failStore := NewStatusStore()
		failRunner, err := NewRunner(&Config{
			Version: "1",
			Groups: []Group{
				{Name: "g-fail", Services: []Service{
					{
						Name:    "svc-start-skip-fail",
						Command: "sleep 1",
						HealthCheck: HealthCheck{
							URL:     "http://127.0.0.1:65534/unhealthy",
							Timeout: 1,
							Retries: 1,
							Backoff: Backoff{
								Initial:    0.1,
								Max:        0.1,
								Multiplier: 1.0,
							},
						},
						OnFailure: "skip",
					},
				}},
			},
		}, failStore)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}
		stubNoPortListenersForTest(failRunner)
		failStore.Update("svc-start-skip-fail", StatusStopped, "")

		failMux := http.NewServeMux()
		registerUIHandlers(failMux, failStore, failRunner, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/start", strings.NewReader(`{"name":"svc-start-skip-fail","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		failMux.ServeHTTP(rec, req)
		assertLifecycleAccepted(t, rec)
		waitForServiceStatus(t, failStore, "svc-start-skip-fail", StatusFailed, 5*time.Second)
	})
}

func TestAPIRestart_OnFailureSkipStartupFailureReturnsError(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-restart-fail", Services: []Service{
				{
					Name:    "svc-restart-skip-fail",
					Command: "sleep 1",
					HealthCheck: HealthCheck{
						URL:     "http://127.0.0.1:65534/unhealthy",
						Timeout: 1,
						Retries: 1,
						Backoff: Backoff{
							Initial:    0.1,
							Max:        0.1,
							Multiplier: 1.0,
						},
					},
					OnFailure: "skip",
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-restart-skip-fail", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/restart", strings.NewReader(`{"name":"svc-restart-skip-fail","session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertLifecycleAccepted(t, rec)
	waitForServiceStatus(t, store, "svc-restart-skip-fail", StatusFailed, 5*time.Second)
}

func TestAPIRestart_RequiresSessionID(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-restart", Services: []Service{
				{
					Name:        "svc-restart",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9979"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-restart", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/restart", strings.NewReader(`{"name":"svc-restart"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertJSONErrorMessage(t, rec, http.StatusBadRequest, "session_id is required")
}

func TestAPIRestart_NonOwnerDelegatesToOwner(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-restart", Services: []Service{
				{
					Name:        "svc-restart-owned",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9978"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-restart-owned", StatusHealthy, "")
	ownership, err := domain.NewServiceOwnership(
		"svc-restart-owned",
		"owner-session",
		1234,
		"config-hash",
		"http://localhost:9978",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/restart", strings.NewReader(`{"name":"svc-restart-owned","session_id":"other-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertLifecycleAccepted(t, rec)
	// Non-owner is delegated to owner; must not fail ownership guard.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := store.Get("svc-restart-owned")
		if st != nil && st.Status != StatusFailed {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	st := store.Get("svc-restart-owned")
	if st != nil && st.Status == StatusFailed {
		t.Fatalf("restart failed unexpectedly: %+v", st)
	}
}

func TestAPIStopGroup(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-stop", Services: []Service{
				{
					Name:        "svc-stop-group",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9988"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("svc-stop-group", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stop-group", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusMethodNotAllowed, "method not allowed")
	})

	t.Run("missing group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/stop-group", strings.NewReader(`{"group":"   "}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "group is required")
	})

	t.Run("missing session id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/stop-group", strings.NewReader(`{"group":"g-stop"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "session_id is required")
	})

	t.Run("runner nil", func(t *testing.T) {
		muxWithNilRunner := http.NewServeMux()
		registerUIHandlers(muxWithNilRunner, store, nil, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/stop-group", strings.NewReader(`{"group":"g-stop","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		muxWithNilRunner.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, "runner is required")
	})

	t.Run("runner error passthrough", func(t *testing.T) {
		expectedErr := runner.StopGroup(context.Background(), "missing-group")
		if expectedErr == nil {
			t.Fatal("expected StopGroup error for missing group")
		}

		req := httptest.NewRequest(http.MethodPost, "/api/stop-group", strings.NewReader(`{"group":"missing-group","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertJSONErrorMessage(t, rec, http.StatusBadRequest, expectedErr.Error())
	})

	t.Run("non owner delegates to registered owner", func(t *testing.T) {
		store.Update("svc-stop-group", StatusHealthy, "")
		store.SetPID("svc-stop-group", 1234)
		ownership, err := domain.NewServiceOwnership(
			"svc-stop-group",
			"owner-session",
			1234,
			"config-hash",
			"http://localhost:9988",
			time.Now(),
		)
		if err != nil {
			t.Fatalf("NewServiceOwnership: %v", err)
		}
		if err := runner.ownershipRepo.Save(ownership); err != nil {
			t.Fatalf("Save ownership: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/stop-group", strings.NewReader(`{"group":"g-stop","session_id":"other-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertLifecycleAccepted(t, rec)
		waitForServiceStatus(t, store, "svc-stop-group", StatusStopped, 2*time.Second)
	})

	t.Run("success", func(t *testing.T) {
		store.Update("svc-stop-group", StatusHealthy, "")

		req := httptest.NewRequest(http.MethodPost, "/api/stop-group", strings.NewReader(`{"group":"g-stop","session_id":"owner-session"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertLifecycleAccepted(t, rec)
		waitForServiceStatus(t, store, "svc-stop-group", StatusStopped, 2*time.Second)
	})
}

func assertJSONErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, wantStatus, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	var body map[string]any
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	errMsg, _ := body["error"].(string)
	if strings.TrimSpace(errMsg) == "" {
		t.Fatalf("error body = %#v, want non-empty error", body)
	}
}

func assertJSONErrorMessage(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	assertJSONErrorResponse(t, rec, wantStatus)
	var body map[string]any
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	errMsg, _ := body["error"].(string)
	if errMsg != wantMessage {
		t.Fatalf("error = %q, want %q", errMsg, wantMessage)
	}
}

func TestReadCascadeFlag(t *testing.T) {
	if !readCascadeFlag(map[string]any{}) {
		t.Fatal("expected default cascade true when field missing")
	}
	if readCascadeFlag(map[string]any{"cascade": false}) {
		t.Fatal("expected cascade false")
	}
	if !readCascadeFlag(map[string]any{"cascade": true}) {
		t.Fatal("expected cascade true")
	}
}

func TestAPIStopService_CascadeFalseBlocksDownstream(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				{Name: "b", Command: "echo b", DependsOn: []string{"a"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("a", StatusHealthy, "")
	store.Update("b", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"a","session_id":"owner-session","cascade":false}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertLifecycleAccepted(t, rec)
	waitForServiceStatus(t, store, "a", StatusHealthy, 2*time.Second)
}

func TestAPIStopService_Preview(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "a", Command: "echo a", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				{Name: "b", Command: "echo b", DependsOn: []string{"a"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("a", StatusHealthy, "")
	store.Update("b", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	decodePreview := func(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body["status"] != "preview" {
			t.Fatalf("expected status=preview, got %v", body["status"])
		}
		return body
	}

	t.Run("preview returns cascade plan without stopping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"a","session_id":"owner-session","cascade":true,"preview":true}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		body := decodePreview(t, rec)

		svcList, ok := body["services"].([]any)
		if !ok {
			t.Fatalf("expected services array, got %v", body["services"])
		}
		names := make([]string, 0, len(svcList))
		for _, s := range svcList {
			names = append(names, fmt.Sprint(s))
		}
		if !containsString(names, "a") || !containsString(names, "b") {
			t.Fatalf("expected plan to include a and b (downstream), got %v", names)
		}
		// Preview must not mutate service state.
		if got := store.Get("a"); got.Status != StatusHealthy {
			t.Fatalf("expected a still Healthy, got %s", got.Status)
		}
		if got := store.Get("b"); got.Status != StatusHealthy {
			t.Fatalf("expected b still Healthy, got %s", got.Status)
		}
	})

	t.Run("leaf service preview contains only target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/stop", strings.NewReader(`{"name":"b","session_id":"owner-session","cascade":true,"preview":true}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		body := decodePreview(t, rec)

		svcList, ok := body["services"].([]any)
		if !ok {
			t.Fatalf("expected services array, got %v", body["services"])
		}
		if len(svcList) != 1 || fmt.Sprint(svcList[0]) != "b" {
			t.Fatalf("expected only b in leaf preview, got %v", svcList)
		}
	})
}

func TestAPIStartGroup(t *testing.T) {
	store := NewStatusStore()
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthServer.Close()

	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-start", Services: []Service{
				{
					Name:        "svc-group-start",
					Command:     "sleep 30",
					HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("svc-group-start", StatusStopped, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/start-group", strings.NewReader(`{"group":"g-start","session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertLifecycleAccepted(t, rec)
	waitForServiceStatus(t, store, "svc-group-start", StatusHealthy, 5*time.Second)
}

func TestAPIStartAll(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g-infra", Services: []Service{
				{
					Name:        "svc-all-infra",
					Command:     "sleep 30",
					HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
				},
			}},
			{Name: "g-app", Services: []Service{
				{
					Name:        "svc-all-app",
					DependsOn:   []string{"svc-all-infra"},
					Command:     "sleep 30",
					HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	store.Update("svc-all-infra", StatusStopped, "")
	store.Update("svc-all-app", StatusStopped, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/start-all", strings.NewReader(`{"session_id":"owner-session"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertLifecycleAccepted(t, rec)
	waitForServiceStatus(t, store, "svc-all-infra", StatusHealthy, 5*time.Second)
	waitForServiceStatus(t, store, "svc-all-app", StatusHealthy, 5*time.Second)
}

func TestUIDevToolLogs_Get(t *testing.T) {
	root := t.TempDir()
	rec := infrastructure.NewFileDevToolLogRecorder(root)
	rec.Append(domain.ToolDbClear, "test line 1")
	rec.Append(domain.ToolDbClear, "test line 2")

	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: rec}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest("GET", "/api/dev/logs?tool=db-clear&lines=10", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var payload struct {
		Tool  string   `json:"tool"`
		Lines []string `json:"lines"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if payload.Tool != "db-clear" {
		t.Errorf("tool = %q, want db-clear", payload.Tool)
	}
	if len(payload.Lines) != 2 {
		t.Errorf("lines count = %d, want 2", len(payload.Lines))
	}
}

func TestUIDevToolLogs_GetMissingTool(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: infrastructure.NewFileDevToolLogRecorder(t.TempDir())}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest("GET", "/api/dev/logs?lines=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestUIDevToolLogs_InvalidTool(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: infrastructure.NewFileDevToolLogRecorder(t.TempDir())}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest("GET", "/api/dev/logs?tool=bad-tool&lines=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestUIDevToolLogs_Clear(t *testing.T) {
	root := t.TempDir()
	rec := infrastructure.NewFileDevToolLogRecorder(root)
	rec.Append(domain.ToolDbClear, "test line")

	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: rec}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	body := strings.NewReader(`{"tool":"db-clear"}`)
	req := httptest.NewRequest("POST", "/api/dev/logs/clear", body)
	req.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	lines, _ := rec.Tail(domain.ToolDbClear, 10)
	if len(lines) != 0 {
		t.Errorf("after clear, Tail returned %d lines, want 0", len(lines))
	}
}

func TestUIDevToolLogs_ClearAll(t *testing.T) {
	root := t.TempDir()
	rec := infrastructure.NewFileDevToolLogRecorder(root)
	for _, tool := range domain.AllDevTools {
		rec.Append(tool, "test")
	}

	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: rec}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	body := strings.NewReader(`{"tool":"all"}`)
	req := httptest.NewRequest("POST", "/api/dev/logs/clear", body)
	req.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	for _, tool := range domain.AllDevTools {
		lines, _ := rec.Tail(tool, 10)
		if len(lines) != 0 {
			t.Errorf("after ClearAll, Tail(%q) returned %d lines, want 0", tool, len(lines))
		}
	}
}

func TestUICancelStartAllWithoutActiveOperation(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{store: store}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/start-all/cancel", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUICancelStopAllWithoutActiveOperation(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{store: store}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/stop-all/cancel", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUICancelStopAllActiveOperation(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{
				{
					Name:        "slow-a",
					Command:     "sleep 30",
					HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
				},
				{
					Name:        "slow-b",
					Command:     "sleep 30",
					HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
				},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	runner.progressBroadcaster = NewProgressBroadcaster()
	store.Update("slow-a", StatusHealthy, "")
	store.Update("slow-b", StatusHealthy, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	startReq := httptest.NewRequest(http.MethodPost, "/api/stop-all", strings.NewReader(`{"session_id":"owner-session"}`))
	startRec := httptest.NewRecorder()
	mux.ServeHTTP(startRec, startReq)
	assertLifecycleAccepted(t, startRec)

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/stop-all/cancel", nil)
	cancelRec := httptest.NewRecorder()
	mux.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", cancelRec.Code, cancelRec.Body.String())
	}
}

func TestAPIBuildAll_Accepted(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-build"})
	store.Update("svc-build", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:         "svc-build",
				Command:      "true",
				BuildCommand: "echo built",
				WorkingDir:   t.TempDir(),
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.progressBroadcaster = NewProgressBroadcaster()

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response["status"] != "accepted" {
		t.Fatalf("status body = %#v, want status=accepted", response)
	}
	if response["run_id"] == "" {
		t.Fatalf("expected run_id in response, got %#v", response)
	}

	// Wait for async build-all to finish so cancel registry is cleared for later tests.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runner.GetActiveBuildAllRunID() == "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("build-all did not finish clearing active run id")
}

func TestUICancelBuildAllWithoutActiveOperation(t *testing.T) {
	clearLifecycleCancel("build-all")
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{store: store}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/build-all/cancel", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestBuildAll_PublishesProgressEvents(t *testing.T) {
	dir := t.TempDir()
	store := NewStatusStore()
	store.Init([]string{"a", "b"})
	store.Update("a", StatusStopped, "")
	store.Update("b", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{
				{Name: "a", Command: "true", BuildCommand: "echo a", WorkingDir: dir},
				{Name: "b", Command: "true", BuildCommand: "echo b", WorkingDir: dir},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.progressBroadcaster = NewProgressBroadcaster()
	runID := runner.SetActiveBuildAllRunID()
	ch := runner.SubscribeProgress(runID)

	done := make(chan *domain.BuildGroupResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := runner.BuildAll(context.Background())
		if err != nil {
			errCh <- err
			return
		}
		done <- result
	}()

	var sawProgress, sawDone bool
	timeout := time.After(5 * time.Second)
	for !(sawProgress && sawDone) {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for build-all progress events")
		case ev, ok := <-ch:
			if !ok {
				if !sawDone {
					t.Fatal("channel closed before done event")
				}
				goto finished
			}
			if ev.Operation != "build" {
				t.Fatalf("operation = %q, want build", ev.Operation)
			}
			if ev.Phase == "progress" || ev.Phase == "starting" {
				sawProgress = true
			}
			if ev.Done {
				sawDone = true
			}
		case err := <-errCh:
			t.Fatalf("BuildAll error: %v", err)
		case result := <-done:
			if result.Built != 2 {
				t.Fatalf("built = %d, want 2", result.Built)
			}
		}
	}
finished:
}

func TestBuildGroup_PublishesProgressEvents(t *testing.T) {
	dir := t.TempDir()
	store := NewStatusStore()
	store.Init([]string{"ga", "gb"})
	store.Update("ga", StatusStopped, "")
	store.Update("gb", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g-build",
			Services: []Service{
				{Name: "ga", Command: "true", BuildCommand: "echo a", WorkingDir: dir},
				{Name: "gb", Command: "true", BuildCommand: "echo b", WorkingDir: dir},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := runner.SetActiveBuildAllRunID()
	ch := runner.SubscribeProgress(runID)

	errCh := make(chan error, 1)
	go func() {
		_, err := runner.BuildGroup(context.Background(), "g-build")
		errCh <- err
	}()

	var sawProgress, sawDone bool
	var built int
	timeout := time.After(5 * time.Second)
	for !sawDone {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for build-group progress events")
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("channel closed before done event")
			}
			if ev.Operation != "build" {
				t.Fatalf("operation = %q, want build", ev.Operation)
			}
			if ev.Phase == "progress" || ev.Phase == "starting" {
				sawProgress = true
			}
			if ev.Done {
				sawDone = true
				built = ev.Started
			}
		}
	}
	if err := <-errCh; err != nil {
		t.Fatalf("BuildGroup error: %v", err)
	}
	if !sawProgress {
		t.Fatal("expected at least one progress/starting event")
	}
	if built != 2 {
		t.Fatalf("started/built = %d, want 2", built)
	}
}

func TestTryBeginBuildAllRun_BusyLock(t *testing.T) {
	runner := &Runner{}
	runID, ok := runner.TryBeginBuildAllRun()
	if !ok || runID == "" {
		t.Fatalf("first TryBeginBuildAllRun failed: ok=%v runID=%q", ok, runID)
	}
	second, ok := runner.TryBeginBuildAllRun()
	if ok || second != "" {
		t.Fatalf("second TryBeginBuildAllRun should fail, got ok=%v runID=%q", ok, second)
	}
	if got := runner.GetActiveBuildAllRunID(); got != runID {
		t.Fatalf("active runID = %q, want %q", got, runID)
	}
}

func TestAPIBuildAll_ConflictWhenBusy(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-build"})
	store.Update("svc-build", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:         "svc-build",
				Command:      "true",
				BuildCommand: "sleep 2",
				WorkingDir:   t.TempDir(),
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`)))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want 202, body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	mux.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`)))
	if second.Code != http.StatusConflict {
		t.Fatalf("second status = %d, want 409, body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Body.String(), "already in progress") {
		t.Fatalf("expected busy error, got %s", second.Body.String())
	}

	// Also reject build-group while build-all is active.
	third := httptest.NewRecorder()
	mux.ServeHTTP(third, httptest.NewRequest(http.MethodPost, "/api/build-group", strings.NewReader(`{"group":"g1"}`)))
	if third.Code != http.StatusConflict {
		t.Fatalf("build-group status = %d, want 409, body=%s", third.Code, third.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runner.GetActiveBuildAllRunID() == "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("build-all did not finish clearing active run id")
}

func TestAPIBuildGroup_ConflictWhenBusy(t *testing.T) {
	store := NewStatusStore()
	dir := t.TempDir()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "bg-busy",
			Services: []Service{{
				Name:         "bg-busy-a",
				BuildCommand: "sleep 2",
				Command:      "true",
				WorkingDir:   dir,
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	store.Update("bg-busy-a", StatusStopped, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/build-group", strings.NewReader(`{"group":"bg-busy"}`)))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want 202, body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	mux.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/build-all", strings.NewReader(`{}`)))
	if second.Code != http.StatusConflict {
		t.Fatalf("build-all status = %d, want 409, body=%s", second.Code, second.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runner.GetActiveBuildAllRunID() == "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("build-group did not finish clearing active run id")
}

func TestAPIStatus_ActiveBulkProgress(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name:     "g1",
			Services: []Service{{Name: "svc", Command: "true"}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runID := runner.SetActiveBuildAllRunID()
	runner.PublishProgress(runID, StartAllProgressEvent{
		Total: 2, Started: 1, Remaining: 1, Phase: "progress", Operation: "build", Current: "svc",
	})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw, ok := payload["active_bulk_progress"].(map[string]any)
	if !ok {
		t.Fatalf("active_bulk_progress missing: %#v", payload)
	}
	if raw["kind"] != "build-all" {
		t.Fatalf("kind = %v, want build-all", raw["kind"])
	}
	if raw["run_id"] != runID {
		t.Fatalf("run_id = %v, want %s", raw["run_id"], runID)
	}
	ev, ok := raw["event"].(map[string]any)
	if !ok {
		t.Fatalf("event missing: %#v", raw)
	}
	if ev["current"] != "svc" {
		t.Fatalf("event.current = %v", ev["current"])
	}
}

func TestActiveBulkProgress_NilWhenIdle(t *testing.T) {
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if snap := runner.ActiveBulkProgress(); snap != nil {
		t.Fatalf("expected nil snapshot, got %#v", snap)
	}
}
