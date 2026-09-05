package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"snowflake"
	"taskCloudService/domain"
	"tracelog"
)

var stepFullArchiveMu sync.Mutex

type stepFullObjectRow struct {
	ID          string
	CompanyID   string
	WorkspaceID string
	TaskID      string
	CommentID   string
	JobID       string
	LayerID     string
	ObjectKey   string
	ETag        string
	Bytes       int
	Source      string
	PayloadJSON string
}

func upsertStepFullObjectRow(row stepFullObjectRow) error {
	if db == nil {
		return fmt.Errorf("db not open")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = snowflake.GenerateIDString()
	}
	_, err := db.Exec(
		`INSERT INTO cloud_job_step_full_object (
			id, company_id, workspace_id, task_id, comment_id, job_id, layer_id,
			object_key, etag, bytes, source, payload_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			layer_id = VALUES(layer_id),
			object_key = VALUES(object_key),
			etag = VALUES(etag),
			bytes = VALUES(bytes),
			source = VALUES(source),
			payload_json = VALUES(payload_json),
			company_id = VALUES(company_id),
			updated_at = CURRENT_TIMESTAMP`,
		row.ID, row.CompanyID, row.WorkspaceID, row.TaskID, row.CommentID, row.JobID, row.LayerID,
		row.ObjectKey, row.ETag, row.Bytes, row.Source, nullIfEmpty(row.PayloadJSON),
	)
	return err
}

func getStepFullObjectRow(workspaceID, taskID, commentID, jobID string) (stepFullObjectRow, bool, error) {
	if db == nil {
		return stepFullObjectRow{}, false, fmt.Errorf("db not open")
	}
	var row stepFullObjectRow
	var payload sql.NullString
	err := db.QueryRow(
		`SELECT id, company_id, workspace_id, task_id, comment_id, job_id, layer_id,
		        object_key, etag, bytes, source, payload_json
		 FROM cloud_job_step_full_object
		 WHERE workspace_id=? AND task_id=? AND comment_id=? AND job_id=?`,
		workspaceID, taskID, commentID, jobID,
	).Scan(&row.ID, &row.CompanyID, &row.WorkspaceID, &row.TaskID, &row.CommentID, &row.JobID, &row.LayerID,
		&row.ObjectKey, &row.ETag, &row.Bytes, &row.Source, &payload)
	if err == sql.ErrNoRows {
		return stepFullObjectRow{}, false, nil
	}
	if err != nil {
		return stepFullObjectRow{}, false, err
	}
	if payload.Valid {
		row.PayloadJSON = payload.String
	}
	return row, true, nil
}

// latestStepFullObjectRowForComment 取评论最新归档行；layerID 非空时仅回退到同层，
// 避免按层分文件后把其他层的 jobs 合并进本层对象 key。
func latestStepFullObjectRowForComment(workspaceID, taskID, commentID, layerID string) (stepFullObjectRow, bool, error) {
	if db == nil {
		return stepFullObjectRow{}, false, fmt.Errorf("db not open")
	}
	var row stepFullObjectRow
	var payload sql.NullString
	query := `SELECT id, company_id, workspace_id, task_id, comment_id, job_id, layer_id,
		        object_key, etag, bytes, source, payload_json
		 FROM cloud_job_step_full_object
		 WHERE workspace_id=? AND task_id=? AND comment_id=?`
	args := []any{workspaceID, taskID, commentID}
	if strings.TrimSpace(layerID) != "" {
		query += ` AND layer_id=?`
		args = append(args, strings.TrimSpace(layerID))
	}
	query += ` ORDER BY updated_at DESC LIMIT 1`
	err := db.QueryRow(query, args...).Scan(&row.ID, &row.CompanyID, &row.WorkspaceID, &row.TaskID, &row.CommentID, &row.JobID, &row.LayerID,
		&row.ObjectKey, &row.ETag, &row.Bytes, &row.Source, &payload)
	if err == sql.ErrNoRows {
		return stepFullObjectRow{}, false, nil
	}
	if err != nil {
		return stepFullObjectRow{}, false, err
	}
	if payload.Valid {
		row.PayloadJSON = payload.String
	}
	return row, true, nil
}

