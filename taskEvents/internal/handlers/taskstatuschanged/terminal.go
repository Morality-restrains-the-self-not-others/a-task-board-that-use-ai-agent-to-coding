package taskstatuschanged

import "strings"

// ResolveTerminalKind maps progress column name / completed transition to a terminal kind.
// Returns "" (non-terminal), "completed", or "cancelled".
// Column matching is case-insensitive for Latin aliases; CJK names match exactly after trim.
func ResolveTerminalKind(columnName string, completedBecameTrue bool) string {
	lower := strings.ToLower(strings.TrimSpace(columnName))
	switch lower {
	case "已取消", "cancelled", "canceled":
		return "cancelled"
	case "已完成", "completed":
		return "completed"
	}
	if completedBecameTrue {
		return "completed"
	}
	return ""
}
