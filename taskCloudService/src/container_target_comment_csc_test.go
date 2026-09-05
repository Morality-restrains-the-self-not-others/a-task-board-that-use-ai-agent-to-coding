package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalContainerTarget_UsesCommentCSCWhenTaskLevelEmpty(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-task-empty", "t1", "ws1", "task-fwd", "", "aliyun", "", "", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed task csc: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-live", "t1", "ws1", "task-fwd", "cmt_live", "aliyun", "i-live-1",
		"http://127.0.0.1:1/ui/cmt-tok/", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/container-target/?tenant_id=t1&workspace_id=ws1&task_id=task-fwd", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["base_url"] != "http://127.0.0.1:1" {
		t.Fatalf("base_url=%v want comment CSC origin", body["base_url"])
	}
	if body["access_token"] != "cmt-tok" {
		t.Fatalf("access_token=%v want cmt-tok", body["access_token"])
	}
}

func TestInternalContainerTarget_CommentIDSelectsMatchingCSC(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-task-empty2", "t1", "ws1", "task-two", "", "aliyun", "", "", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed task csc: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-a", "t1", "ws1", "task-two", "cmt-a", "aliyun", "i-a",
		"http://127.0.0.1:1/ui/tok-a/", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment a: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-b", "t1", "ws1", "task-two", "cmt-b", "aliyun", "i-b",
		"http://127.0.0.1:1/ui/tok-b/", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment b: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/container-target/?tenant_id=t1&workspace_id=ws1&task_id=task-two&comment_id=cmt-a", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["access_token"] != "tok-a" {
		t.Fatalf("access_token=%v want tok-a (comment a)", body["access_token"])
	}
}

func TestValidateAICommentPost_UsesCommentCSCWhenTaskLevelEmpty(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-val-empty", "t1", "ws1", "task-val", "", "aliyun", "", "", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed task csc: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-val-live", "t1", "ws1", "task-val", "cmt_val", "aliyun", "i-val",
		"http://127.0.0.1:1/ui/val-tok/", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}
	status, payload := validateAICommentPost("t1", "ws1", "task-val", map[string]interface{}{
		"content":      "hello",
		"command_kind": "shell",
	}, "")
	if status != 200 {
		t.Fatalf("status=%d payload=%v want 200 via comment CSC", status, payload)
	}
}
