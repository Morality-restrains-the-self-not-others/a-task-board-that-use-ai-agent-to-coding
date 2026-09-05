package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWorkspaceMachineSummaryBusyAndIdleCounts(t *testing.T) {
	setupCloudTestDB(t)

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-busy",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-busy",
		CommentID:   "cmt-busy",
		Platform:    "mock",
		InstanceID:  "mock-busy",
		ServerURL:   "http://127.0.0.1:8080/",
		Region:      "cn-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-idle",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-idle",
		CommentID:   "cmt-idle",
		Platform:    "mock",
		InstanceID:  "mock-idle",
		PublicIP:    "1.2.3.4",
		Region:      "cn-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfigHistory(CloudServerConfigHistory{
		ID:          "hist-idle",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-idle",
		Platform:    "mock",
		InstanceID:  "mock-idle",
		StartedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',45,'[]')`)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-summary/", nil)
	req.Header.Set("X-User-Id", "test-user")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-machine-summary/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "success" {
		t.Fatalf("status=%v", body["status"])
	}
	if body["started_count"] != float64(2) {
		t.Fatalf("started_count=%v want 2", body["started_count"])
	}
	if body["busy_count"] != float64(1) {
		t.Fatalf("busy_count=%v want 1", body["busy_count"])
	}
	if body["idle_count"] != float64(1) {
		t.Fatalf("idle_count=%v want 1", body["idle_count"])
	}
	if body["idle_recycle_minutes"] != float64(45) {
		t.Fatalf("idle_recycle_minutes=%v want 45", body["idle_recycle_minutes"])
	}
}

func TestWorkspaceMachineSummaryCoResidentInstanceCounts(t *testing.T) {
	setupCloudTestDB(t)

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-busy-shared",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-busy",
		CommentID:   "cmt-busy-shared",
		Platform:    "mock",
		InstanceID:  "mock-shared",
		ServerURL:   "http://127.0.0.1:8080/",
		Region:      "cn-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:          "cfg-idle-shared",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-idle",
		CommentID:   "cmt-idle-shared",
		Platform:    "mock",
		InstanceID:  "mock-shared",
		PublicIP:    "1.2.3.4",
		Region:      "cn-test",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',45,'[]')`)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-summary/", nil)
	req.Header.Set("X-User-Id", "test-user")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-machine-summary/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["started_count"] != float64(1) {
		t.Fatalf("started_count=%v want 1 (unique instance)", body["started_count"])
	}
	if body["busy_count"] != float64(1) {
		t.Fatalf("busy_count=%v want 1", body["busy_count"])
	}
	if body["idle_count"] != float64(0) {
		t.Fatalf("idle_count=%v want 0 (instance has busy container)", body["idle_count"])
	}
}
