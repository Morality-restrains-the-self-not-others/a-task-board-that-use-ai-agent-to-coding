package taskcommentimagementioned

import (
	"strings"

	"taskEvents/internal/handlers/payload"
)

// shouldColdStartOnImageMention 决定 @镜像 事件是否立即 start-vm。
// independent 立即启动；wait_previous 仅在无未完成前序时启动（首条评论）。
// 有前序的串行评论由 comment_container_bindings advance 在前序 completed 后再启。
func shouldColdStartOnImageMention(data map[string]interface{}) bool {
	mode := strings.TrimSpace(payload.StrField(data, "execution_mode"))
	if mode == "independent" {
		return true
	}
	if boolish(data["has_unfinished_predecessors"]) {
		return false
	}
	if mentionDependsOnCommentIDs(data) > 0 {
		return false
	}
	return true
}

func boolish(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		s := strings.TrimSpace(strings.ToLower(b))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

func mentionDependsOnCommentIDs(data map[string]interface{}) int {
	if data == nil {
		return 0
	}
	raw, ok := data["depends_on_comment_ids"]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case []interface{}:
		n := 0
		for _, item := range v {
			if strings.TrimSpace(payload.StrField(map[string]interface{}{"id": item}, "id")) != "" {
				n++
			}
		}
		return n
	case []string:
		n := 0
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				n++
			}
		}
		return n
	case string:
		s := strings.TrimSpace(v)
		if s == "" || s == "[]" {
			return 0
		}
		return 1
	default:
		return 0
	}
}
