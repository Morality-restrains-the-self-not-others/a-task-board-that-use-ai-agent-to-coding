package main

import (
	"testing"
)

func TestEnsureCommentBindingForStartVM_InsertsWhenMissing(t *testing.T) {
	setupCloudTestDB(t)
	if err := ensureCommentBindingForStartVM("t-scope", "ws-scope", "task-scope", "cmt-scope", ""); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	b, err := loadCommentContainerBinding("t-scope", "task-scope", "cmt-scope")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if b.ExecutionMode != ccbExecutionWaitPrevious {
		t.Fatalf("mode=%s want wait_previous", b.ExecutionMode)
	}
	if b.Status != ccbStatusPending {
		t.Fatalf("status=%s want pending", b.Status)
	}
	if b.WorkspaceID != "ws-scope" {
		t.Fatalf("workspace=%s", b.WorkspaceID)
	}
}

func TestEnsureCommentBindingForStartVM_DoesNotFlipExistingMode(t *testing.T) {
	setupCloudTestDB(t)
	if _, err := insertCommentContainerBinding("t-scope2", "task-scope2", "cmt-scope2", ccbExecutionIndependent, "", "ws-scope2"); err != nil {
		t.Fatal(err)
	}
	if err := ensureCommentBindingForStartVM("t-scope2", "ws-scope2", "task-scope2", "cmt-scope2", ccbExecutionWaitPrevious); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	b, err := loadCommentContainerBinding("t-scope2", "task-scope2", "cmt-scope2")
	if err != nil {
		t.Fatal(err)
	}
	if b.ExecutionMode != ccbExecutionIndependent {
		t.Fatalf("mode=%s want independent preserved", b.ExecutionMode)
	}
}

func TestFillStartVmEventCommentCSCID_UsesExistingCommentRow(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-29 01:42:47"
	if _, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc_scope_cmt', 't-fill', 'ws-fill', 'task-fill', 'cmt-fill', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-1', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	eventData := map[string]interface{}{"task_id": "task-fill", "comment_id": "cmt-fill"}
	fillStartVmEventCommentCSCID("t-fill", "ws-fill", "task-fill", "cmt-fill", eventData)
	if got := strField(eventData, "csc_id"); got != "csc_scope_cmt" {
		t.Fatalf("csc_id=%q want csc_scope_cmt", got)
	}
}

func TestFillStartVmEventCommentCSCID_SkipsWhenAlreadySet(t *testing.T) {
	setupCloudTestDB(t)
	eventData := map[string]interface{}{"csc_id": "csc_preset"}
	fillStartVmEventCommentCSCID("t-none", "ws-none", "task-none", "cmt-none", eventData)
	if got := strField(eventData, "csc_id"); got != "csc_preset" {
		t.Fatalf("csc_id=%q must keep preset", got)
	}
}

func TestEnsureCommentBindingForStartVM_NoCommentIsNoop(t *testing.T) {
	setupCloudTestDB(t)
	if err := ensureCommentBindingForStartVM("t1", "ws1", "task1", "", ""); err != nil {
		t.Fatalf("empty comment must noop: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_comment_container_bindings WHERE task_id='task1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("must not insert binding without comment_id, n=%d", n)
	}
}
