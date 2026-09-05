package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPersistStartVmInstanceBindingFillsCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-cmt-empty', 't1', 'ws1', 'task-persist', 'cmt_p', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	if err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id":    "task-persist",
		"comment_id": "cmt_p",
		"csc_id":     "csc-cmt-empty",
	}, "i-aliyun-1", "req-1"); err != nil {
		t.Fatalf("persist: %v", err)
	}

	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-persist", "cmt_p")
	if err != nil || cfg == nil {
		t.Fatalf("load comment csc: err=%v cfg=%v", err, cfg)
	}
	if cfg.InstanceID != "i-aliyun-1" {
		t.Fatalf("comment instance_id=%q want i-aliyun-1", cfg.InstanceID)
	}
	if cfg.LastRuntimeStatus != machineRuntimeStarting {
		t.Fatalf("last_runtime_status=%q want Starting", cfg.LastRuntimeStatus)
	}
}

func TestInternalLookupConfigByCommentID(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskAndCommentCSC(t, "task-lu", "cmt-lu", "i-task", "i-cmt")

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/lookup/?tenant_id=t1&workspace_id=ws1&task_id=task-lu&comment_id=cmt-lu", nil)
	rec := httptest.NewRecorder()
	handleInternalLookupConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["instance_id"] != "i-cmt" {
		t.Fatalf("lookup must return comment CSC, instance_id=%v", out["instance_id"])
	}
	if out["comment_id"] != "cmt-lu" {
		t.Fatalf("comment_id=%v", out["comment_id"])
	}
}

func TestImportCloudServerConfigsWritesCommentID(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-imp-cmt', 't1', 'ws1', 'task-imp', 'cmt_imp', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	count, err := importCloudServerConfigs([]CloudServerConfig{{
		ID: "csc-imp-cmt", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-imp",
		CommentID: "cmt_imp", Platform: "aliyun", InstanceID: "i-imported",
		Region: "cn-hangzhou", ZoneID: "cn-hangzhou-i", AuthorizationID: "auth1",
	}})
	if err != nil || count != 1 {
		t.Fatalf("import: count=%d err=%v", count, err)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-imp", "cmt_imp")
	if err != nil || cfg == nil {
		t.Fatalf("load: err=%v", err)
	}
	if cfg.InstanceID != "i-imported" {
		t.Fatalf("instance_id=%q want i-imported", cfg.InstanceID)
	}
	if cfg.CommentID != "cmt_imp" {
		t.Fatalf("comment_id=%q", cfg.CommentID)
	}
}

func TestServerRuntimeStatusDoesNotStealOtherCommentInstance(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-task-steal', 't1', 'ws1', 'task-steal', '', 'aliyun', 'mock-owned-1', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-owner', 't1', 'ws1', 'task-steal', 'cmt_owner', 'aliyun', 'mock-owned-1', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-empty', 't1', 'ws1', 'task-steal', 'cmt_empty', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-steal&comment_id=cmt_empty", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-steal")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["instance_id"] != nil {
		t.Fatalf("must not steal other comment instance: body=%v", body)
	}
	if body["runtime_status"] != "Starting" {
		t.Fatalf("runtime_status=%v want Starting", body["runtime_status"])
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-steal", "cmt_empty")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("empty comment CSC mutated: instance_id=%q", cfg.InstanceID)
	}
}

func TestServerRuntimeStatusDoesNotHealReleasedCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-task-rel', 't1', 'ws1', 'task-rel', '', 'aliyun', 'mock-rel-1', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status, created_at, updated_at)
		VALUES ('csc-cmt-rel', 't1', 'ws1', 'task-rel', 'cmt_rel', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', 'Released', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-rel&comment_id=cmt_rel", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-rel")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] != "Released" {
		t.Fatalf("released comment must stay Released: body=%v", body)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-rel", "cmt_rel")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("released comment CSC healed: instance_id=%q", cfg.InstanceID)
	}
}
