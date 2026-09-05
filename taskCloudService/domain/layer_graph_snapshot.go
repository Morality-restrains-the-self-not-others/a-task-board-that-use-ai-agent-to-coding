package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LayerGraphSnapshotID is the aggregate identity for a comment-scoped ztree snapshot.
type LayerGraphSnapshotID struct {
	WorkspaceID string
	TaskID      string
	CommentID   string
}

func (id LayerGraphSnapshotID) Normalize() LayerGraphSnapshotID {
	return LayerGraphSnapshotID{
		WorkspaceID: strings.TrimSpace(id.WorkspaceID),
		TaskID:      strings.TrimSpace(id.TaskID),
		CommentID:   strings.TrimSpace(id.CommentID),
	}
}

func (id LayerGraphSnapshotID) Validate() error {
	id = id.Normalize()
	if id.WorkspaceID == "" || id.TaskID == "" || id.CommentID == "" {
		return fmt.Errorf("workspace_id, task_id and comment_id required")
	}
	return nil
}

// ValidateGraphDocument requires layers and jobs to be JSON arrays.
func ValidateGraphDocument(raw []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("graph_json must be a JSON object")
	}
	if _, ok := doc["layers"].([]any); !ok {
		return fmt.Errorf("layers 须为 JSON 数组")
	}
	if _, ok := doc["jobs"].([]any); !ok {
		return fmt.Errorf("jobs 须为 JSON 数组")
	}
	return nil
}

func EmptyGraphDocument() map[string]any {
	return map[string]any{
		"layers": []any{},
		"jobs":   []any{},
	}
}
