package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	executionModeWaitPrevious = "wait_previous"
	executionModeIndependent  = "independent"
)

func normalizeCommentExecutionMode(raw string) (string, error) {
	mode := strings.TrimSpace(raw)
	if mode == "" {
		return executionModeWaitPrevious, nil
	}
	switch mode {
	case executionModeWaitPrevious, executionModeIndependent:
		return mode, nil
	default:
		return "", fmt.Errorf("execution_mode must be wait_previous or independent")
	}
}

func updateCommentExecutionMode(commentID, mode string) error {
	now := time.Now().UTC()
	_, err := db.Exec(
		`UPDATE ai_comment_task_comments SET execution_mode=?, updated_at=? WHERE id=?`,
		mode, now, commentID,
	)
	return err
}
