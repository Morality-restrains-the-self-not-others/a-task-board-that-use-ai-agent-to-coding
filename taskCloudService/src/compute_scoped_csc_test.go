package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentIDFromComputeRequestPrefersPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/workbench-link/tenant_id/t1/workspace_id/w1/task_id/task1/comment_id/cmt-path/?comment_id=cmt-q",
		nil)
	if got := commentIDFromComputeRequest(req, map[string]interface{}{"comment_id": "cmt-body"}); got != "cmt-path" {
		t.Fatalf("got %q want cmt-path", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/x/?comment_id=cmt-q", nil)
	if got := commentIDFromComputeRequest(req, nil); got != "cmt-q" {
		t.Fatalf("got %q want cmt-q", got)
	}
}

func seedTaskAndCommentCSC(t *testing.T, taskID, commentID, taskInst, commentInst string) {
	t.Helper()
	now := "2026-08-13 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-task-"+taskID, "t1", "ws1", taskID, "", "aliyun", taskInst, "cn-hangzhou", "cn-hangzhou-i", "auth1", now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-"+commentID, "t1", "ws1", taskID, commentID, "aliyun", commentInst, "cn-hangzhou", "cn-hangzhou-i", "auth1", now, now)
	if err != nil {
		t.Fatal(err)
	}
}

// 生产回归：评论 CSC 先建成 mock（空授权/地域），任务级已是 aliyun；
// 读路径必须升级元数据，否则 Workbench 报「平台 mock」、runtime-status 走未授权降级。
func TestResolveScopedUpgradesMockCommentFromTaskBase(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-heal', 't1', 'ws1', 'task-heal', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-heal', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, public_ip, server_url, created_at, updated_at)
		VALUES ('cfg-cmt-heal', 't1', 'ws1', 'task-heal', 'cmt-heal', 'mock', 'i-m5edps4oejpnwahrjpoa', '', '', '', '47.105.78.163', 'http://47.105.78.163:8080', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := resolveScopedCloudServerConfig("t1", "ws1", "task-heal", "cmt-heal")
	if err != nil || cfg == nil {
		t.Fatalf("err=%v cfg=%v", err, cfg)
	}
	if cfg.Platform != "aliyun" || cfg.Region != "cn-qingdao" || cfg.AuthorizationID != "cpa-heal" {
		t.Fatalf("healed meta platform=%s region=%s auth=%s", cfg.Platform, cfg.Region, cfg.AuthorizationID)
	}
	if cfg.InstanceID != "i-m5edps4oejpnwahrjpoa" {
		t.Fatalf("must keep real instance, got %q", cfg.InstanceID)
	}
	reloaded, err := loadCloudServerConfigForComment("t1", "ws1", "task-heal", "cmt-heal")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Platform != "aliyun" || reloaded.AuthorizationID != "cpa-heal" {
		t.Fatalf("upgrade must persist platform=%s auth=%s", reloaded.Platform, reloaded.AuthorizationID)
	}
}

func TestResolveScopedPreservesCommentOverrideAndFillsEmptyRegion(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-ov', 't1', 'ws1', 'task-ov', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-tpl', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-ov', 't1', 'ws1', 'task-ov', 'cmt-ov', 'aliyun', 'i-user', '', '', 'cpa-user', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := resolveScopedCloudServerConfig("t1", "ws1", "task-ov", "cmt-ov")
	if err != nil || cfg == nil {
		t.Fatalf("err=%v cfg=%v", err, cfg)
	}
	if cfg.AuthorizationID != "cpa-user" {
		t.Fatalf("must keep comment auth, got %q", cfg.AuthorizationID)
	}
	if cfg.Region != "cn-qingdao" {
		t.Fatalf("empty region should fill from template, got %q", cfg.Region)
	}
	if cfg.InstanceID != "i-user" {
		t.Fatalf("instance=%q", cfg.InstanceID)
	}
}

func TestResolveScopedCloudServerConfigRequiresCommentID(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskAndCommentCSC(t, "task-scope", "cmt-x", "mock-task", "mock-cmt")
	cfg, err := resolveScopedCloudServerConfig("t1", "ws1", "task-scope", "cmt-x")
	if err != nil || cfg == nil {
		t.Fatalf("err=%v cfg=%v", err, cfg)
	}
	if cfg.InstanceID != "mock-cmt" {
		t.Fatalf("instance_id=%q want mock-cmt", cfg.InstanceID)
	}
	_, err = resolveScopedCloudServerConfig("t1", "ws1", "task-scope", "")
	if !errors.Is(err, errCommentIDRequired) {
		t.Fatalf("empty comment_id want errCommentIDRequired, got %v", err)
	}
}

func TestServerRuntimeStatusCommentIdSelectsCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskAndCommentCSC(t, "task-rt", "cmt-rt", "mock-task-rt", "mock-cmt-rt")
	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-rt&comment_id=cmt-rt", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-rt")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["instance_id"] != "mock-cmt-rt" {
		t.Fatalf("instance_id=%v want mock-cmt-rt body=%s", body["instance_id"], rec.Body.String())
	}
}

func TestServerRuntimeStatusWithoutCommentIdRejected(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskAndCommentCSC(t, "task-rt2", "cmt-rt2", "mock-task-rt2", "mock-cmt-rt2")
	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-rt2", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-rt2")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "缺少评论ID") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestStopVmNativeCommentIdDoesNotStopTaskLevelWhenCommentHasNoInstance(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-13 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-stop', 't1', 'ws1', 'task-stop-cmt', '', 'aliyun', 'i-task-live', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-stop', 't1', 'ws1', 'task-stop-cmt', 'cmt-empty', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/stop-vm/",
		strings.NewReader(`{"task_id":"task-stop-cmt","comment_id":"cmt-empty"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/stop-vm/")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("must not stop task-level instance; status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "未提供实例ID") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestStopVmNativeWithoutCommentIdRejected(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/stop-vm/",
		strings.NewReader(`{"task_id":"task-stop-cmt"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/stop-vm/")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "缺少评论ID") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
