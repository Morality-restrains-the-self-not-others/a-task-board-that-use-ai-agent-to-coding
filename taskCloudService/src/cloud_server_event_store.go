package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"snowflake"
	"strings"
	"time"
)

const (
	cloudServerEventTypeStart        = "start"
	cloudServerEventStatusPending    = "pending"
	cloudServerEventStatusError      = "error"
	cloudServerEventStatusSuccess    = "success"
	cloudServerEventStatusProcessing = "processing"
)

type CloudServerEvent struct {
	ID              string
	CompanyID       string
	WorkspaceID     string
	TaskID          string
	CommentID       string
	CompanyMemberID string
	EventType       string
	EventData       map[string]interface{}
	Status          string
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func pruneFailedStartEvents(companyID, taskID string) (int64, error) {
	res, err := db.Exec(`
		DELETE FROM cloud_server_events
		WHERE company_id = ? AND task_id = ? AND event_type = ? AND status = ?
	`, companyID, taskID, cloudServerEventTypeStart, cloudServerEventStatusError)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func insertPendingStartEvent(
	companyID, workspaceID, taskID, memberID, commentID string,
	eventData map[string]interface{},
) (eventID string, merged map[string]interface{}, err error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	memberID = trim(memberID)
	commentID = trim(commentID)
	if companyID == "" || workspaceID == "" || taskID == "" {
		return "", nil, fmt.Errorf("company_id, workspace_id and task_id required")
	}

	pruned, err := pruneFailedStartEvents(companyID, taskID)
	if err != nil {
		return "", nil, err
	}
	if pruned > 0 {
		log.Printf("[taskCloudService] pruned stale failed start events: company_id=%s task_id=%s count=%d",
			companyID, taskID, pruned)
	}

	merged = cloneEventDataMap(eventData)
	raw, err := json.Marshal(merged)
	if err != nil {
		return "", nil, fmt.Errorf("marshal event_data: %w", err)
	}

	eventID = snowflake.GenerateIDString()
	nowUTC := formatMySQLUTCDateTime(time.Now().UTC())
	_, err = db.Exec(`
		INSERT INTO cloud_server_events
			(id, company_id, workspace_id, task_id, company_member_id, event_type, event_data, status, comment_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, eventID, companyID, workspaceID, taskID, memberID, cloudServerEventTypeStart, string(raw), cloudServerEventStatusPending, commentID, nowUTC, nowUTC)
	if err != nil {
		return "", nil, err
	}
	return eventID, merged, nil
}

func deletePendingStartEvent(eventID, companyID string) error {
	eventID = trim(eventID)
	companyID = trim(companyID)
	if eventID == "" || companyID == "" {
		return fmt.Errorf("event_id and company_id required")
	}
	_, err := db.Exec(`
		DELETE FROM cloud_server_events
		WHERE id = ? AND company_id = ? AND status = ?
	`, eventID, companyID, cloudServerEventStatusPending)
	return err
}

// errCloudServerEventStatusConflict 标记带 from_status 守卫的状态迁移失败：
// 目标行不存在或当前状态已不是 from_status（并发消费者已抢先 claim / 推进）。
// taskEvents 侧据此把「丢失 claim」当幂等跳过，而非误投 DLT。
var errCloudServerEventStatusConflict = errors.New("cloud server event status changed")

func updateCloudServerEventStatus(eventID, status, errMsg, fromStatus string) error {
	eventID = trim(eventID)
	status = trim(status)
	if eventID == "" || status == "" {
		return fmt.Errorf("event_id and status required")
	}
	var res sql.Result
	var err error
	switch {
	case strings.TrimSpace(errMsg) != "" && fromStatus != "":
		res, err = db.Exec(`
			UPDATE cloud_server_events
			SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND status = ?
		`, status, errMsg, eventID, fromStatus)
	case strings.TrimSpace(errMsg) != "":
		res, err = db.Exec(`
			UPDATE cloud_server_events
			SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, status, errMsg, eventID)
	case fromStatus != "":
		res, err = db.Exec(`
			UPDATE cloud_server_events
			SET status = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND status = ?
		`, status, eventID, fromStatus)
	default:
		res, err = db.Exec(`
			UPDATE cloud_server_events
			SET status = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, status, eventID)
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if fromStatus != "" {
			return errCloudServerEventStatusConflict
		}
		return fmt.Errorf("cloud server event %s not found", eventID)
	}
	return nil
}

func updateCloudServerEventData(eventID string, data map[string]interface{}) error {
	eventID = trim(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id required")
	}
	if data == nil {
		data = map[string]interface{}{}
	}

	var existingRaw string
	err := db.QueryRow(`SELECT event_data FROM cloud_server_events WHERE id = ?`, eventID).Scan(&existingRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("cloud server event %s not found", eventID)
		}
		return err
	}

	merged, err := mergeEventDataJSON(existingRaw, data)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return fmt.Errorf("marshal event_data: %w", err)
	}

	res, err := db.Exec(`
		UPDATE cloud_server_events
		SET event_data = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, string(raw), eventID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("cloud server event %s not found", eventID)
	}
	return nil
}

func latestPendingStartEvent(companyID, taskID string) (id, status string, err error) {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" {
		return "", "", fmt.Errorf("company_id and task_id required")
	}
	err = db.QueryRow(`
		SELECT id, status FROM cloud_server_events
		WHERE company_id = ? AND task_id = ? AND event_type = ? AND status = ?
		ORDER BY created_at DESC LIMIT 1
	`, companyID, taskID, cloudServerEventTypeStart, cloudServerEventStatusPending).Scan(&id, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("pending start event not found for task %s", taskID)
		}
		return "", "", err
	}
	return id, status, nil
}

