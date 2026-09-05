package main

import (
	"context"
	"strings"
	"testing"
)

func TestEmptyInstanceRuntimeDecisionPrefersReleasedOverCommentCSC(t *testing.T) {
	cfg := &CloudServerConfig{CommentID: "cmt_1", LastRuntimeStatus: machineRuntimeReleased}
	status, msg, failed := emptyInstanceRuntimeDecision(cfg, "t1", "ws1", "task-1")
	if status != machineRuntimeReleased {
		t.Fatalf("status=%v want Released", status)
	}
	if failed {
		t.Fatal("released CSC is not a start failure")
	}
	if msg == "云实例创建中，等待分配" {
		t.Fatalf("released CSC must not use starting message: %q", msg)
	}
}

func TestEmptyInstanceRuntimeDecisionStartingWhenCommentCSCProvisioning(t *testing.T) {
	cfg := &CloudServerConfig{CommentID: "cmt_1", LastRuntimeStatus: ""}
	status, msg, failed := emptyInstanceRuntimeDecision(cfg, "t1", "ws1", "task-1")
	if failed {
		t.Fatal("in-progress comment CSC must not be start_failed")
	}
	if status != machineRuntimeStarting {
		t.Fatalf("status=%v want Starting", status)
	}
	if msg != "云实例创建中，等待分配" {
		t.Fatalf("message=%q", msg)
	}
}

func TestEmptyInstanceRuntimeDecisionStartFailedWhenBindingFailed(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-rt-bal"
	workspaceID := "ws-rt-bal"
	taskID := "taskRtBal"
	commentID := "cmtRtBal"
	cfg := &CloudServerConfig{
		ID: "csc-rt-bal", CompanyID: companyID, WorkspaceID: workspaceID,
		TaskID: taskID, CommentID: commentID, LastRuntimeStatus: machineRuntimeStarting,
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", workspaceID)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, cfg.ID); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	status, msg, failed := emptyInstanceRuntimeDecision(cfg, companyID, workspaceID, taskID)
	if !failed {
		t.Fatal("failed binding with empty instance must set start_failed")
	}
	if status == machineRuntimeStarting {
		t.Fatalf("must not stay Starting after start-vm failure: status=%v msg=%q", status, msg)
	}
	if !strings.Contains(msg, "启动失败") && !strings.Contains(msg, "尚未分配") {
		t.Fatalf("message=%q want start-failure copy", msg)
	}
}

func TestEmptyInstanceRuntimeDecisionStartFailedUsesAliyunBalanceLog(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-rt-sdk"
	workspaceID := "ws-rt-sdk"
	taskID := "taskRtSdk"
	commentID := "cmtRtSdk"
	errMsg := "SDKError:\n   StatusCode: 403\n   Code: InvalidAccountStatus.NotEnoughBalance\n   Message: Your account does not have enough balance"
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", workspaceID)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	persistServerSchedulingLogForBinding(b, startVmErrorStatusData(companyID, workspaceID, errMsg))
	cfg := &CloudServerConfig{
		CommentID: commentID, LastRuntimeStatus: "", CompanyID: companyID, WorkspaceID: workspaceID, TaskID: taskID,
	}
	status, msg, failed := emptyInstanceRuntimeDecision(cfg, companyID, workspaceID, taskID)
	if !failed {
		t.Fatal("NotEnoughBalance log must mark start_failed")
	}
	if status == machineRuntimeStarting {
		t.Fatalf("must not report Starting after NotEnoughBalance: %v %q", status, msg)
	}
	if !strings.Contains(msg, "余额不足") {
		t.Fatalf("message=%q want humanized 余额不足", msg)
	}
}

func TestMissingCommentCSCRuntimeMessageUsesStartErrorLog(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-rt-err"
	workspaceID := "ws-rt-err"
	taskID := "taskRtErr"
	commentID := "cmtRtErr"
	errMsg := "调用镜像市场 API 失败: 镜像服务返回错误: 502"

	if err := publishTaskSSE(context.Background(), taskID, commentID, map[string]interface{}{
		"status":       "error",
		"message":      errMsg,
		"progress":     0,
		"event_name":   "server_status_update",
		"tenant_id":    companyID,
		"workspace_id": workspaceID,
		"trace_id":     "6abd23eb9234860f8c5f26a7",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	msg := missingCommentCSCRuntimeMessage(companyID, workspaceID, taskID, commentID)
	if !strings.Contains(msg, "镜像服务返回错误: 502") {
		t.Fatalf("message=%q want actual start-vm error", msg)
	}
	if strings.Contains(msg, "启动已发起") {
		t.Fatalf("in-progress copy must not hide start-vm error: %q", msg)
	}
	_, startFailed := missingCommentCSCRuntime(companyID, workspaceID, taskID, commentID)
	if !startFailed {
		t.Fatal("startFailed must be true so FE can label 启动失败 without snippet matching")
	}
	if got := loadBindingStartTraceID(taskID, commentID); got != "6abd23eb9234860f8c5f26a7" {
		t.Fatalf("start_trace_id=%q want preserved for diagnosis", got)
	}
}
