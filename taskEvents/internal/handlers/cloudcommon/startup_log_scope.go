package cloudcommon

import (
	"strings"
)

// StartupLogScope correlates SSE / status log lines to an ECS instance and container name.
type StartupLogScope struct {
	InstanceID    string
	ContainerName string
	// Optional inputs used to derive ContainerName when unset.
	TaskID    string
	CommentID string
}

// FormatStartupLogLabel builds [实例ID、容器名]. Missing slots use "-".
func FormatStartupLogLabel(s StartupLogScope) string {
	inst := strings.TrimSpace(s.InstanceID)
	name := strings.TrimSpace(s.ContainerName)
	if name == "" {
		name = DeriveContainerName("", s.TaskID, s.CommentID)
	}
	if inst == "" && name == "" {
		return ""
	}
	return "[" + slotOrDash(inst, 48) + "、" + slotOrDash(name, 64) + "]"
}

// DeriveContainerName prefers explicit name, else a single-prefixed
// task_{taskID}_{commentID}. When taskID already starts with "task_", do not
// prepend another "task_" (avoids task_task_1566…_cmt_…).
func DeriveContainerName(containerName, taskID, commentID string) string {
	if n := strings.TrimSpace(containerName); n != "" {
		return n
	}
	tid := strings.TrimSpace(taskID)
	cid := strings.TrimSpace(commentID)
	if tid == "" || cid == "" {
		return ""
	}
	if strings.HasPrefix(tid, "task_") {
		return tid + "_" + cid
	}
	return "task_" + tid + "_" + cid
}

// ApplyStartupLogScope prefixes message and attaches structured fields for frontend filtering.
func ApplyStartupLogScope(statusData map[string]interface{}, scope StartupLogScope) map[string]interface{} {
	if statusData == nil {
		statusData = map[string]interface{}{}
	}
	if scope.InstanceID == "" {
		scope.InstanceID = firstNonEmptyTrim(
			strField(statusData, "instance_id"),
			nestedStr(statusData, "vm_info", "instance_id"),
		)
	}
	if scope.ContainerName == "" {
		scope.ContainerName = firstNonEmptyTrim(
			strField(statusData, "container_name"),
			strField(statusData, "mock_container_name"),
			DeriveContainerName("", scope.TaskID, scope.CommentID),
		)
	}
	label := FormatStartupLogLabel(scope)
	if label != "" {
		statusData["log_label"] = label
		if msg, ok := statusData["message"].(string); ok && strings.TrimSpace(msg) != "" {
			trimmed := strings.TrimSpace(msg)
			if !hasStartupLogPrefix(trimmed) {
				statusData["message"] = label + " " + msg
			}
		}
	}
	if v := strings.TrimSpace(scope.InstanceID); v != "" {
		statusData["instance_id"] = v
	}
	if v := strings.TrimSpace(scope.ContainerName); v != "" {
		statusData["container_name"] = v
	}
	if v := strings.TrimSpace(scope.CommentID); v != "" {
		statusData["comment_id"] = v
	}
	return statusData
}

// ScopeFromEventData extracts startup scope fields from a domain event payload map.
func ScopeFromEventData(data map[string]interface{}) StartupLogScope {
	if data == nil {
		return StartupLogScope{}
	}
	commentID := strField(data, "parent_comment_id")
	if commentID == "" {
		commentID = strField(data, "comment_id")
	}
	taskID := strField(data, "task_id")
	name := firstNonEmptyTrim(
		strField(data, "container_name"),
		strField(data, "mock_container_name"),
		DeriveContainerName("", taskID, commentID),
	)
	return StartupLogScope{
		InstanceID:    firstNonEmptyTrim(strField(data, "instance_id"), nestedStr(data, "vm_info", "instance_id")),
		ContainerName: name,
		TaskID:        taskID,
		CommentID:     commentID,
	}
}

// WithInstanceID returns a copy of scope with InstanceID set when non-empty.
func WithInstanceID(scope StartupLogScope, instanceID string) StartupLogScope {
	if id := strings.TrimSpace(instanceID); id != "" {
		scope.InstanceID = id
	}
	return scope
}

func hasStartupLogPrefix(msg string) bool {
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

func nestedStr(data map[string]interface{}, key, nestedKey string) string {
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

func slotOrDash(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return truncateLabel(s, max)
}

func truncateLabel(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func firstNonEmptyTrim(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
