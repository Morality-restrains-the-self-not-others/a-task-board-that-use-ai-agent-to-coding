package main

import "strings"

// resolveTerminalKind maps progress column name / completed flag to a terminal kind.
// Aligned with taskEvents/internal/handlers/taskstatuschanged.ResolveTerminalKind.
// Returns "" (non-terminal), "completed", or "cancelled".
func resolveTerminalKind(columnName string, completedOrBecameTrue bool) string {
	lower := strings.ToLower(strings.TrimSpace(columnName))
	switch lower {
	case "已取消", "cancelled", "canceled":
		return "cancelled"
	case "已完成", "completed":
		return "completed"
	}
	if completedOrBecameTrue {
		return "completed"
	}
	return ""
}

func isTerminalState(columnName string, completed bool) bool {
	return resolveTerminalKind(columnName, completed) != ""
}

// updateEntersTerminal is true when a task update moves into 已完成/已取消
// (by progress column name or completed flag flipping to true).
func updateEntersTerminal(prevColumnID, newColumnID, columnName string, prevCompleted, completed bool) bool {
	if prevColumnID != newColumnID && resolveTerminalKind(columnName, false) != "" {
		return true
	}
	return !prevCompleted && completed
}
