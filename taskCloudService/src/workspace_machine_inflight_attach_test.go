package main

import (
	"database/sql"
	"errors"
	"testing"
)

func TestTryAttachSameTaskInFlightMachineStarting(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "cfg-inflight",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            "task-a",
		CommentID:         "cmt-a",
		Platform:          "aliyun",
		InstanceID:        "i-starting",
		PublicIP:          "1.2.3.4",
		LastRuntimeStatus: "Starting",
		AuthorizationID:   "auth1",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-a", "cmt-a", "")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Reused || got.InstanceID != "i-starting" || got.PublicIP != "1.2.3.4" {
		t.Fatalf("got=%+v", got)
	}
}

func TestTryAttachSameTaskInFlightMachineSkipsEmpty(t *testing.T) {
	setupCloudTestDB(t)
	got, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-missing", "cmt-missing", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Reused {
		t.Fatalf("unexpected reuse: %+v", got)
	}
}

func TestTryAttachSameTaskInFlightMachineDoesNotHealTaskLevel(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "cfg-task",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            "task-a",
		Platform:          "aliyun",
		InstanceID:        "i-task-level",
		PublicIP:          "1.2.3.4",
		Region:            "cn-hangzhou",
		LastRuntimeStatus: "Starting",
		AuthorizationID:   "auth1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:              "cfg-comment",
		CompanyID:       "t1",
		WorkspaceID:     "ws1",
		TaskID:          "task-a",
		CommentID:       "c1",
		Platform:        "aliyun",
		AuthorizationID: "auth1",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-a", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Reused {
		t.Fatalf("must not heal task-level instance: %+v", got)
	}
	commentCfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-a", "c1")
	if err != nil {
		t.Fatalf("load comment csc: %v", err)
	}
	if trim(commentCfg.InstanceID) != "" {
		t.Fatalf("comment csc instance_id=%q want empty", commentCfg.InstanceID)
	}
}

func TestTryAttachSameTaskInFlightMachineReusesOwnComment(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "cfg-comment",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            "task-a",
		CommentID:         "c1",
		Platform:          "aliyun",
		InstanceID:        "i-own",
		PublicIP:          "9.9.9.9",
		LastRuntimeStatus: "Starting",
		AuthorizationID:   "auth1",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-a", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Reused || got.InstanceID != "i-own" || got.PublicIP != "9.9.9.9" {
		t.Fatalf("got=%+v", got)
	}
	again, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-a", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !again.Reused || again.InstanceID != "i-own" {
		t.Fatalf("second attach=%+v", again)
	}
}

func TestTryAttachHealsMockPlatformFromTemplate(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status, public_ip, created_at, updated_at)
		VALUES ('cfg-tpl-att', 't1', 'ws1', 'task-heal-att', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-att', '', '', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status, public_ip, created_at, updated_at)
		VALUES ('cfg-cmt-att', 't1', 'ws1', 'task-heal-att', 'c-att', 'mock', 'i-attaching', '', '', '', 'Starting', '1.2.3.4', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-heal-att", "c-att", "")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Reused || got.InstanceID != "i-attaching" {
		t.Fatalf("attach=%+v", got)
	}
	reloaded, err := loadCloudServerConfigForComment("t1", "ws1", "task-heal-att", "c-att")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Platform != "aliyun" || reloaded.Region != "cn-qingdao" || reloaded.AuthorizationID != "cpa-att" {
		t.Fatalf("healed platform=%s region=%s auth=%s", reloaded.Platform, reloaded.Region, reloaded.AuthorizationID)
	}
	if reloaded.InstanceID != "i-attaching" {
		t.Fatalf("instance=%q", reloaded.InstanceID)
	}
}

func TestTryAttachSameTaskInFlightMachineDoesNotCreateFromTaskLevel(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "cfg-task",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            "task-b",
		Platform:          "aliyun",
		InstanceID:        "i-running",
		PublicIP:          "5.6.7.8",
		LastRuntimeStatus: "Running",
		AuthorizationID:   "auth1",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := tryAttachSameTaskInFlightMachine("t1", "ws1", "task-b", "c-new", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Reused {
		t.Fatalf("must not reuse task-level instance: %+v", got)
	}
	commentCfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-b", "c-new")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if commentCfg != nil && trim(commentCfg.InstanceID) != "" {
		t.Fatalf("must not create comment CSC from task-level: %+v", commentCfg)
	}
}
