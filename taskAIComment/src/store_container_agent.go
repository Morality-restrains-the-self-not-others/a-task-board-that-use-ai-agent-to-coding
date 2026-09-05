package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ContainerAgentComment struct {
	ID                string
	TenantID          string
	WorkspaceID       string
	TaskID            string
	ParentCommentID   string
	InstalledImageID  string
	RunStatus         string
	Content           sql.NullString
	AssistantResponse sql.NullString
	ContextPackJSON   sql.NullString
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func insertContainerAgentComment(c *ContainerAgentComment) error {
	now := time.Now().UTC()
	if c.ID == "" {
		c.ID = newCommentID()
	}
	if c.RunStatus == "" {
		c.RunStatus = runStatusPending
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	_, err := db.Exec(
		`INSERT INTO ai_comment_container_agent_comments(
			id, tenant_id, workspace_id, task_id, parent_comment_id, installed_image_id,
			run_status, content, assistant_response, context_pack_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.TenantID, c.WorkspaceID, c.TaskID, c.ParentCommentID, c.InstalledImageID,
		c.RunStatus, nullStringValue(c.Content), nullStringValue(c.AssistantResponse),
		nullStringValue(c.ContextPackJSON), c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func loadContainerAgentComment(id string) (*ContainerAgentComment, error) {
	row := db.QueryRow(
		`SELECT id, tenant_id, workspace_id, task_id, parent_comment_id, installed_image_id,
			run_status, content, assistant_response, context_pack_json, created_at, updated_at
		 FROM ai_comment_container_agent_comments WHERE id = ?`, id,
	)
	return scanContainerAgentComment(row)
}

func listContainerAgentCommentsByTask(taskID string) ([]ContainerAgentComment, error) {
	rows, err := db.Query(
		`SELECT id, tenant_id, workspace_id, task_id, parent_comment_id, installed_image_id,
			run_status, content, assistant_response, context_pack_json, created_at, updated_at
		 FROM ai_comment_container_agent_comments WHERE task_id = ? ORDER BY created_at ASC, id ASC`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContainerAgentComment{}
	for rows.Next() {
		c, err := scanContainerAgentComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func listContainerAgentCommentsByTaskPage(taskID string, p commentPageParams) ([]ContainerAgentComment, bool, error) {
	limit := p.limit
	if limit < 1 {
		limit = defaultCommentLimit
	}
	query := `SELECT id, tenant_id, workspace_id, task_id, parent_comment_id, installed_image_id,
			run_status, content, assistant_response, context_pack_json, created_at, updated_at
		 FROM ai_comment_container_agent_comments WHERE task_id = ?`
	args := []interface{}{taskID}
	if p.hasCursor {
		query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
		args = append(args, p.cursorCreatedAt, p.cursorCreatedAt, p.cursorID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []ContainerAgentComment{}
	for rows.Next() {
		c, err := scanContainerAgentComment(rows)
		if err != nil {
			return nil, false, err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	return out, hasMore, nil
}

func loadLatestActiveContainerAgentByTask(taskID string) (*ContainerAgentComment, error) {
	row := db.QueryRow(
		`SELECT id, tenant_id, workspace_id, task_id, parent_comment_id, installed_image_id,
			run_status, content, assistant_response, context_pack_json, created_at, updated_at
		 FROM ai_comment_container_agent_comments
		 WHERE task_id = ? AND run_status IN (?, ?, ?, ?)
		 ORDER BY created_at DESC, id DESC LIMIT 1`,
		taskID, runStatusPending, runStatusStarting, runStatusRunning, runStatusStreaming,
	)
	c, err := scanContainerAgentComment(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func countActiveContainerAgentRunsByParent(parentCommentID string) (int, error) {
	row := db.QueryRow(
		`SELECT COUNT(*) FROM ai_comment_container_agent_comments
		 WHERE parent_comment_id = ? AND run_status IN (?, ?, ?, ?)`,
		parentCommentID, runStatusPending, runStatusStarting, runStatusRunning, runStatusStreaming,
	)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func loadLatestActiveContainerAgentByParent(parentCommentID string) (*ContainerAgentComment, error) {
	row := db.QueryRow(
		`SELECT id, tenant_id, workspace_id, task_id, parent_comment_id, installed_image_id,
			run_status, content, assistant_response, context_pack_json, created_at, updated_at
		 FROM ai_comment_container_agent_comments
		 WHERE parent_comment_id = ? AND run_status IN (?, ?, ?, ?)
		 ORDER BY created_at DESC LIMIT 1`,
		parentCommentID, runStatusPending, runStatusStarting, runStatusRunning, runStatusStreaming,
	)
	return scanContainerAgentComment(row)
}

func appendContainerAgentAssistantResponse(id, chunk string) (string, bool, error) {
	return appendContainerAgentChunkBatched(id, chunk)
}

func touchContainerAgentStreamingStatus(id string) error {
	now := time.Now().UTC()
	_, err := db.Exec(
		`UPDATE ai_comment_container_agent_comments SET updated_at = ?,
		 run_status = CASE WHEN run_status IN (?, ?, ?) THEN ? ELSE run_status END
		 WHERE id = ?`,
		now, runStatusPending, runStatusStarting, runStatusRunning, runStatusStreaming, id,
	)
	return err
}

func setContainerAgentAssistantResponse(id, text string) (bool, error) {
	now := time.Now().UTC()
	res, err := db.Exec(
		`UPDATE ai_comment_container_agent_comments SET assistant_response = ?, updated_at = ? WHERE id = ?`,
		text, now, id,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func updateContainerAgentRunStatus(id, status string) (bool, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return false, fmt.Errorf("run_status required")
	}
	now := time.Now().UTC()
	res, err := db.Exec(
		`UPDATE ai_comment_container_agent_comments SET run_status = ?, updated_at = ? WHERE id = ?`,
		status, now, id,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func updateContainerAgentContextPack(id, packJSON string) (bool, error) {
	now := time.Now().UTC()
	res, err := db.Exec(
		`UPDATE ai_comment_container_agent_comments SET context_pack_json = ?, updated_at = ? WHERE id = ?`,
		packJSON, now, id,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func updateContainerAgentPatch(id string, runStatus *string, packJSON *string) (bool, error) {
	if runStatus == nil && packJSON == nil {
		return false, fmt.Errorf("nothing to update")
	}
	now := time.Now().UTC()
	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	if runStatus != nil {
		sets = append(sets, "run_status = ?")
		args = append(args, *runStatus)
	}
	if packJSON != nil {
		sets = append(sets, "context_pack_json = ?")
		args = append(args, *packJSON)
	}
	args = append(args, id)
	res, err := db.Exec(
		`UPDATE ai_comment_container_agent_comments SET `+strings.Join(sets, ", ")+` WHERE id = ?`,
		args...,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func scanContainerAgentComment(row rowScanner) (*ContainerAgentComment, error) {
	var c ContainerAgentComment
	var content, assistant, pack sql.NullString
	err := row.Scan(
		&c.ID, &c.TenantID, &c.WorkspaceID, &c.TaskID, &c.ParentCommentID, &c.InstalledImageID,
		&c.RunStatus, &content, &assistant, &pack, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Content = content
	c.AssistantResponse = assistant
	c.ContextPackJSON = pack
	return &c, nil
}

func parseContextPackJSON(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func enrichContextPackRunIDs(pack map[string]interface{}, agentCommentID string) map[string]interface{} {
	if pack == nil {
		return nil
	}
	at, _ := pack["at_mention_run"].(map[string]interface{})
	if at == nil {
		at = map[string]interface{}{}
		pack["at_mention_run"] = at
	}
	at["run_id"] = agentCommentID
	at["agent_comment_id"] = agentCommentID
	return pack
}

func marshalContextPack(pack map[string]interface{}) (string, error) {
	if pack == nil {
		return "", nil
	}
	b, err := json.Marshal(pack)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
