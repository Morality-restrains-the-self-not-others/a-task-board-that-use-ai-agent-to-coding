package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"snowflake"
	"taskCloudService/domain"
	"tracelog"
)

var startupLogArchiveMu sync.Mutex

type startupLogObjectRow struct {
	ID          string
	CompanyID   string
	WorkspaceID string
	TaskID      string
	CommentID   string
	ObjectKey   string
	ETag        string
	Bytes       int
	Source      string
	PayloadJSON string
}

func persistCommentStartupLogBestEffort(ctx context.Context, l CommentContainerBindingLog) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := persistCommentStartupLog(ctx, l); err != nil {
		tracelog.LogForwardStage(ctx, "ccb_startup_log_archive_err", map[string]any{
			"error":        err.Error(),
			"workspace_id": l.WorkspaceID,
			"task_id":      l.TaskID,
			"comment_id":   l.CommentID,
			"log_id":       l.ID,
		})
	}
}

func persistCommentStartupLog(ctx context.Context, l CommentContainerBindingLog) error {
	startupLogArchiveMu.Lock()
	defer startupLogArchiveMu.Unlock()
	id := domain.StartupLogIDs{
		WorkspaceID: l.WorkspaceID,
		TaskID:      l.TaskID,
		CommentID:   l.CommentID,
		KeyPrefix:   strings.TrimSpace(stepFullCOSCfg.KeyPrefix),
	}
	if err := id.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(l.ID) == "" {
		return fmt.Errorf("log id required")
	}
	rule := strings.TrimSpace(stepFullCOSCfg.StartupLogsPathRule)
	key, err := domain.RenderStartupLogObjectKey(rule, id)
	if err != nil {
		return err
	}
	rawOld, _ := loadStartupLogBundleBytes(ctx, key, "")
	if len(rawOld) == 0 {
		if row, found, qerr := getStartupLogObjectRow(id.WorkspaceID, id.TaskID, id.CommentID); qerr == nil && found {
			rawOld, _ = loadStartupLogBundleBytes(ctx, row.ObjectKey, row.PayloadJSON)
		}
	}
	existing := domain.EmptyStartupLogBundle(id)
	if len(rawOld) > 0 {
		if b, perr := domain.ParseStartupLogBundle(rawOld); perr == nil {
			existing = b
		}
	}
	entry := domain.StartupLogEntry{
		ID:          l.ID,
		BindingID:   l.BindingID,
		Stage:       l.Stage,
		Message:     l.Message,
		CreatedAt:   formatCloudUTCJSON(l.CreatedAt),
		CompanyID:   l.CompanyID,
		WorkspaceID: l.WorkspaceID,
		TaskID:      l.TaskID,
		CommentID:   l.CommentID,
	}
	if strings.TrimSpace(entry.CreatedAt) == "" && !l.CreatedAt.IsZero() {
		entry.CreatedAt = l.CreatedAt.UTC().Format("2006-01-02 15:04:05")
	}
	merged := domain.MergeStartupLogEntry(existing, id, entry)
	raw, err := domain.MarshalStartupLogBundle(merged)
	if err != nil {
		return err
	}
	source := "local"
	etag := ""
	payload := string(raw)
	putOK := false
	if stepFullObjects != nil {
		e, putErr := stepFullObjects.Put(ctx, key, raw)
		if putErr != nil {
			tracelog.LogForwardStage(ctx, "ccb_startup_log_object_put_err", map[string]any{
				"error": putErr.Error(), "object_key": key, "log_id": l.ID,
			})
		} else {
			putOK = true
			etag = e
			if strings.EqualFold(strings.TrimSpace(stepFullCOSCfg.Backend), "cos") {
				source = "cos"
				payload = ""
			}
		}
	}
	if err := upsertStartupLogObjectRow(startupLogObjectRow{
		CompanyID: l.CompanyID, WorkspaceID: id.WorkspaceID, TaskID: id.TaskID,
		CommentID: id.CommentID, ObjectKey: key, ETag: etag, Bytes: len(raw),
		Source: source, PayloadJSON: payload,
	}); err != nil {
		return err
	}
	eventKey := id.WorkspaceID + ":" + id.TaskID + ":" + id.CommentID + ":" + l.ID
	if err := publishDomainEvent(ctx, "CommentStartupLogArchived", map[string]interface{}{
		"workspace_id": id.WorkspaceID,
		"task_id":      id.TaskID,
		"comment_id":   id.CommentID,
		"log_id":       l.ID,
		"object_key":   key,
		"bytes":        len(raw),
	}, eventKey); err != nil {
		tracelog.LogForwardStage(ctx, "ccb_startup_log_archive_event_err", map[string]any{
			"error": err.Error(), "log_id": l.ID,
		})
	}
	if putOK {
		if delErr := deleteCCBLogShardRow(id.WorkspaceID, l.ID); delErr != nil {
			tracelog.LogForwardStage(ctx, "ccb_startup_log_shard_delete_err", map[string]any{
				"error": delErr.Error(), "workspace_id": id.WorkspaceID, "log_id": l.ID,
			})
		}
	}
	tracelog.LogForwardStage(ctx, "ccb_startup_log_archive_ok", map[string]any{
		"workspace_id": id.WorkspaceID, "task_id": id.TaskID, "comment_id": id.CommentID,
		"log_id": l.ID, "object_key": key, "bytes": len(raw), "source": source,
	})
	return nil
}

func loadStartupLogBundleBytes(ctx context.Context, objectKey, payloadJSON string) ([]byte, string) {
	if stepFullObjects != nil && strings.TrimSpace(objectKey) != "" {
		raw, found, err := stepFullObjects.Get(ctx, objectKey)
		if err != nil {
			tracelog.LogForwardStage(ctx, "ccb_startup_log_object_get_err", map[string]any{
				"error": err.Error(), "object_key": objectKey,
			})
		} else if found && len(raw) > 0 {
			return raw, "saas_cos"
		}
	}
	if strings.TrimSpace(payloadJSON) != "" {
		return []byte(payloadJSON), "saas_local"
	}
	return nil, ""
}