func loadStepFullBundleBytes(ctx context.Context, objectKey, payloadJSON string) ([]byte, string, error) {
	if stepFullObjects != nil && strings.TrimSpace(objectKey) != "" {
		raw, found, err := stepFullObjects.Get(ctx, objectKey)
		if err != nil {
			tracelog.LogForwardStage(ctx, "step_full_object_get_err", map[string]any{
				"error": err.Error(), "object_key": objectKey,
			})
		} else if found && len(raw) > 0 {
			return raw, "saas_cos", nil
		}
	}
	if strings.TrimSpace(payloadJSON) != "" {
		return []byte(payloadJSON), "saas_local", nil
	}
	return nil, "", nil
}

func persistStepFullArchive(ctx context.Context, id domain.StepFullIDs, companyID, status string, steps []any) (string, error) {
	stepFullArchiveMu.Lock()
	defer stepFullArchiveMu.Unlock()
	id = id.Normalize()
	if err := id.Validate(); err != nil {
		return "", err
	}
	rule := strings.TrimSpace(stepFullCOSCfg.PathRule)
	id.KeyPrefix = strings.TrimSpace(stepFullCOSCfg.KeyPrefix)
	key, err := domain.RenderStepFullObjectKey(rule, id)
	if err != nil {
		return "", err
	}
	var existing domain.StepFullBundle
	rawOld, _, _ := loadStepFullBundleBytes(ctx, key, "")
	if len(rawOld) == 0 {
		if row, found, err := getStepFullObjectRow(id.WorkspaceID, id.TaskID, id.CommentID, id.JobID); err == nil && found {
			rawOld, _, _ = loadStepFullBundleBytes(ctx, row.ObjectKey, row.PayloadJSON)
		}
	}
	if len(rawOld) == 0 && !strings.Contains(rule, "{jobId}") {
		fallbackLayer := ""
		if strings.Contains(rule, "{layerId}") {
			fallbackLayer = id.LayerID
		}
		if row, found, err := latestStepFullObjectRowForComment(id.WorkspaceID, id.TaskID, id.CommentID, fallbackLayer); err == nil && found {
			rawOld, _, _ = loadStepFullBundleBytes(ctx, row.ObjectKey, row.PayloadJSON)
		}
	}
	if len(rawOld) > 0 {
		if b, err := domain.ParseStepFullBundle(rawOld); err == nil {
			existing = b
		}
	}
	if existing.Jobs == nil {
		existing = domain.EmptyStepFullBundle(id)
	}
	merged := domain.MergeStepFullJob(existing, id, status, steps)
	raw, err := domain.MarshalStepFullBundle(merged)
	if err != nil {
		return "", err
	}
	source := "local"
	etag := ""
	payload := string(raw)
	if stepFullObjects != nil {
		e, putErr := stepFullObjects.Put(ctx, key, raw)
		if putErr != nil {
			tracelog.LogForwardStage(ctx, "step_full_object_put_err", map[string]any{
				"error": putErr.Error(), "object_key": key, "job_id": id.JobID,
			})
		} else {
			etag = e
			if strings.EqualFold(strings.TrimSpace(stepFullCOSCfg.Backend), "cos") {
				source = "cos"
				payload = ""
			}
		}
	}
	if err := upsertStepFullObjectRow(stepFullObjectRow{
		CompanyID: companyID, WorkspaceID: id.WorkspaceID, TaskID: id.TaskID,
		CommentID: id.CommentID, JobID: id.JobID, LayerID: id.LayerID,
		ObjectKey: key, ETag: etag, Bytes: len(raw), Source: source, PayloadJSON: payload,
	}); err != nil {
		return "", err
	}
	eventKey := id.WorkspaceID + ":" + id.TaskID + ":" + id.CommentID + ":" + id.JobID
	if err := publishDomainEvent(ctx, "JobStepFullArchived", map[string]interface{}{
		"workspace_id": id.WorkspaceID,
		"task_id":      id.TaskID,
		"comment_id":   id.CommentID,
		"job_id":       id.JobID,
		"object_key":   key,
		"bytes":        len(raw),
	}, eventKey); err != nil {
		tracelog.LogForwardStage(ctx, "step_full_archive_event_err", map[string]any{
			"error": err.Error(), "job_id": id.JobID,
		})
	}
	tracelog.LogForwardStage(ctx, "step_full_archive_ok", map[string]any{
		"workspace_id": id.WorkspaceID, "task_id": id.TaskID, "comment_id": id.CommentID,
		"job_id": id.JobID, "object_key": key, "bytes": len(raw), "source": source,
	})
	return key, nil
}
