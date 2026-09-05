package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type AIComment struct {
	ID                string
	Content           string
	AssistantResponse sql.NullString
	TaskID            string
	TenantID          string
	WorkspaceID       string
	CreatedByID       string
	ExecutionMode     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func newCommentID() string {
	return strconv.FormatInt(GenerateID(), 10)
}

func insertComment(c *AIComment) error {
	now := time.Now().UTC()
	if c.ID == "" {
		c.ID = newCommentID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	if c.ExecutionMode == "" {
		c.ExecutionMode = executionModeWaitPrevious
	}
	_, err := db.Exec(
		`INSERT INTO ai_comment_task_comments(id, content, assistant_response, task_id, tenant_id, workspace_id, created_by_id, execution_mode, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Content, nullStringValue(c.AssistantResponse), c.TaskID, c.TenantID, c.WorkspaceID, c.CreatedByID, c.ExecutionMode, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func nullStringValue(v sql.NullString) interface{} {
	if v.Valid {
		return v.String
	}
	return nil
}

func loadComment(id string) (*AIComment, error) {
	row := db.QueryRow(
		`SELECT id, content, assistant_response, task_id, tenant_id, workspace_id, created_by_id, execution_mode, created_at, updated_at
		 FROM ai_comment_task_comments WHERE id = ?`, id,
	)
	return scanComment(row)
}

func listCommentsByTask(taskID string) ([]AIComment, error) {
	rows, err := db.Query(
		`SELECT id, content, assistant_response, task_id, tenant_id, workspace_id, created_by_id, execution_mode, created_at, updated_at
		 FROM ai_comment_task_comments WHERE task_id = ? ORDER BY created_at ASC, id ASC`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AIComment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func listCommentsByTaskPage(taskID string, p commentPageParams) ([]AIComment, bool, error) {
	limit := p.limit
	if limit < 1 {
		limit = defaultCommentLimit
	}
	query := `SELECT id, content, assistant_response, task_id, tenant_id, workspace_id, created_by_id, execution_mode, created_at, updated_at
		 FROM ai_comment_task_comments WHERE task_id = ?`
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
	out := []AIComment{}
	for rows.Next() {
		c, err := scanComment(rows)
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

func updateAssistantResponse(id, text string) (bool, error) {
	now := time.Now().UTC()
	res, err := db.Exec(
		`UPDATE ai_comment_task_comments SET assistant_response = ?, updated_at = ? WHERE id = ?`,
		text, now, id,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func importComments(rows []AIComment) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	count := 0
	for _, c := range rows {
		if c.ID == "" {
			return count, fmt.Errorf("row %d: id required", count)
		}
		if c.TaskID == "" {
			return count, fmt.Errorf("row %d: task_id required", count)
		}
		if c.CreatedByID == "" {
			return count, fmt.Errorf("row %d: created_by_id required", count)
		}
		if c.Content == "" {
			return count, fmt.Errorf("row %d: content required", count)
		}
		createdAt := c.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		updatedAt := c.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}
		if c.ExecutionMode == "" {
			c.ExecutionMode = executionModeWaitPrevious
		}
		_, err := tx.Exec(
			`INSERT INTO ai_comment_task_comments(id, content, assistant_response, task_id, tenant_id, workspace_id, created_by_id, execution_mode, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
			   content=VALUES(content),
			   assistant_response=VALUES(assistant_response),
			   task_id=VALUES(task_id),
			   tenant_id=VALUES(tenant_id),
			   workspace_id=VALUES(workspace_id),
			   created_by_id=VALUES(created_by_id),
			   execution_mode=VALUES(execution_mode),
			   created_at=VALUES(created_at),
			   updated_at=VALUES(updated_at)`,
			c.ID, c.Content, nullStringValue(c.AssistantResponse), c.TaskID, c.TenantID, c.WorkspaceID, c.CreatedByID, c.ExecutionMode, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanComment(row rowScanner) (*AIComment, error) {
	var c AIComment
	var assistant sql.NullString
	err := row.Scan(&c.ID, &c.Content, &assistant, &c.TaskID, &c.TenantID, &c.WorkspaceID, &c.CreatedByID, &c.ExecutionMode, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.AssistantResponse = assistant
	if c.ExecutionMode == "" {
		c.ExecutionMode = executionModeWaitPrevious
	}
	return &c, nil
}