func upsertStartupLogObjectRow(row startupLogObjectRow) error {
	if db == nil {
		return fmt.Errorf("db not open")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = snowflake.GenerateIDString()
	}
	_, err := db.Exec(
		`INSERT INTO cloud_comment_startup_log_object (
			id, company_id, workspace_id, task_id, comment_id,
			object_key, etag, bytes, source, payload_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			object_key = VALUES(object_key),
			etag = VALUES(etag),
			bytes = VALUES(bytes),
			source = VALUES(source),
			payload_json = VALUES(payload_json),
			company_id = VALUES(company_id),
			updated_at = CURRENT_TIMESTAMP`,
		row.ID, row.CompanyID, row.WorkspaceID, row.TaskID, row.CommentID,
		row.ObjectKey, row.ETag, row.Bytes, row.Source, nullIfEmpty(row.PayloadJSON),
	)
	return err
}

func getStartupLogObjectRow(workspaceID, taskID, commentID string) (startupLogObjectRow, bool, error) {
	if db == nil {
		return startupLogObjectRow{}, false, fmt.Errorf("db not open")
	}
	var row startupLogObjectRow
	var payload sql.NullString
	err := db.QueryRow(
		`SELECT id, company_id, workspace_id, task_id, comment_id,
		        object_key, etag, bytes, source, payload_json
		 FROM cloud_comment_startup_log_object
		 WHERE workspace_id=? AND task_id=? AND comment_id=?`,
		workspaceID, taskID, commentID,
	).Scan(&row.ID, &row.CompanyID, &row.WorkspaceID, &row.TaskID, &row.CommentID,
		&row.ObjectKey, &row.ETag, &row.Bytes, &row.Source, &payload)
	if err == sql.ErrNoRows {
		return startupLogObjectRow{}, false, nil
	}
	if err != nil {
		return startupLogObjectRow{}, false, err
	}
	if payload.Valid {
		row.PayloadJSON = payload.String
	}
	return row, true, nil
}

func listStartupLogObjectRowsForTask(workspaceID, companyID, taskID string) ([]startupLogObjectRow, error) {
	if db == nil {
		return nil, fmt.Errorf("db not open")
	}
	rows, err := db.Query(
		`SELECT id, company_id, workspace_id, task_id, comment_id,
		        object_key, etag, bytes, source, payload_json
		 FROM cloud_comment_startup_log_object
		 WHERE workspace_id=? AND company_id=? AND task_id=?`,
		workspaceID, companyID, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]startupLogObjectRow, 0)
	for rows.Next() {
		var row startupLogObjectRow
		var payload sql.NullString
		if err := rows.Scan(&row.ID, &row.CompanyID, &row.WorkspaceID, &row.TaskID, &row.CommentID,
			&row.ObjectKey, &row.ETag, &row.Bytes, &row.Source, &payload); err != nil {
			return nil, err
		}
		if payload.Valid {
			row.PayloadJSON = payload.String
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func mergeStartupLogsFromObjectStore(workspaceID, companyID, taskID string, mysqlLogs []CommentContainerBindingLog) []CommentContainerBindingLog {
	ctx := context.Background()
	rows, err := listStartupLogObjectRowsForTask(workspaceID, companyID, taskID)
	if err != nil {
		tracelog.LogForwardStage(ctx, "ccb_startup_log_pointer_list_err", map[string]any{
			"error": err.Error(), "workspace_id": workspaceID, "task_id": taskID,
		})
		return mysqlLogs
	}
	byID := map[string]CommentContainerBindingLog{}
	for _, row := range rows {
		raw, _ := loadStartupLogBundleBytes(ctx, row.ObjectKey, row.PayloadJSON)
		if len(raw) == 0 {
			continue
		}
		bundle, perr := domain.ParseStartupLogBundle(raw)
		if perr != nil {
			continue
		}
		for _, e := range bundle.Logs {
			if strings.TrimSpace(e.ID) == "" {
				continue
			}
			byID[e.ID] = commentLogFromBundleEntry(e, workspaceID, companyID, taskID, row.CommentID)
		}
	}
	for _, l := range mysqlLogs {
		if strings.TrimSpace(l.ID) == "" {
			continue
		}
		if _, ok := byID[l.ID]; ok {
			continue
		}
		byID[l.ID] = l
	}
	out := make([]CommentContainerBindingLog, 0, len(byID))
	for _, l := range byID {
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func commentLogFromBundleEntry(e domain.StartupLogEntry, workspaceID, companyID, taskID, commentID string) CommentContainerBindingLog {
	ws := strings.TrimSpace(e.WorkspaceID)
	if ws == "" {
		ws = workspaceID
	}
	co := strings.TrimSpace(e.CompanyID)
	if co == "" {
		co = companyID
	}
	tid := strings.TrimSpace(e.TaskID)
	if tid == "" {
		tid = taskID
	}
	cid := strings.TrimSpace(e.CommentID)
	if cid == "" {
		cid = commentID
	}
	created := domain.ParseStartupLogCreatedAt(e.CreatedAt)
	if created.IsZero() {
		created = parseCloudUTCDateTime(e.CreatedAt)
	}
	if created.IsZero() {
		created = time.Now().UTC()
	}
	return CommentContainerBindingLog{
		ID: e.ID, WorkspaceID: ws, CompanyID: co, TaskID: tid, CommentID: cid,
		BindingID: e.BindingID, Stage: e.Stage, Message: e.Message, CreatedAt: created,
	}
}
