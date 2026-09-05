package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStopVmNativeMissingTaskID(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/stop-vm/",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/stop-vm/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "缺少任务ID") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestStopVmNativeNoServerConfig(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/stop-vm/",
		strings.NewReader(`{"task_id":"task-missing","comment_id":"cmt-missing"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/stop-vm/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "未找到任务对应的服务器配置") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestStopVmNativeSuccessClearsConfig(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	now := "2026-07-08 00:00:00"
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES('csc1','t1','ws1','task-stop','cmt-stop','aliyun','i-stop-test','cn-hangzhou','auth1',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	cfg.KafkaBootstrapServers = ""

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/stop-vm/",
		strings.NewReader(`{"task_id":"task-stop","comment_id":"cmt-stop"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/stop-vm/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "虚拟机停止请求已提交") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "not yet ported") {
		t.Fatal("unexpected 501")
	}

	cfgAfter, err := loadCloudServerConfigForComment("t1", "ws1", "task-stop", "cmt-stop")
	if err != nil {
		t.Fatal(err)
	}
	if cfgAfter.InstanceID != "i-stop-test" {
		t.Fatalf("async stop should keep instance until consumer clears, got %q", cfgAfter.InstanceID)
	}
}

func TestStopVmNativeMockPlatformSucceedsWithoutAuth(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-07-08 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,created_at,updated_at)
		VALUES('csc-mock','t1','ws1','task-mock-stop','cmt-mock-stop','mock','mock-e2e-stop','cn-hongkong',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	cfg.KafkaBootstrapServers = ""

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/stop-vm/",
		strings.NewReader(`{"task_id":"task-mock-stop","comment_id":"cmt-mock-stop"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/stop-vm/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "虚拟机停止请求已提交") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "尚未在 Go 服务中实现") {
		t.Fatal("mock platform must not 501")
	}
}
