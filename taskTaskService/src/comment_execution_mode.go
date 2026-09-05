package main

import (
	"fmt"
	"strings"
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

func updateHumanCommentExecutionMode(commentID, mode string) error {
	_, err := db.Exec(
		`UPDATE task_comments SET execution_mode=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		mode, commentID,
	)
	return err
}

func loadHumanCommentExecutionMode(commentID string) (string, error) {
	var mode string
	err := db.QueryRow(`SELECT execution_mode FROM task_comments WHERE id=?`, commentID).Scan(&mode)
	return mode, err
}
