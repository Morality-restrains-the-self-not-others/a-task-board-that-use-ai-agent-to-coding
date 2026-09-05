package domain

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultStepFullPathRule = "workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json"
	MaxStepFullBytes        = 8 << 20
)

var (
	safeIDRe            = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	placeholderRe       = regexp.MustCompile(`\{([A-Za-z]+)\}`)
	allowedPlaceholders = map[string]struct{}{
		"workspaceId": {},
		"taskId":      {},
		"commentId":   {},
		"jobId":       {},
		"layerId":     {},
		"keyPrefix":   {},
	}
)

// StepFullIDs identifies one archived job inside a comment bundle.
type StepFullIDs struct {
	WorkspaceID string
	TaskID      string
	CommentID   string
	JobID       string
	LayerID     string
	KeyPrefix   string
}

func (id StepFullIDs) Normalize() StepFullIDs {
	return StepFullIDs{
		WorkspaceID: strings.TrimSpace(id.WorkspaceID),
		TaskID:      strings.TrimSpace(id.TaskID),
		CommentID:   strings.TrimSpace(id.CommentID),
		JobID:       strings.TrimSpace(id.JobID),
		LayerID:     strings.TrimSpace(id.LayerID),
		KeyPrefix:   strings.TrimSpace(id.KeyPrefix),
	}
}

func (id StepFullIDs) Validate() error {
	id = id.Normalize()
	if id.WorkspaceID == "" || id.TaskID == "" || id.CommentID == "" || id.JobID == "" {
		return fmt.Errorf("workspace_id, task_id, comment_id and job_id required")
	}
	return nil
}

func SanitizePathToken(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "..", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, `\`, "")
	s = safeIDRe.ReplaceAllString(s, "_")
	if s == "" {
		return "_"
	}
	return s
}

func ValidateStepFullPathRule(rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return fmt.Errorf("pathRule 不能为空")
	}
	if strings.Contains(rule, "..") {
		return fmt.Errorf("pathRule 禁止 ..")
	}
	if strings.HasPrefix(rule, "/") {
		return fmt.Errorf("pathRule 禁止绝对路径")
	}
	found := map[string]bool{}
	for _, m := range placeholderRe.FindAllStringSubmatch(rule, -1) {
		name := m[1]
		if _, ok := allowedPlaceholders[name]; !ok {
			return fmt.Errorf("pathRule 未知占位符 {%s}", name)
		}
		found[name] = true
	}
	for _, need := range []string{"workspaceId", "taskId", "commentId"} {
		if !found[need] {
			return fmt.Errorf("pathRule 必须包含 {%s}", need)
		}
	}
	return nil
}

func RenderStepFullObjectKey(rule string, id StepFullIDs) (string, error) {
	id = id.Normalize()
	if err := id.Validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(rule) == "" {
		rule = DefaultStepFullPathRule
	}
	if err := ValidateStepFullPathRule(rule); err != nil {
		return "", err
	}
	repl := map[string]string{
		"workspaceId": SanitizePathToken(id.WorkspaceID),
		"taskId":      SanitizePathToken(id.TaskID),
		"commentId":   SanitizePathToken(id.CommentID),
		"jobId":       SanitizePathToken(id.JobID),
		"layerId":     SanitizePathToken(id.LayerID),
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
