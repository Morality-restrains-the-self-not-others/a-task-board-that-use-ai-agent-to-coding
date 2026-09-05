package main

import "testing"

func TestNotifyContainerAgentPendingDoesNotCreatePendingRow(t *testing.T) {
	prev := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = ""
	t.Cleanup(func() { cfg.AICommentServiceURL = prev })

	err := notifyContainerAgentPending("t1", "ws1", "task_1", "cmt_1", "img-1", "trae-agent", "@trae-agent hi", "u1")
	if err != nil {
		t.Fatalf("platform notify must defer to container, got %v", err)
	}
}
