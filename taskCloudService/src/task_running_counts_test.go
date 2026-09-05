package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedCommentCSC(t *testing.T, id, taskID, commentID, instanceID, status, serverURL string) {
	t.Helper()
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: id, CompanyID: "t1", WorkspaceID: "ws1", TaskID: taskID, CommentID: commentID,
		Platform: "aliyun", InstanceID: instanceID, LastRuntimeStatus: status,
		ServerURL: serverURL, Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
}

func seedTaskTemplate(t *testing.T, taskID string) {
	t.Helper()
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "tpl-" + taskID, CompanyID: "t1", WorkspaceID: "ws1", TaskID: taskID,
		Platform: "aliyun", Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRecomputeTaskRunningCountsPredicates(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskTemplate(t, "task-cnt")
	seedCommentCSC(t, "c1", "task-cnt", "cmt-a", "i-a", "Running", "")
	seedCommentCSC(t, "c2", "task-cnt", "cmt-b", "i-b", "Starting", "")
	recomputeTaskRunningCounts("t1", "ws1", "task-cnt")
	got := loadTaskRunningCounts("t1", "ws1", "task-cnt")
	if got.Machines != 1 || got.Containers != 0 {
		t.Fatalf("starting ignored: %+v want machines=1 containers=0", got)
	}

	seedCommentCSC(t, "c1", "task-cnt", "cmt-a", "i-a", "Running", "http://10.0.0.1:8080/")
	recomputeTaskRunningCounts("t1", "ws1", "task-cnt")
	got = loadTaskRunningCounts("t1", "ws1", "task-cnt")
	if got.Machines != 1 || got.Containers != 1 {
		t.Fatalf("url added: %+v want 1/1", got)
	}

	seedCommentCSC(t, "c3", "task-cnt", "cmt-c", "i-c", "Running", "http://10.0.0.2:8080/")
	recomputeTaskRunningCounts("t1", "ws1", "task-cnt")
	got = loadTaskRunningCounts("t1", "ws1", "task-cnt")
	if got.Machines != 2 || got.Containers != 2 {
		t.Fatalf("two running: %+v want 2/2", got)
	}

	seedCommentCSC(t, "c3", "task-cnt", "cmt-c", "", "Released", "")
	recomputeTaskRunningCounts("t1", "ws1", "task-cnt")
	got = loadTaskRunningCounts("t1", "ws1", "task-cnt")
	if got.Machines != 1 || got.Containers != 1 {
		t.Fatalf("released: %+v want 1/1", got)
	}
}

func TestPersistWithoutCommentDoesNotWriteTaskInstance(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskTemplate(t, "task-persist-empty")
	if err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id": "task-persist-empty",
	}, "i-should-not-land", "req-x"); err == nil {
		t.Fatal("unscoped persist must error")
	}
	cfg, err := loadCloudServerConfig("t1", "ws1", "task-persist-empty")
	if err != nil || cfg == nil {
		t.Fatalf("load template: %v cfg=%v", err, cfg)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("task-level instance_id=%q want empty", cfg.InstanceID)
	}
}

func TestSetLastRuntimeStatusDoesNotBrushWholeTask(t *testing.T) {
	setupCloudTestDB(t)
	seedCommentCSC(t, "keep", "task-brush", "cmt-keep", "i-keep", "Running", "")
	seedCommentCSC(t, "chg", "task-brush", "cmt-chg", "i-chg", "Running", "")
	if err := setCloudServerLastRuntimeStatus("t1", "ws1", "task-brush", "cmt-chg", "i-chg", "Stopped"); err != nil {
		t.Fatal(err)
	}
	keep, _ := loadCloudServerConfigForComment("t1", "ws1", "task-brush", "cmt-keep")
	chg, _ := loadCloudServerConfigForComment("t1", "ws1", "task-brush", "cmt-chg")
	if keep == nil || chg == nil {
		t.Fatal("load comments")
	}
	if keep.LastRuntimeStatus != "Running" {
		t.Fatalf("other comment status=%q", keep.LastRuntimeStatus)
	}
	if chg.LastRuntimeStatus != "Stopped" {
		t.Fatalf("target status=%q", chg.LastRuntimeStatus)
	}
}

func TestServerRuntimeStatusDoesNotHealTaskLevelInstance(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-task-noheal', 't1', 'ws1', 'task-noheal', '', 'aliyun', 'mock-noheal-1', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-cmt-noheal', 't1', 'ws1', 'task-noheal', 'cmt_noheal', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-noheal&comment_id=cmt_noheal", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-noheal")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["instance_id"] != nil {
		t.Fatalf("must not heal task-level instance: body=%v", body)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-noheal", "cmt_noheal")
	if err != nil || cfg == nil {
		t.Fatal(err)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("comment CSC healed: instance_id=%q", cfg.InstanceID)
	}
}

func TestIndicatorsIgnoreTemplateInstanceAndExposeCounts(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "tpl-ind", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-ind",
		Platform: "aliyun", InstanceID: "i-template-should-ignore", LastRuntimeStatus: "Running",
		Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	seedCommentCSC(t, "c-ind", "task-ind", "cmt-ind", "i-cmt", "Running", "http://10.0.0.9:8080/")
	recomputeTaskRunningCounts("t1", "ws1", "task-ind")

	items, err := listWorkspaceRuntimeIndicators("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	var found *workspaceRuntimeIndicator
	for i := range items {
		if items[i].TaskID == "task-ind" {
			found = &items[i]
		}
	}
	if found == nil {
		t.Fatal("task-ind missing")
	}
	if !found.MachineRunning || !found.ContainerRunning {
		t.Fatalf("flags=%+v", found)
	}
	if found.RunningMachineCount != 1 || found.RunningContainerCount != 1 {
		t.Fatalf("counts=%+v want 1/1", found)
	}
}

func TestImportWithoutCommentDoesNotWriteTaskInstance(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskTemplate(t, "task-imp-empty")
	count, err := importCloudServerConfigs([]CloudServerConfig{{
		ID: "tpl-task-imp-empty", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-imp-empty",
		Platform: "aliyun", InstanceID: "i-must-not-land", LastRuntimeStatus: "Running",
		Region: "cn-test", ZoneID: "a",
	}})
	if err != nil || count != 1 {
		t.Fatalf("import: count=%d err=%v", count, err)
	}
	cfg, err := loadCloudServerConfig("t1", "ws1", "task-imp-empty")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v cfg=%v", err, cfg)
	}
	if cfg.InstanceID != "" || cfg.LastRuntimeStatus != "" {
		t.Fatalf("task-level runtime leaked: instance=%q status=%q", cfg.InstanceID, cfg.LastRuntimeStatus)
	}
}

func TestBuildContainerTaskUIContextExposesTaskCounts(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskTemplate(t, "task-ui-cnt")
	seedCommentCSC(t, "c-ui", "task-ui-cnt", "cmt-ui", "i-ui", "Running", "http://10.0.0.8:8080/")
	recomputeTaskRunningCounts("t1", "ws1", "task-ui-cnt")
	payload := buildContainerTaskUIContext("t1", "ws1", "task-ui-cnt")
	if payload["running_machine_count"] != 1 || payload["running_container_count"] != 1 {
		t.Fatalf("ui-context counts=%v", payload)
	}
	if payload["has_server_config"] != false {
		t.Fatalf("empty comment must not expose comment CSC: %v", payload)
	}
}
