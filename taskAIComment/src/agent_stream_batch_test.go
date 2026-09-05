package main

import (
	"testing"
	"time"
)

func TestContainerAgentAppendBatcherFlushesOnSize(t *testing.T) {
	setupTestDB(t)

	comment := &ContainerAgentComment{
		ID: "batch-size-1", TenantID: "t1", WorkspaceID: "ws1", TaskID: "task1",
		ParentCommentID: "p1", InstalledImageID: "img1", RunStatus: runStatusPending,
	}
	if err := insertContainerAgentComment(comment); err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() { removeContainerAgentBatcher(comment.ID) })

	big := make([]byte, agentChunkBatchMaxLen)
	for i := range big {
		big[i] = 'x'
	}
	merged, ok, err := appendContainerAgentChunkBatched(comment.ID, string(big))
	if err != nil || !ok {
		t.Fatalf("append big: ok=%v err=%v", ok, err)
	}
	if len(merged) != agentChunkBatchMaxLen {
		t.Fatalf("merged len=%d", len(merged))
	}
	loaded, err := loadContainerAgentComment(comment.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.AssistantResponse.Valid || len(loaded.AssistantResponse.String) != agentChunkBatchMaxLen {
		t.Fatalf("db not flushed on size: %v", loaded.AssistantResponse)
	}
}

func TestContainerAgentAppendBatcherFlushesOnTimer(t *testing.T) {
	setupTestDB(t)

	comment := &ContainerAgentComment{
		ID: "batch-timer-1", TenantID: "t1", WorkspaceID: "ws1", TaskID: "task1",
		ParentCommentID: "p1", InstalledImageID: "img1", RunStatus: runStatusPending,
	}
	if err := insertContainerAgentComment(comment); err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() { removeContainerAgentBatcher(comment.ID) })

	if _, _, err := appendContainerAgentChunkBatched(comment.ID, "hello"); err != nil {
		t.Fatalf("append: %v", err)
	}
	time.Sleep(agentChunkBatchWindow + 40*time.Millisecond)
	loaded, err := loadContainerAgentComment(comment.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.AssistantResponse.Valid || loaded.AssistantResponse.String != "hello" {
		t.Fatalf("timed flush missing: %v", loaded.AssistantResponse)
	}
}

func TestFlushContainerAgentBatcherBeforeComplete(t *testing.T) {
	setupTestDB(t)

	bodyMap := map[string]interface{}{
		"tenant_id": "t1", "workspace_id": "ws1", "task_id": "task1",
		"parent_comment_id": "human-flush", "installed_image_id": "img-1",
	}
	c, st, eb := createPendingContainerAgentComment(bodyMap)
	if eb != nil {
		t.Fatalf("create: %d %v", st, eb)
	}
	t.Cleanup(func() { removeContainerAgentBatcher(c.ID) })

	if _, _, err := appendContainerAgentChunkBatched(c.ID, "pending "); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := flushContainerAgentBatcher(c.ID); err != nil {
		t.Fatalf("flush: %v", err)
	}
	loaded, err := loadContainerAgentComment(c.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.AssistantResponse.Valid || loaded.AssistantResponse.String != "pending " {
		t.Fatalf("assistant_response = %v", loaded.AssistantResponse)
	}
}
