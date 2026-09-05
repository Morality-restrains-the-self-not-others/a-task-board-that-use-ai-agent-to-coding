package main

import (
	"context"
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	scheduleHistoryEventRhythmSaved       = "rhythm_saved"
	scheduleHistoryEventMemberEnqueued    = "member_enqueued"
	scheduleHistoryEventMemberDequeued    = "member_dequeued"
	scheduleHistoryEventMemberStarted     = "member_started"
	scheduleHistoryEventWindowEntered     = "window_entered"
	scheduleHistoryEventWindowExited      = "window_exited"
	scheduleHistoryEventAutoCloseWarned   = "auto_close_warned"
	scheduleHistoryEventAutoCloseReleased = "auto_close_released"
)

type scheduleHistoryInput struct {
	TenantID    string
	WorkspaceID string
	EventType   string
	TaskID      string
	Message     string
	ActorUserID string
}

func appendScheduleHistory(in scheduleHistoryInput) {
	if db == nil {
		return
	}
	ws := strings.TrimSpace(in.WorkspaceID)
	et := strings.TrimSpace(in.EventType)
	if ws == "" || et == "" {
		return
	}
	msg := strings.TrimSpace(in.Message)
	if len(msg) > 512 {
		msg = msg[:512]
	}
	id := genID("qsh")
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO task_queued_schedule_history
		(id, tenant_id, workspace_id, event_type, task_id, message, actor_user_id, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, strings.TrimSpace(in.TenantID), ws, et, strings.TrimSpace(in.TaskID), msg, strings.TrimSpace(in.ActorUserID), now)
	if err != nil {
		log.Printf("[taskTaskService] event=schedule_history_append_failed workspace_id=%s event_type=%s err=%v", ws, et, err)
	}
}

func listScheduleHistory(tenantID, workspaceID, cursor string, limit int) (items []map[string]interface{}, nextCursor string, hasMore bool, err error) {
	if db == nil {
		return []map[string]interface{}{}, "", false, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	ws := strings.TrimSpace(workspaceID)
	tid := strings.TrimSpace(tenantID)
	args := []interface{}{ws, tid}
	where := `h.workspace_id=? AND h.tenant_id=?`
	if curTs, curID, ok := parseScheduleHistoryCursor(cursor); ok {
		where += ` AND (h.created_at < ? OR (h.created_at = ? AND h.id < ?))`
		args = append(args, curTs, curTs, curID)
	}
	args = append(args, limit+1)
	q := `SELECT h.id, h.event_type, h.task_id, h.message, h.actor_user_id, h.created_at, COALESCE(t.title,'')
		FROM task_queued_schedule_history h
		LEFT JOIN task_tasks t ON t.id = h.task_id
		WHERE ` + where + `
		ORDER BY h.created_at DESC, h.id DESC
		LIMIT ?`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()
	out := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		var id, et, taskID, msg, actor string
		var created time.Time
		var title string
		if err := rows.Scan(&id, &et, &taskID, &msg, &actor, &created, &title); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":            id,
			"event_type":    et,
			"task_id":       taskID,
			"task_title":    title,
			"message":       msg,
			"actor_user_id": actor,
			"created_at":    created.UTC().Format(time.RFC3339),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}
	if len(out) > limit {
		hasMore = true
		out = out[:limit]
	}
	if hasMore && len(out) > 0 {
		last := out[len(out)-1]
		nextCursor = formatScheduleHistoryCursor(last["created_at"].(string), last["id"].(string))
	}
	return out, nextCursor, hasMore, nil
}

func formatScheduleHistoryCursor(createdAt, id string) string {
	return createdAt + "|" + id
}

func parseScheduleHistoryCursor(raw string) (ts time.Time, id string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, "", false
	}
	i := strings.LastIndex(raw, "|")
	if i <= 0 || i >= len(raw)-1 {
		return time.Time{}, "", false
	}
	t, err := time.Parse(time.RFC3339, raw[:i])
	if err != nil {
		return time.Time{}, "", false
	}
	return t.UTC(), raw[i+1:], true
}

func loadLastInWindow(workspaceID string) (inWindow bool, set bool) {
	if db == nil {
		return false, false
	}
	var v sql.NullInt64
	err := db.QueryRow(`SELECT last_in_window FROM workspace_schedule_rhythms WHERE workspace_id=?`, workspaceID).Scan(&v)
	if err != nil || !v.Valid {
		return false, false
	}
	return v.Int64 != 0, true
}

func setLastInWindow(workspaceID string, inWindow bool) {
	if db == nil {
		return
	}
	flag := 0
	if inWindow {
		flag = 1
	}
	_, _ = db.Exec(`UPDATE workspace_schedule_rhythms SET last_in_window=? WHERE workspace_id=?`, flag, workspaceID)
}

func recordWorkspaceWindowTransition(r *workspaceScheduleRhythm, inWindow bool) {
	if r == nil {
		return
	}
	last, ok := loadLastInWindow(r.WorkspaceID)
	if ok && last == inWindow {
		return
	}
	et := scheduleHistoryEventWindowExited
	evName := "WorkspaceScheduleWindowExited"
	msg := deferredReasonWorkspace(r)
	if inWindow {
		et = scheduleHistoryEventWindowEntered
		evName = "WorkspaceScheduleWindowEntered"
		msg = "进入允许运行时段"
	}
	appendScheduleHistory(scheduleHistoryInput{
		TenantID:    r.TenantID,
		WorkspaceID: r.WorkspaceID,
		EventType:   et,
		Message:     msg,
	})
	_ = publishDomainEvent(context.Background(), evName, map[string]interface{}{
		"workspace_id": r.WorkspaceID,
		"tenant_id":    r.TenantID,
		"in_window":    inWindow,
	}, r.WorkspaceID)
	setLastInWindow(r.WorkspaceID, inWindow)
}

func clampHistoryLimit(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 20
	}
	if n > 50 {
		return 50
	}
	return n
}
