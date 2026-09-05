package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStartVmAutoAuthNotInPolicyWhitelist403(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-blocked','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'["auth-allowed"]')`)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"task_id":"task-new","container_image_id":"img1","region_id":"cn-hangzhou","auto_create_security_group":true,"authorization_id":"auth-blocked"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "error" {
		t.Fatalf("status=%v", resp["status"])
	}
	msg, _ := resp["message"].(string)
	if msg == "" || !strings.Contains(msg, "机器节点") {
		t.Fatalf("message=%q", msg)
	}
}

func TestStartVmAutoDoesNotReuseIdleMachine(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                 "cfg-idle-src",
		CompanyID:          "t1",
		WorkspaceID:        "ws1",
		TaskID:             "task-idle-src",
		CommentID:          "cmt-idle",
		Platform:           "aliyun",
		InstanceID:         "i-reuse-me",
		PublicIP:           "10.0.0.9",
		Region:             "cn-hangzhou",
		ZoneID:             "cn-hangzhou-b",
		AuthorizationID:    "auth1",
		SecurityGroupID:    "sg-1",
		VswitchID:          "vsw-1",
		LastRuntimeStatus:  "Running",
		ImageInvokerUserID: "user1",
	}); err != nil {
		t.Fatal(err)
	}

	handled, err := applyStartVmPolicyGate(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil).Context(),
		"t1", "ws1", "task-new", &cloudAuthRecord{ID: "auth1"}, "cmt-new", "img1")
	if err != nil {
		t.Fatal(err)
	}
	if handled {
		t.Fatal("idle machine must not be reused; start-vm-auto should continue to cold start")
	}
	src, err := loadCloudServerConfigForComment("t1", "ws1", "task-idle-src", "cmt-idle")
	if err != nil {
		t.Fatal(err)
	}
	if src.InstanceID != "i-reuse-me" {
		t.Fatalf("source instance must stay bound, got %q", src.InstanceID)
	}
}

func TestStartVmAutoInflightAttachReturnsTraceID(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "cfg-inflight",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            "task-a",
		CommentID:         "cmt-a",
		Platform:          "aliyun",
		InstanceID:        "i-starting",
		PublicIP:          "10.0.0.8",
		Region:            "cn-hangzhou",
		AuthorizationID:   "auth1",
		LastRuntimeStatus: "Starting",
	}); err != nil {
		t.Fatal(err)
	}

	body := `{"task_id":"task-a","comment_id":"cmt-a","container_image_id":"img1","region_id":"cn-hangzhou","auto_create_security_group":true,"authorization_id":"auth1"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	req.Header.Set("X-Trace-Id", "web-start-cmt-aaa")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["reuse"] != true || resp["inflight_attach"] != true {
		t.Fatalf("want inflight attach, got %v", resp)
	}
	if resp["instance_id"] != "i-starting" {
		t.Fatalf("instance_id=%v", resp["instance_id"])
	}
	tid, _ := resp["trace_id"].(string)
	if tid == "" || tid == "task-a" {
		t.Fatalf("trace_id=%q want independent non-empty", tid)
	}
}
