package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRuntimeStatusIncludesMachineOwnerWhenContainerRunning(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-owner-a", "t1", "ws1", "task-owner", testLiveCommentID, "mock", "mock-shared-1", "http://127.0.0.1:18080/", "cn-test", "cn-test-a", "test", "Running",
	)
	if err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, platform, instance_id, server_url, region, zone_id, authorization_id, last_runtime_status, terminal_released)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-owner-b", "t1", "ws1", "task-sibling", "mock", "mock-shared-1", "", "cn-test", "cn-test-a", "test", "Running", 0,
	)
	if err != nil {
		t.Fatalf("seed sibling: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-owner&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-owner")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["container_running"] != true {
		t.Fatalf("container_running=%v want true", body["container_running"])
	}
	owners, ok := body["machine_owner_task_ids"].([]interface{})
	if !ok || len(owners) < 2 {
		t.Fatalf("machine_owner_task_ids=%v", body["machine_owner_task_ids"])
	}
	got := map[string]bool{}
	for _, o := range owners {
		got[o.(string)] = true
	}
	if !got["task-owner"] || !got["task-sibling"] {
		t.Fatalf("owners=%v", owners)
	}
	bound, ok := body["machine_bound_tasks"].([]interface{})
	if !ok || len(bound) < 2 {
		t.Fatalf("machine_bound_tasks=%v", body["machine_bound_tasks"])
	}
}

func TestServerRuntimeStatusContainerNotRunningOmitsDrivingFlag(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-idle", "t1", "ws1", "task-idle", testLiveCommentID, "mock", "mock-idle-1", "", "cn-test", "cn-test-a", "test", "Running",
	)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-idle&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-idle")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["container_running"] != false {
		t.Fatalf("container_running=%v want false", body["container_running"])
	}
	owners, _ := body["machine_owner_task_ids"].([]interface{})
	if len(owners) != 1 || owners[0].(string) != "task-idle" {
		t.Fatalf("machine_owner_task_ids=%v", body["machine_owner_task_ids"])
	}
}
