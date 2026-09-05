package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestCreateTaskForkRetargetsAncestorWorkBranch(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	ancestor := "feature/2026-09-02____daydaymoneytask_882908895993950208_add-current-time-to-now.md"
	body, err := json.Marshal(map[string]interface{}{
		"title":        "fork branch retarget",
		"workspace_id": "ws1",
		"fork_from":    "task_882943770763489280",
		"task_kind":    "bug-fix",
		"branch_strategy": map[string]string{
			"work_branch_name":         ancestor,
			"merge_target_branch_name": "main",
			"target_branch_name":       ancestor,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, payload := postCreateTask(t, string(body), nil)
	if rec.Code != 201 {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	id := fmt.Sprintf("%v", payload["id"])
	if !strings.HasPrefix(id, "task_") {
		t.Fatalf("id=%q", id)
	}
	bs, _ := payload["branch_strategy"].(map[string]interface{})
	work := fmt.Sprintf("%v", bs["work_branch_name"])
	target := fmt.Sprintf("%v", bs["target_branch_name"])
	if strings.Contains(work, "882908895993950208") {
		t.Fatalf("work_branch still ancestor: %q", work)
	}
	if !strings.Contains(work, strings.TrimPrefix(id, "task_")) {
		t.Fatalf("work_branch=%q missing new task id %s", work, id)
	}
	if work != target {
		t.Fatalf("work=%q target=%q", work, target)
	}
	if fmt.Sprintf("%v", bs["merge_target_branch_name"]) != "main" {
		t.Fatalf("merge_target=%v", bs["merge_target_branch_name"])
	}
}
