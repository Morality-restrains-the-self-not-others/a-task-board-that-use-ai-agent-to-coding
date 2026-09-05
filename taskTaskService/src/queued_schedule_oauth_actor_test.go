package main

import (
	"context"
	"testing"
)

func TestStartQueuedMembershipUsesCommentAuthorNotOwner(t *testing.T) {
	setupTestDB(t)
	_, _ = startAutoRunMockServices(t, true)

	var gotUserID string
	prevVM := startVMFn
	startVMFn = func(ctx context.Context, tenantID, workspaceID, apiPath, userID string, body map[string]interface{}) error {
		gotUserID = userID
		return nil
	}
	t.Cleanup(func() { startVMFn = prevVM })

	taskID := insertQueuedTestTask(t, "t1", "ws1", "queued oauth actor", "")
	if _, err := db.Exec(`UPDATE task_tasks SET installed_image_id=?, owner_id=? WHERE id=?`, "img1", "owner-user", taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_oauth_actor", taskID, "p1", "main", "feat/x", "https://git.example/repo.git",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,repo_identities_json,created_at) VALUES(?,?,?,?,?,?,NOW())`,
		"cmt-mock-auto-run", taskID, "enabler-user", "【自动运行】queued", "[]", "[]",
	); err != nil {
		t.Fatal(err)
	}

	err := startQueuedMembership(queuedMembership{
		TaskID:        taskID,
		TenantID:      "t1",
		WorkspaceID:   "ws1",
		TopTaskID:     taskID,
		Status:        "queued",
		EnablerUserID: "enabler-user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotUserID != "enabler-user" {
		t.Fatalf("UserID=%q want enabler-user (not owner)", gotUserID)
	}
}
