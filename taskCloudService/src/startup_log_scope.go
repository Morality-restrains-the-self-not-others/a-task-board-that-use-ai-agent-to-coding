package main

import (
	"strings"
)

// withStartupImageLogScope prefixes SSE message with [实例ID、容器名].
func withStartupImageLogScope(body map[string]interface{}, statusData map[string]interface{}) map[string]interface{} {
	if statusData == nil {
		statusData = map[string]interface{}{}
	}
	inst := firstNonEmptyStartupField(
		strField(statusData, "instance_id"),
		strField(body, "instance_id"),
		nestedMapStr(statusData, "vm_info", "instance_id"),
	)
	cmt := firstNonEmptyStartupField(strField(body, "parent_comment_id"), strField(body, "comment_id"))
	taskID := strField(body, "task_id")
	name := firstNonEmptyStartupField(
		strField(statusData, "container_name"),
		strField(body, "container_name"),
		strField(body, "mock_container_name"),
		deriveStartupContainerName(taskID, cmt),
	)
	if inst == "" && name == "" {
		return statusData
	}
	label := "[" + startupSlotOrDash(inst, 48) + "、" + startupSlotOrDash(name, 64) + "]"
	statusData["log_label"] = label
	if msg := strField(statusData, "message"); msg != "" && !hasStartupLogLabelPrefix(msg) {
		statusData["message"] = label + " " + msg
	}
	if inst != "" {
		statusData["instance_id"] = inst
	}
	if name != "" {
		statusData["container_name"] = name
	}
	if cmt != "" {
		statusData["comment_id"] = cmt
	}
	return statusData
}

func deriveStartupContainerName(taskID, commentID string) string {
	return buildCommentMockContainerName(taskID, commentID)
}

func hasStartupLogLabelPrefix(msg string) bool {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "[") {
		if end := strings.Index(msg, "]"); end > 1 && strings.Contains(msg[1:end], "、") {
			return true
		}
	}
	return strings.HasPrefix(msg, "[实例:") ||
		strings.HasPrefix(msg, "[镜像:") ||
		strings.HasPrefix(msg, "[评论:")
}

func nestedMapStr(data map[string]interface{}, key, nestedKey string) string {
	if data == nil {
		return ""
	}
	raw, ok := data[key]
	if !ok || raw == nil {
		return ""
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	return strField(m, nestedKey)
}

func firstNonEmptyStartupField(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func startupSlotOrDash(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return truncateStartupLabel(s, max)
}

func truncateStartupLabel(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
