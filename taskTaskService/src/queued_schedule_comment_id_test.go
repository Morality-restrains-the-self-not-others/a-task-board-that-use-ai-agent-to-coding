package main

import (
	"context"
	"errors"
	"testing"
)

func TestStartQueuedMembershipAttachesCommentID(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	taskID := insertQueuedTestTask(t, "t1", "ws1", "queued auto run", "")
	if _, err := db.Exec(`UPDATE task_tasks SET installed_image_id=? WHERE id=?`, "img1", taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_q_cmt", taskID, "p1", "main", "feat/x", "https://git.example/repo.git",
	); err != nil {
		t.Fatal(err)
	}

	err := startQueuedMembership(queuedMembership{
		TaskID:      taskID,
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TopTaskID:   taskID,
		Status:      "queued",
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	body := startVmBodyFromCloudCalls(t, cloudCalls)
	if body["comment_id"] != "cmt-mock-auto-run" {
		t.Fatalf("comment_id=%v body=%v", body["comment_id"], body)
	}
	if body["parent_comment_id"] != "cmt-mock-auto-run" {
		t.Fatalf("parent_comment_id=%v", body["parent_comment_id"])
	}
	if !hasQueuedSlot(taskID) {
		t.Fatal("success must hold queued slot so UI occupied count is 1")
	}
}

func TestStartQueuedMembershipFailureResetsQueuedAndReleasesSlot(t *testing.T) {
	setupTestDB(t)
	_, _ = startAutoRunMockServices(t, true)

	taskID := insertQueuedTestTask(t, "t1", "ws1", "queued auto run", "")
	if _, err := db.Exec(`UPDATE task_tasks SET installed_image_id=? WHERE id=?`, "img1", taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_q_cmt_fail", taskID, "p1", "main", "feat/x", "https://git.example/repo.git",
	); err != nil {
		t.Fatal(err)
	}
	rec, err := loadTask(taskID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enqueueQueuedAutoRun(rec, "u-test"); err != nil {
		t.Fatal(err)
	}

	prev := startVMFn
	startVMFn = func(ctx context.Context, tenantID, workspaceID, apiPath, userID string, body map[string]interface{}) error {
		return errors.New("cloud timeout")
	}
	t.Cleanup(func() { startVMFn = prev })

	err = startQueuedMembership(queuedMembership{
		TaskID:      taskID,
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TopTaskID:   taskID,
		Status:      "queued",
	})
	if err == nil {
		t.Fatal("want startVM error")
	}
	m, _ := loadMembership(taskID)
	if m == nil || m.Status != "queued" {
		t.Fatalf("failure must reset membership to queued, got %+v", m)
	}
	if hasQueuedSlot(taskID) {
		t.Fatal("failure must release slot so occupied count is 0")
	}
}
