package main

import "testing"

func TestMarkCommentContainerBindingFailedPreservesExistingCSCIDWhenEmpty(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-fail-keep"
	workspaceID := "ws-fail-keep"
	taskID := "taskFailKeep"
	commentID := "cmtFailKeep"
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", workspaceID)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	wantCSC := "csc-keep-1"
	if _, err := db.Exec(`UPDATE cloud_comment_container_bindings SET csc_id=? WHERE id=?`, wantCSC, b.ID); err != nil {
		t.Fatalf("seed csc_id: %v", err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, ""); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	loaded, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || loaded == nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Status != ccbStatusFailed {
		t.Fatalf("status=%q want failed", loaded.Status)
	}
	if loaded.CSCID != wantCSC {
		t.Fatalf("csc_id=%q want preserved %q", loaded.CSCID, wantCSC)
	}
}

func TestMarkCommentBindingFailedAfterStartErrorAttachesCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-fail-attach"
	workspaceID := "ws-fail-attach"
	taskID := "taskFailAttach"
	commentID := "cmtFailAttach"
	cscID := "csc-attach-1"
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
	markCommentBindingFailedAfterStartError(b)
	if b.CSCID != cscID {
		t.Fatalf("in-memory csc_id=%q want %q", b.CSCID, cscID)
	}
	loaded, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || loaded == nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.CSCID != cscID {
		t.Fatalf("stored csc_id=%q want %q", loaded.CSCID, cscID)
	}
	if loaded.Status != ccbStatusFailed {
		t.Fatalf("status=%q want failed", loaded.Status)
	}
}
