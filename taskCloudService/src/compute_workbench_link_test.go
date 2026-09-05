package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getWorkbenchLink(t *testing.T, taskID, commentID string) *httptest.ResponseRecorder {
	t.Helper()
	u := "/api/tenant/t1/workspace/ws1/cloud/compute/workbench-link/?task_id=" + taskID
	if commentID != "" {
		u += "&comment_id=" + commentID
	}
	req := httptest.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Trace-Id", "trace-wb-runtime-1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workbench-link/")
	return rec
}

// Regression：评论容器已启动、任务级 CSC 仍无 instance_id/region 时，
// workbench-link 须与 runtime-status 一样读评论级物理机，而不是 400「缺少实例ID或地域」。
func TestWorkbenchLinkUsesCommentLevelCSCWhenTaskLevelMissingInstance(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-13 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-empty', 't1', 'ws1', 'task-cmt-wb', '', 'aliyun', '', '', '', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-wb', 't1', 'ws1', 'task-cmt-wb', 'cmt-1', 'aliyun', 'i-bp-comment', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	rec := getWorkbenchLink(t, "task-cmt-wb", "cmt-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	url, _ := body["workbench_url"].(string)
	if !strings.Contains(url, "instanceId=i-bp-comment") {
		t.Fatalf("workbench_url missing comment instance: %q", url)
	}
	if !strings.Contains(url, "regionId=cn-hangzhou") {
		t.Fatalf("workbench_url missing region: %q", url)
	}
}

func TestWorkbenchLinkFillsRegionFromSiblingCSC(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-13 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-noregion', 't1', 'ws1', 'task-wb-region', '', 'aliyun', 'i-shared', '', '', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-region', 't1', 'ws1', 'task-wb-region', 'cmt-2', 'aliyun', 'i-shared', 'cn-qingdao', 'cn-qingdao-b', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	rec := getWorkbenchLink(t, "task-wb-region", "cmt-2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "regionId=cn-qingdao") {
		t.Fatalf("expected sibling region fill: %s", rec.Body.String())
	}
}

func TestWorkbenchLinkMissingInstanceAndRegion(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-13 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-empty', 't1', 'ws1', 'task-wb-empty', 'cmt-empty', 'aliyun', '', '', '', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	rec := getWorkbenchLink(t, "task-wb-empty", "cmt-empty")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "服务器配置缺少实例ID或地域") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestWorkbenchLinkUpgradesMockPlatformRealAliyunInstance(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-wb-mock', 't1', 'ws1', 'task-wb-mock', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-wb', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-wb-mock', 't1', 'ws1', 'task-wb-mock', 'cmt-wb-mock', 'mock', 'i-m5edps4oejpnwahrjpoa', '', '', '', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	rec := getWorkbenchLink(t, "task-wb-mock", "cmt-wb-mock")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "Mock") || strings.Contains(strings.ToLower(body), `"mock":true`) {
		t.Fatalf("must not reject leftover mock platform: %s", body)
	}
	if !strings.Contains(body, "instanceId=i-m5edps4oejpnwahrjpoa") {
		t.Fatalf("workbench_url missing real instance: %s", body)
	}
	if !strings.Contains(body, "regionId=cn-qingdao") {
		t.Fatalf("workbench_url missing healed region: %s", body)
	}
}

func TestWorkbenchLinkCommentIdSelectsAmongTwoComments(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-13 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-task-two', 't1', 'ws1', 'task-two-cmt', '', 'aliyun', 'i-task', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-a', 't1', 'ws1', 'task-two-cmt', 'cmt-a', 'aliyun', 'i-comment-a', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-b', 't1', 'ws1', 'task-two-cmt', 'cmt-b', 'aliyun', 'i-comment-b', 'cn-qingdao', 'cn-qingdao-b', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workbench-link/?task_id=task-two-cmt&comment_id=cmt-b", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workbench-link/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "instanceId=i-comment-b") {
		t.Fatalf("want comment-b instance, got %s", body)
	}
	if strings.Contains(body, "instanceId=i-task") || strings.Contains(body, "instanceId=i-comment-a") {
		t.Fatalf("must not use task/other comment instance: %s", body)
	}
}
