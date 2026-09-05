package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const MaxStartupLogBytes = 2 << 20

// StartupLogEntry is one comment startup-log line (same id as the MySQL shard row).
type StartupLogEntry struct {
	ID          string `json:"id"`
	BindingID   string `json:"binding_id,omitempty"`
	Stage       string `json:"stage,omitempty"`
	Message     string `json:"message,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	CompanyID   string `json:"company_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	CommentID   string `json:"comment_id,omitempty"`
}

// StartupLogBundle is the comment-level COS object.
type StartupLogBundle struct {
	WorkspaceID string            `json:"workspace_id"`
	TaskID      string            `json:"task_id"`
	CommentID   string            `json:"comment_id"`
	Logs        []StartupLogEntry `json:"logs"`
}

func EmptyStartupLogBundle(id StartupLogIDs) StartupLogBundle {
	id = id.Normalize()
	return StartupLogBundle{
		WorkspaceID: id.WorkspaceID,
		TaskID:      id.TaskID,
		CommentID:   id.CommentID,
		Logs:        []StartupLogEntry{},
	}
}

func ParseStartupLogBundle(raw []byte) (StartupLogBundle, error) {
	var b StartupLogBundle
	if len(raw) == 0 {
		return b, fmt.Errorf("empty bundle")
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return b, err
	}
	if b.Logs == nil {
		b.Logs = []StartupLogEntry{}
	}
	return b, nil
}

func MarshalStartupLogBundle(b StartupLogBundle) ([]byte, error) {
	if b.Logs == nil {
		b.Logs = []StartupLogEntry{}
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxStartupLogBytes {
		return nil, fmt.Errorf("startup log bundle exceeds %d bytes", MaxStartupLogBytes)
	}
	return raw, nil
}

func MergeStartupLogEntry(existing StartupLogBundle, id StartupLogIDs, entry StartupLogEntry) StartupLogBundle {
	id = id.Normalize()
	if existing.Logs == nil {
		existing = EmptyStartupLogBundle(id)
	}
	existing.WorkspaceID = id.WorkspaceID
	existing.TaskID = id.TaskID
	existing.CommentID = id.CommentID
	entry.ID = strings.TrimSpace(entry.ID)
	if entry.ID == "" {
		return existing
	}
	for i := range existing.Logs {
		if existing.Logs[i].ID == entry.ID {
			existing.Logs[i] = entry
			return existing
		}
	}
	existing.Logs = append(existing.Logs, entry)
	sort.SliceStable(existing.Logs, func(i, j int) bool {
		if existing.Logs[i].CreatedAt != existing.Logs[j].CreatedAt {
			return existing.Logs[i].CreatedAt < existing.Logs[j].CreatedAt
		}
		return existing.Logs[i].ID < existing.Logs[j].ID
	})
	return existing
}

func ParseStartupLogCreatedAt(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
