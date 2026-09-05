package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"snowflake"
	"taskTaskService/domain"
	"tracelog"
)

type taskRevisionRow struct {
	ID            string
	TenantID      string
	WorkspaceID   string
	TaskID        string
	VersionNum    int
	Title         string
	Description   string
	ActorUserID   string
	ChangedFields string
	CreatedAt     time.Time
}

func insertTaskRevisionTx(tx *sql.Tx, row taskRevisionRow) error {
	if tx == nil {
		return fmt.Errorf("insertTaskRevisionTx: nil tx")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = snowflake.GenerateIDString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	_, err := tx.Exec(
		`INSERT INTO task_revision(id,tenant_id,workspace_id,task_id,version_num,title,description,actor_user_id,changed_fields,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.TenantID, row.WorkspaceID, row.TaskID, row.VersionNum,
		row.Title, row.Description, row.ActorUserID, row.ChangedFields, row.CreatedAt,
	)
	if err != nil {
		log.Printf("[taskTaskService] event=task_revision_insert_failed task_id=%s tenant_id=%s err=%v",
			row.TaskID, row.TenantID, err)
	}
	return err
}

var insertTaskRevisionTxFn = insertTaskRevisionTx

func maxTaskRevisionVersion(tx *sql.Tx, taskID, tenantID string) (int, error) {
	var max sql.NullInt64
	err := tx.QueryRow(
		`SELECT MAX(version_num) FROM task_revision WHERE task_id=? AND tenant_id=?`,
		taskID, tenantID,
	).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64), nil
}

// persistTaskRevisionV1 writes the create snapshot inside the live-row transaction.
func persistTaskRevisionV1(tx *sql.Tx, tenantID, workspaceID, taskID, title, description, actorUserID string, now time.Time) (string, error) {
	if !domain.ShouldRecordCreate() {
		return "", nil
	}
	id := snowflake.GenerateIDString()
	row := taskRevisionRow{
		ID:            id,
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		TaskID:        taskID,
		VersionNum:    domain.NextVersion(0),
		Title:         title,
		Description:   description,
		ActorUserID:   actorUserID,
		ChangedFields: domain.CreateChangedFields(),
		CreatedAt:     now,
	}
	if err := insertTaskRevisionTxFn(tx, row); err != nil {
		return "", err
	}
	log.Printf("[taskTaskService] event=task_revision_recorded task_id=%s revision_id=%s version_num=%d actor_user_id=%s changed_fields=%s",
		taskID, id, row.VersionNum, actorUserID, row.ChangedFields)
	return id, nil
}

// persistTaskRevisionIfChanged appends a snapshot when versioned fields actually changed.
func persistTaskRevisionIfChanged(tx *sql.Tx, tenantID, workspaceID, taskID, actorUserID string, old, new domain.VersionedTaskContent, now time.Time) (revisionID string, versionNum int, changed []string, recorded bool, err error) {
	changed, record := domain.DiffVersionedTask(old, new)
	if !record {
		return "", 0, nil, false, nil
	}
	max, err := maxTaskRevisionVersion(tx, taskID, tenantID)
	if err != nil {
		return "", 0, changed, false, err
	}
	id := snowflake.GenerateIDString()
	ver := domain.NextVersion(max)
	row := taskRevisionRow{
		ID:            id,
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		TaskID:        taskID,
		VersionNum:    ver,
		Title:         domain.NormalizeText(new.Title),
		Description:   new.Description,
		ActorUserID:   actorUserID,
		ChangedFields: domain.JoinChangedFields(changed),
		CreatedAt:     now,
	}
	if err := insertTaskRevisionTxFn(tx, row); err != nil {
		return "", 0, changed, false, err
	}
	log.Printf("[taskTaskService] event=task_revision_recorded task_id=%s revision_id=%s version_num=%d actor_user_id=%s changed_fields=%s",
		taskID, id, ver, actorUserID, row.ChangedFields)
	return id, ver, changed, true, nil
}

func listTaskRevisions(taskID, tenantID string, limit, offset int) ([]taskRevisionRow, int, error) {
	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM task_revision WHERE task_id=? AND tenant_id=?`,
		taskID, tenantID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(
		`SELECT id,tenant_id,workspace_id,task_id,version_num,title,COALESCE(description,''),actor_user_id,changed_fields,created_at
		 FROM task_revision WHERE task_id=? AND tenant_id=?
		 ORDER BY version_num DESC LIMIT ? OFFSET ?`,
		taskID, tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []taskRevisionRow
	for rows.Next() {
		var r taskRevisionRow
		if err := rows.Scan(&r.ID, &r.TenantID, &r.WorkspaceID, &r.TaskID, &r.VersionNum,
			&r.Title, &r.Description, &r.ActorUserID, &r.ChangedFields, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func getTaskRevision(taskID, tenantID, revisionID string) (*taskRevisionRow, error) {
	var r taskRevisionRow
	err := db.QueryRow(
		`SELECT id,tenant_id,workspace_id,task_id,version_num,title,COALESCE(description,''),actor_user_id,changed_fields,created_at
		 FROM task_revision WHERE id=? AND task_id=? AND tenant_id=?`,
		revisionID, taskID, tenantID,
	).Scan(&r.ID, &r.TenantID, &r.WorkspaceID, &r.TaskID, &r.VersionNum,
		&r.Title, &r.Description, &r.ActorUserID, &r.ChangedFields, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func taskRevisionListJSON(r taskRevisionRow) map[string]interface{} {
	return map[string]interface{}{
		"id":             r.ID,
		"version_num":    r.VersionNum,
		"title":          r.Title,
		"description":    domain.TruncateRunes(r.Description, 200),
		"actor_user_id":  r.ActorUserID,
		"changed_fields": r.ChangedFields,
		"created_at":     r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func taskRevisionDetailJSON(r taskRevisionRow) map[string]interface{} {
	m := taskRevisionListJSON(r)
	m["description"] = r.Description
	m["tenant_id"] = r.TenantID
	m["workspace_id"] = r.WorkspaceID
	m["task_id"] = r.TaskID
	return m
}

func logRevisionAuthDenied(traceID, userID, taskID, reason string) {
	tracelog.EmitComponent("warn", "task_revision_auth_denied", "taskTaskService", traceID, map[string]string{
		"user_id": userID,
		"task_id": taskID,
		"reason":  reason,
	})
}
