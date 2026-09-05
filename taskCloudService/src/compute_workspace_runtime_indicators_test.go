package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceRuntimeIndicatorsMachineAndContainer(t *testing.T) {
	setupCloudTestDB(t)

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-run",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-both",
		CommentID:   "cmt-both",
		Platform:    "mock",
		InstanceID:  "mock-abc",
		ServerURL:   "http://127.0.0.1:8765/",
		Region:      "cn-test",
		ZoneID:      "cn-test-a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-machine",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-machine",
		CommentID:   "cmt-machine",
		Platform:    "mock",
		InstanceID:  "mock-running",
		Region:      "cn-test",
		ZoneID:      "cn-test-a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfigHistory(CloudServerConfigHistory{
		ID:          "hist-open",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-open-hist",
		Platform:    "aliyun",
		InstanceID:  "i-hist",
		StartedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	// idle config in another workspace must not appear
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-other-ws",
		CompanyID:   "t1",
		WorkspaceID: "ws-other",
		TaskID:      "task-other",
		Platform:    "mock",
		InstanceID:  "mock-other",
		ServerURL:   "http://127.0.0.1:9/",
		Region:      "cn-test",
		ZoneID:      "cn-test-a",
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/cloud/compute/workspace-runtime-indicators/", nil)
	req.Header.Set("X-User-Id", "test-user")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceRuntimeIndicators(rec, req, "t1", "ws1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Status     string                      `json:"status"`
		Count      int                         `json:"count"`
		Indicators []workspaceRuntimeIndicator `json:"indicators"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "success" {
		t.Fatalf("status=%s", body.Status)
	}

	byTask := map[string]workspaceRuntimeIndicator{}
	for _, it := range body.Indicators {
		byTask[it.TaskID] = it
	}
	if got, ok := byTask["task-both"]; !ok || !got.MachineRunning || !got.ContainerRunning {
		t.Fatalf("task-both=%v ok=%v", got, ok)
	}
	if got, ok := byTask["task-machine"]; !ok || !got.MachineRunning || got.ContainerRunning {
		t.Fatalf("task-machine=%v ok=%v", got, ok)
	}
	// open history alone (no config / no Running status) must not count as started
	if _, ok := byTask["task-open-hist"]; ok {
		t.Fatalf("orphan open history must not appear as machine_running")
	}
	if _, ok := byTask["task-other"]; ok {
		t.Fatalf("other workspace task leaked")
	}
}

func TestWorkspaceRuntimeIndicatorsRouteDispatch(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-runtime-indicators/", nil)
	req.Header.Set("X-User-Id", "test-user")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-runtime-indicators/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"indicators"`) {
		t.Fatalf("expected indicators: %s", rec.Body.String())
	}
}

func TestWorkspaceRuntimeIndicatorsIgnoresOrphanServerURL(t *testing.T) {
	setupCloudTestDB(t)

	// 停机/释放后残留：无 instance_id、无 Running 态，但 server_url 仍在
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-orphan", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-orphan",
		Platform: "mock", ServerURL: "http://127.0.0.1:8765/",
		Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	// Stopped 机器 + 残留 server_url
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-stopped", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-stopped",
		Platform: "aliyun", InstanceID: "i-stopped", ServerURL: "http://10.0.0.1:8765/",
		Region: "cn-test", ZoneID: "a", LastRuntimeStatus: "Stopped",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := listWorkspaceRuntimeIndicators("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.TaskID == "task-orphan" || it.TaskID == "task-stopped" {
			t.Fatalf("stale reachability must not appear as container_running: %+v", it)
		}
	}

	orphan, err := loadCloudServerConfig("t1", "ws1", "task-orphan")
	if err != nil || orphan == nil {
		t.Fatalf("load orphan: %v cfg=%v", err, orphan)
	}
	if strings.TrimSpace(orphan.ServerURL) == "" {
		t.Fatalf("GET indicators must not purge server_url, got empty")
	}
	stopped, err := loadCloudServerConfig("t1", "ws1", "task-stopped")
	if err != nil || stopped == nil {
		t.Fatalf("load stopped: %v cfg=%v", err, stopped)
	}
	if strings.TrimSpace(stopped.ServerURL) == "" {
		t.Fatalf("GET indicators must not purge stopped server_url, got empty")
	}

	purgeStaleContainerReachability("t1", "ws1")
	orphan, err = loadCloudServerConfig("t1", "ws1", "task-orphan")
	if err != nil || orphan == nil {
		t.Fatalf("load orphan after purge: %v cfg=%v", err, orphan)
	}
	if strings.TrimSpace(orphan.ServerURL) != "" {
		t.Fatalf("timer/purge must clear orphan server_url, got %q", orphan.ServerURL)
	}
	stopped, err = loadCloudServerConfig("t1", "ws1", "task-stopped")
	if err != nil || stopped == nil {
		t.Fatalf("load stopped after purge: %v cfg=%v", err, stopped)
	}
	if strings.TrimSpace(stopped.ServerURL) != "" {
		t.Fatalf("timer/purge must clear stopped server_url, got %q", stopped.ServerURL)
	}
}

func TestBuildContainerTaskUIContextOrphanServerURL(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-ui-orphan", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-ui-orphan",
		CommentID: testLiveCommentID,
		Platform:  "mock", ServerURL: "http://127.0.0.1:8765/",
		BusinessAPIEndpoint: "http://127.0.0.1:8765/",
		Region:              "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	payload := buildContainerTaskUIContextScoped("t1", "ws1", "task-ui-orphan", testLiveCommentID)
	if payload["container_endpoint_registered"] != false {
		t.Fatalf("orphan must not report registered: %v", payload)
	}
	if payload["container_page_url"] != "" {
		t.Fatalf("orphan must not expose page url: %v", payload)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-ui-orphan", testLiveCommentID)
	if err != nil || cfg == nil {
		t.Fatalf("load: %v cfg=%v", err, cfg)
	}
	if strings.TrimSpace(cfg.ServerURL) != "" {
		t.Fatalf("UI context path must purge orphan server_url, got %q", cfg.ServerURL)
	}
}