func loadCloudServerEvent(eventID string) (*CloudServerEvent, error) {
	eventID = trim(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("event_id required")
	}
	row := db.QueryRow(`
		SELECT id, company_id, workspace_id, task_id, company_member_id, event_type,
			event_data, status, COALESCE(error_message,''), created_at, updated_at, COALESCE(comment_id,'')
		FROM cloud_server_events WHERE id = ?
	`, eventID)
	return scanCloudServerEvent(row)
}

func loadLatestStartEvent(companyID, taskID, optionalEventID, optionalCommentID string) (*CloudServerEvent, error) {
	companyID = trim(companyID)
	taskID = trim(taskID)
	optionalEventID = trim(optionalEventID)
	optionalCommentID = trim(optionalCommentID)
	if companyID == "" || taskID == "" {
		return nil, fmt.Errorf("company_id and task_id required")
	}

	q := `
		SELECT id, company_id, workspace_id, task_id, company_member_id, event_type,
			event_data, status, COALESCE(error_message,''), created_at, updated_at, COALESCE(comment_id,'')
		FROM cloud_server_events
		WHERE company_id = ? AND task_id = ? AND event_type = ?`
	args := []interface{}{companyID, taskID, cloudServerEventTypeStart}
	if optionalEventID != "" {
		q += ` AND id = ?`
		args = append(args, optionalEventID)
	}
	if optionalCommentID != "" {
		q += ` AND comment_id = ?`
		args = append(args, optionalCommentID)
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT 1`

	row := db.QueryRow(q, args...)
	ev, err := scanCloudServerEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("start event not found for task %s", taskID)
		}
		return nil, err
	}
	return ev, nil
}

func scanCloudServerEvent(row *sql.Row) (*CloudServerEvent, error) {
	var ev CloudServerEvent
	var eventDataRaw, created, updated string
	err := row.Scan(
		&ev.ID, &ev.CompanyID, &ev.WorkspaceID, &ev.TaskID, &ev.CompanyMemberID, &ev.EventType,
		&eventDataRaw, &ev.Status, &ev.ErrorMessage, &created, &updated, &ev.CommentID,
	)
	if err != nil {
		return nil, err
	}
	ev.EventData, err = parseEventDataJSON(eventDataRaw)
	if err != nil {
		return nil, err
	}
	ev.CreatedAt = parseCloudUTCDateTime(created)
	ev.UpdatedAt = parseCloudUTCDateTime(updated)
	return &ev, nil
}

func parseEventDataJSON(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}, nil
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("parse event_data: %w", err)
	}
	return out, nil
}

func mergeEventDataJSON(existingRaw string, patch map[string]interface{}) (map[string]interface{}, error) {
	merged, err := parseEventDataJSON(existingRaw)
	if err != nil {
		return nil, err
	}
	for k, v := range patch {
		merged[k] = v
	}
	return merged, nil
}

func importCloudServerEvents(rows []map[string]interface{}) (int, error) {
	count := 0
	for _, row := range rows {
		id := strField(row, "id")
		if id == "" {
			continue
		}
		eventData := strField(row, "event_data")
		if eventData == "" {
			if raw, ok := row["event_data"].(map[string]interface{}); ok {
				b, err := json.Marshal(raw)
				if err != nil {
					return count, err
				}
				eventData = string(b)
			} else {
				eventData = "{}"
			}
		}
		_, err := db.Exec(`REPLACE INTO cloud_server_events
			(id, company_id, workspace_id, task_id, company_member_id, event_type, event_data, status, error_message, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
			id,
			strField(row, "company_id"),
			strField(row, "workspace_id"),
			strField(row, "task_id"),
			strField(row, "company_member_id"),
			strField(row, "event_type"),
			eventData,
			strField(row, "status"),
			strField(row, "error_message"),
			strField(row, "created_at"),
			strField(row, "updated_at"),
		)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func cloneEventDataMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
