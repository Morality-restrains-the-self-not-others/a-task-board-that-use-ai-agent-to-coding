package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerRuntimeStatusReleasedClearsLocalBinding(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-gone2','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-gone2", "t1", "ws1", "task-gone2", testLiveCommentID, "aliyun", "i-gone-2", "http://old/", "cn-hongkong", "cn-hongkong-b", "auth-gone2", "Running",
	)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_config_histories(id, company_id, workspace_id, task_id, platform, instance_id, started_at, created_at)
		 VALUES (?,?,?,?,?,?,NOW(),NOW())`,
		"hist-gone2", "t1", "ws1", "task-gone2", "aliyun", "i-gone-2",
	)
	if err != nil {
		t.Fatalf("seed history: %v", err)
	}

	old := describeInstanceForRuntime
	t.Cleanup(func() { describeInstanceForRuntime = old })
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		return nil, false, "req-empty", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-gone2&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-gone2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] != "Released" || body["released"] != true {
		t.Fatalf("body=%v", body)
	}

	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-gone2", testLiveCommentID)
	if err != nil || cfg == nil {
		t.Fatalf("load config: %v cfg=%v", err, cfg)
	}
	if strings.TrimSpace(cfg.InstanceID) != "" || strings.TrimSpace(cfg.ServerURL) != "" {
		t.Fatalf("Released must clear local binding: instance_id=%q server_url=%q", cfg.InstanceID, cfg.ServerURL)
	}
	open, err := loadOpenCloudServerConfigHistory("t1", "ws1", "task-gone2")
	if err != nil {
		t.Fatal(err)
	}
	if open != nil {
		t.Fatal("Released must close open history")
	}
}

func TestWorkspaceRuntimeIndicatorsRespectsRuntimeStatus(t *testing.T) {
	setupCloudTestDB(t)

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-run", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-running", CommentID: "cmt-run",
		Platform: "aliyun", InstanceID: "i-run", Region: "cn-test", ZoneID: "a",
		LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-start", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-starting", CommentID: "cmt-start",
		Platform: "aliyun", InstanceID: "i-start", Region: "cn-test", ZoneID: "a",
		LastRuntimeStatus: "Starting",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-rel", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-released", CommentID: "cmt-rel",
		Platform: "aliyun", InstanceID: "i-rel", Region: "cn-test", ZoneID: "a",
		LastRuntimeStatus: "Released",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-mock", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-mock", CommentID: "cmt-mock",
		Platform: "mock", InstanceID: "mock-1", ServerURL: "http://127.0.0.1:1/",
		Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := listWorkspaceRuntimeIndicators("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	byTask := map[string]workspaceRuntimeIndicator{}
	for _, it := range items {
		byTask[it.TaskID] = it
	}
	if got, ok := byTask["task-running"]; !ok || !got.MachineRunning || got.MachineStarting {
		t.Fatalf("task-running=%v ok=%v", got, ok)
	}
	if got, ok := byTask["task-starting"]; !ok || got.MachineRunning || !got.MachineStarting {
		t.Fatalf("task-starting=%v ok=%v", got, ok)
	}
	if _, ok := byTask["task-released"]; ok {
		t.Fatalf("released task must not appear in indicators")
	}
	if got, ok := byTask["task-mock"]; !ok || !got.MachineRunning || !got.ContainerRunning {
		t.Fatalf("task-mock=%v ok=%v", got, ok)
	}
}

func TestWorkspaceMachineSummaryCountsOnlyRunning(t *testing.T) {
	setupCloudTestDB(t)
	seeds := []CloudServerConfig{
		{ID: "s1", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "t-run", CommentID: "c1", Platform: "aliyun", InstanceID: "i-a", LastRuntimeStatus: "Running"},
		{ID: "s2", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "t-start", CommentID: "c2", Platform: "aliyun", InstanceID: "i-b", LastRuntimeStatus: "Starting"},
		{ID: "s3", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "t-stop", CommentID: "c3", Platform: "aliyun", InstanceID: "i-c", LastRuntimeStatus: "Stopped"},
		{ID: "s4", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "t-mock", CommentID: "c4", Platform: "mock", InstanceID: "mock-z", ServerURL: "http://x/"},
	}
	for _, s := range seeds {
		if err := upsertCloudServerConfig(s); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := computeWorkspaceMachineCounts("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	if counts.StartedCount != 2 { // Running + mock
		t.Fatalf("started=%d want 2", counts.StartedCount)
	}
	if counts.StartingCount != 1 {
		t.Fatalf("starting=%d want 1", counts.StartingCount)
	}
	if counts.BusyCount != 1 { // mock with server_url
		t.Fatalf("busy=%d want 1", counts.BusyCount)
	}
	if counts.IdleCount != 1 { // Running without server_url
		t.Fatalf("idle=%d want 1", counts.IdleCount)
	}
}
