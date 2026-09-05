package main

import (
	"strings"
	"testing"
)

func TestPersistCommentCSCStartFailureWritesFailedReason(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-csc-fail"
	workspaceID := "ws-csc-fail"
	taskID := "taskCscFail"
	commentID := "cmtCscFail"
	cscID := "csc-fail-1"
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		cscID, companyID, workspaceID, taskID, commentID, "aliyun", "", "cn-test", "cn-test-a", "test", machineRuntimeStarting,
	)
	if err != nil {
		t.Fatalf("seed csc: %v", err)
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", workspaceID)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	b.CSCID = cscID
	raw := "SDKError:\n   Code: InvalidAccountStatus.NotEnoughBalance\n   Message: Your account does not have enough balance"
	persistCommentCSCStartFailure(b, raw)

	cfg, err := loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		t.Fatalf("load csc: %v", err)
	}
	if cfg.LastRuntimeStatus != machineRuntimeFailed {
		t.Fatalf("last_runtime_status=%q want Failed", cfg.LastRuntimeStatus)
	}
	if !strings.Contains(cfg.ErrorReason, "余额不足") {
		t.Fatalf("error_reason=%q want 余额不足", cfg.ErrorReason)
	}
}

func TestPersistCommentCSCStartFailureResolvesCSCWhenBindingIDEmpty(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-csc-empty"
	workspaceID := "ws-csc-empty"
	taskID := "taskCscEmpty"
	commentID := "cmtCscEmpty"
	cscID := "csc-empty-1"
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		cscID, companyID, workspaceID, taskID, commentID, "aliyun", "", "cn-test", "cn-test-a", "test", machineRuntimeStarting,
	)
	if err != nil {
		t.Fatalf("seed csc: %v", err)
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", workspaceID)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	persistCommentCSCStartFailure(b, "InvalidAccountStatus.NotEnoughBalance")
	if b.CSCID != cscID {
		t.Fatalf("binding.CSCID=%q want %q", b.CSCID, cscID)
	}
	cfg, err := loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		t.Fatalf("load csc: %v", err)
	}
	if cfg.LastRuntimeStatus != machineRuntimeFailed {
		t.Fatalf("last_runtime_status=%q want Failed", cfg.LastRuntimeStatus)
	}
}
