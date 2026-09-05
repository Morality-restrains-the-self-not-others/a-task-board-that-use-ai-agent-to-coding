package domain

import (
	"fmt"
	"strings"
)

const DefaultStartupLogPathRule = "workspace_{workspaceId}/task_{taskId}/comment_{commentId}/startup_logs.json"

var startupLogPlaceholders = map[string]struct{}{
	"workspaceId": {},
	"taskId":      {},
	"commentId":   {},
	"keyPrefix":   {},
}

// StartupLogIDs identifies one comment-level startup log bundle in object storage.
type StartupLogIDs struct {
	WorkspaceID string
	TaskID      string
	CommentID   string
	KeyPrefix   string
}

func (id StartupLogIDs) Normalize() StartupLogIDs {
	return StartupLogIDs{
		WorkspaceID: strings.TrimSpace(id.WorkspaceID),
		TaskID:      strings.TrimSpace(id.TaskID),
		CommentID:   strings.TrimSpace(id.CommentID),
		KeyPrefix:   strings.TrimSpace(id.KeyPrefix),
	}
}

func (id StartupLogIDs) Validate() error {
	id = id.Normalize()
	if id.WorkspaceID == "" || id.TaskID == "" || id.CommentID == "" {
		return fmt.Errorf("workspace_id, task_id and comment_id required")
	}
	return nil
}

func ValidateStartupLogPathRule(rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return fmt.Errorf("startupLogsPathRule 不能为空")
	}
	if strings.Contains(rule, "..") {
		return fmt.Errorf("startupLogsPathRule 禁止 ..")
	}
	if strings.HasPrefix(rule, "/") {
		return fmt.Errorf("startupLogsPathRule 禁止绝对路径")
	}
	found := map[string]bool{}
	for _, m := range placeholderRe.FindAllStringSubmatch(rule, -1) {
		name := m[1]
		if _, ok := startupLogPlaceholders[name]; !ok {
			return fmt.Errorf("startupLogsPathRule 未知占位符 {%s}", name)
		}
		found[name] = true
	}
	for _, need := range []string{"workspaceId", "taskId", "commentId"} {
		if !found[need] {
			return fmt.Errorf("startupLogsPathRule 必须包含 {%s}", need)
		}
	}
	return nil
}

func RenderStartupLogObjectKey(rule string, id StartupLogIDs) (string, error) {
	id = id.Normalize()
	if err := id.Validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(rule) == "" {
		rule = DefaultStartupLogPathRule
	}
	if err := ValidateStartupLogPathRule(rule); err != nil {
		return "", err
	}
	repl := map[string]string{
		"workspaceId": SanitizePathToken(id.WorkspaceID),
		"taskId":      SanitizePathToken(id.TaskID),
		"commentId":   SanitizePathToken(id.CommentID),
		"keyPrefix":   SanitizePathToken(id.KeyPrefix),
	}
	out := placeholderRe.ReplaceAllStringFunc(rule, func(tok string) string {
		name := strings.Trim(tok, "{}")
		return repl[name]
	})
	out = strings.Trim(out, "/")
	if out == "" || strings.Contains(out, "..") {
		return "", fmt.Errorf("rendered object key invalid")
	}
	return out, nil
}
