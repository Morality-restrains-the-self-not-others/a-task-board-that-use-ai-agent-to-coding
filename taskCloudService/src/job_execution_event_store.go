package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"snowflake"
	"tracelog"
)

type jobExecutionEventRow struct {
	ID              string
	CompanyID       string
	WorkspaceID     string
	TaskID          string
	CommentID       string
	JobID           string
	Seq             int
	Phase           string
	Message         string
	StepNumber      int
	DeliverySummary string
	StepState       string
	JobStatus       string
	LayerID         string
	EventJSON       string
	CreatedAt       time.Time
}

func shouldPersistJobStreamPhase(phase string) bool {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "step", "start", "running", "completed", "failed", "interrupted":
		return true
	default:
		return false
	}
}

func insertJobExecutionEvent(row jobExecutionEventRow) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db not open")
	}
	row.TaskID = strings.TrimSpace(row.TaskID)
	row.JobID = strings.TrimSpace(row.JobID)
	row.Phase = strings.TrimSpace(row.Phase)
	if row.TaskID == "" || row.JobID == "" {
		return false, fmt.Errorf("task_id and job_id required")
	}
	if !shouldPersistJobStreamPhase(row.Phase) {
		return false, nil
	}
	table, err := jobExecutionEventTable(row.WorkspaceID)
	if err != nil {
		return false, err
	}
	var existing string
	err = db.QueryRow(
		`SELECT id FROM `+table+` WHERE task_id = ? AND job_id = ? AND seq = ? LIMIT 1`,
		row.TaskID, row.JobID, row.Seq,
	).Scan(&existing)
	if err == nil && existing != "" {
		return false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = snowflake.GenerateIDString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	eventJSON := row.EventJSON
	if strings.TrimSpace(eventJSON) == "" {
		eventJSON = "null"
	}
	_, err = db.Exec(
		`INSERT INTO `+table+` (
			id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message,
			step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID, row.CompanyID, row.WorkspaceID, row.TaskID, row.CommentID, row.JobID, row.Seq, row.Phase, row.Message,
		row.StepNumber, row.DeliverySummary, row.StepState, row.JobStatus, row.LayerID, eventJSON, row.CreatedAt,
	)
	if err != nil {
		return false, err
	}
	tracelog.LogForwardStage(context.Background(), "job_execution_event_insert", map[string]any{
		"task_id": row.TaskID,
		"job_id":  row.JobID,
		"seq":     row.Seq,
		"phase":   row.Phase,
	})
	return true, nil
}

func listJobExecutionEvents(workspaceID, taskID, commentID, jobID string) ([]jobExecutionEventRow, error) {
	if db == nil {
		return nil, fmt.Errorf("db not open")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	taskID = strings.TrimSpace(taskID)
	jobID = strings.TrimSpace(jobID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id required")
	}
	if taskID == "" {
		return nil, fmt.Errorf("task_id required")
	}
	table, err := jobExecutionEventTable(workspaceID)
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if jobID != "" && strings.TrimSpace(commentID) != "" {
		rows, err = db.Query(
			`SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message,
			        step_number, delivery_summary, step_state, job_status, layer_id, IFNULL(event_json,''), created_at
			 FROM `+table+`
			 WHERE workspace_id = ? AND task_id = ? AND job_id = ? AND (comment_id = ? OR comment_id = '')
			 ORDER BY seq ASC, created_at ASC`,
			workspaceID, taskID, jobID, strings.TrimSpace(commentID),
		)
	} else if jobID != "" {
		rows, err = db.Query(
			`SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message,
			        step_number, delivery_summary, step_state, job_status, layer_id, IFNULL(event_json,''), created_at
			 FROM `+table+`
			 WHERE workspace_id = ? AND task_id = ? AND job_id = ?
			 ORDER BY seq ASC, created_at ASC`,
			workspaceID, taskID, jobID,
		)
	} else if strings.TrimSpace(commentID) != "" {
		rows, err = db.Query(
			`SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message,
			        step_number, delivery_summary, step_state, job_status, layer_id, IFNULL(event_json,''), created_at
			 FROM `+table+`
			 WHERE workspace_id = ? AND task_id = ? AND comment_id = ?
			 ORDER BY created_at DESC, seq DESC LIMIT 500`,
			workspaceID, taskID, strings.TrimSpace(commentID),
		)
	} else {
		return nil, fmt.Errorf("job_id or comment_id required")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []jobExecutionEventRow
	for rows.Next() {
		var r jobExecutionEventRow
		var msg sql.NullString
		var summary sql.NullString
		var eventJSON sql.NullString
		if err := rows.Scan(
			&r.ID, &r.CompanyID, &r.WorkspaceID, &r.TaskID, &r.CommentID, &r.JobID, &r.Seq, &r.Phase, &msg,
			&r.StepNumber, &summary, &r.StepState, &r.JobStatus, &r.LayerID, &eventJSON, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.Message = msg.String
		r.DeliverySummary = summary.String
		r.EventJSON = eventJSON.String
		out = append(out, r)
	}
	return out, rows.Err()
}

func latestJobIDForComment(workspaceID, taskID, commentID string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("db not open")
	}
	table, err := jobExecutionEventTable(workspaceID)
	if err != nil {
		return "", err
	}
	var jobID string
	err = db.QueryRow(
		`SELECT job_id FROM `+table+`
		 WHERE task_id = ? AND comment_id = ?
		 ORDER BY created_at DESC, seq DESC LIMIT 1`,
		strings.TrimSpace(taskID), strings.TrimSpace(commentID),
	).Scan(&jobID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return jobID, err
}

func reconstructJobExecutionLog(events []jobExecutionEventRow, jobID string, afterStep, limit int) map[string]any {
	if afterStep < 0 {
		afterStep = 0
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	job := map[string]any{
		"id":             jobID,
		"output_omitted": true,
		"output_chars":   0,
		"source":         "saas_db",
	}
	var steps []map[string]any
	maxStep := 0
	for _, ev := range events {
		if ev.LayerID != "" {
			job["layer_id"] = ev.LayerID
		}
		ph := strings.ToLower(strings.TrimSpace(ev.Phase))
		if ph == "running" || ph == "start" || ph == "completed" || ph == "failed" || ph == "interrupted" {
			job["status"] = ph
		}
		if ev.JobStatus != "" {
			job["status"] = ev.JobStatus
		}
		if ph != "step" {
			continue
		}
		n := ev.StepNumber
		if n <= 0 {
			n = len(steps) + 1
		}
		if n > maxStep {
			maxStep = n
		}
		if n <= afterStep {
			continue
		}
		summary := ev.DeliverySummary
		if summary == "" {
			summary = ev.Message
		}
		steps = append(steps, map[string]any{
			"step_number":      n,
			"delivery_summary": summary,
			"state":            ev.StepState,
			"message":          ev.Message,
		})
	}
	if len(steps) > limit {
		steps = steps[:limit]
	}
	nextAfter := afterStep
	for _, s := range steps {
		if n, ok := s["step_number"].(int); ok && n > nextAfter {
			nextAfter = n
		}
	}
	hasMore := false
	totalSteps := 0
	for _, ev := range events {
		if strings.EqualFold(ev.Phase, "step") {
			totalSteps++
			n := ev.StepNumber
			if n > afterStep && n > nextAfter {
				hasMore = true
			}
		}
	}
	var next any
	if hasMore {
		next = nextAfter
	}
	return map[string]any{
		"job": job,
		"steps": map[string]any{
			"steps":           steps,
			"total_steps":     totalSteps,
			"after_step":      afterStep,
			"next_after_step": next,
			"has_more":        hasMore,
			"source":          "saas_db",
		},
		"source": "saas_db",
	}
}

func eventJSONString(v any) string {
	if v == nil {
		return "null"
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(raw)
}
