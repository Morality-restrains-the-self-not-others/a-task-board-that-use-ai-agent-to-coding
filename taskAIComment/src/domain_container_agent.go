package main

import (
	"fmt"
	"strings"
)

const (
	runStatusPending    = "pending"
	runStatusStarting   = "starting"
	runStatusRunning    = "running"
	runStatusStreaming  = "streaming"
	runStatusCompleted  = "completed"
	runStatusFailed     = "failed"
)

var terminalRunStatuses = map[string]bool{
	runStatusCompleted: true,
	runStatusFailed:    true,
}

var activeRunStatuses = map[string]bool{
	runStatusPending:   true,
	runStatusStarting:  true,
	runStatusRunning:   true,
	runStatusStreaming: true,
}

func isTerminalRunStatus(status string) bool {
	return terminalRunStatuses[strings.TrimSpace(status)]
}

func isActiveRunStatus(status string) bool {
	return activeRunStatuses[strings.TrimSpace(status)]
}

// ValidateOneActiveRun ensures at most one non-terminal run exists per parent comment.
func ValidateOneActiveRun(activeCount int) error {
	if activeCount > 0 {
		return fmt.Errorf("parent_comment_id already has an active container agent run")
	}
	return nil
}
