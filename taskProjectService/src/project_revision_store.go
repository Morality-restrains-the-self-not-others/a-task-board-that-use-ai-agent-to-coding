package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"snowflake"
	"taskProjectService/domain"
	"tracelog"
)

type projectRevisionRow struct {
	ID            string
	TenantID      string
	ProjectID     string
	VersionNum    int
	Name          string
	Description   string
	ActorUserID   string
	ChangedFields string
	CreatedAt     time.Time
}

func insertProjectRevisionTx(tx *sql.Tx, row projectRevisionRow) error {
	if tx == nil {
		return fmt.Errorf("insertProjectRevisionTx: nil tx")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = snowflake.GenerateIDString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	_, err := tx.Exec(
		`INSERT INTO project_revision(id,tenant_id,project_id,version_num,name,description,actor_user_id,changed_fields,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		row.ID, row.TenantID, row.ProjectID, row.VersionNum,
		row.Name, row.Description, row.ActorUserID, row.ChangedFields, row.CreatedAt,
	)
	if err != nil {
		log.Printf("[taskProjectService] event=project_revision_insert_failed project_id=%s tenant_id=%s err=%v",
			row.ProjectID, row.TenantID, err)
	}
	return err
}

var insertProjectRevisionTxFn = insertProjectRevisionTx

func maxProjectRevisionVersion(tx *sql.Tx, projectID, tenantID string) (int, error) {
	var max sql.NullInt64
	err := tx.QueryRow(
		`SELECT MAX(version_num) FROM project_revision WHERE project_id=? AND tenant_id=?`,
		projectID, tenantID,
	).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64), nil
}

func persistProjectRevisionV1(tx *sql.Tx, tenantID, projectID, name, description, actorUserID string, now time.Time) (string, error) {
	if !domain.ShouldRecordCreate() {
		return "", nil
	}
	id := snowflake.GenerateIDString()
	row := projectRevisionRow{
		ID:            id,
		TenantID:      tenantID,
		ProjectID:     projectID,
		VersionNum:    domain.NextVersion(0),
		Name:          name,
		Description:   description,
		ActorUserID:   actorUserID,
		ChangedFields: domain.CreateChangedFields(),
		CreatedAt:     now,
	}
	if err := insertProjectRevisionTxFn(tx, row); err != nil {
		return "", err
	}
	log.Printf("[taskProjectService] event=project_revision_recorded project_id=%s revision_id=%s version_num=%d actor_user_id=%s changed_fields=%s",
		projectID, id, row.VersionNum, actorUserID, row.ChangedFields)
	return id, nil
}

func persistProjectRevisionIfChanged(tx *sql.Tx, tenantID, projectID, actorUserID string, old, new domain.VersionedProjectContent, now time.Time) (revisionID string, versionNum int, changed []string, recorded bool, err error) {
	changed, record := domain.DiffVersionedProject(old, new)
	if !record {
		return "", 0, nil, false, nil
	}
	max, err := maxProjectRevisionVersion(tx, projectID, tenantID)
	if err != nil {
		return "", 0, changed, false, err
	}
	id := snowflake.GenerateIDString()
	ver := domain.NextVersion(max)
	row := projectRevisionRow{
		ID:            id,
		TenantID:      tenantID,
		ProjectID:     projectID,
		VersionNum:    ver,
		Name:          domain.NormalizeText(new.Name),
		Description:   new.Description,
		ActorUserID:   actorUserID,
		ChangedFields: domain.JoinChangedFields(changed),
		CreatedAt:     now,
	}
	if err := insertProjectRevisionTxFn(tx, row); err != nil {
		return "", 0, changed, false, err
	}
	log.Printf("[taskProjectService] event=project_revision_recorded project_id=%s revision_id=%s version_num=%d actor_user_id=%s changed_fields=%s",
		projectID, id, ver, actorUserID, row.ChangedFields)
	return id, ver, changed, true, nil
}

func listProjectRevisions(projectID, tenantID string, limit, offset int) ([]projectRevisionRow, int, error) {
	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM project_revision WHERE project_id=? AND tenant_id=?`,
		projectID, tenantID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(
		`SELECT id,tenant_id,project_id,version_num,name,COALESCE(description,''),actor_user_id,changed_fields,created_at
		 FROM project_revision WHERE project_id=? AND tenant_id=?
		 ORDER BY version_num DESC LIMIT ? OFFSET ?`,
		projectID, tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []projectRevisionRow
	for rows.Next() {
		var r projectRevisionRow
		if err := rows.Scan(&r.ID, &r.TenantID, &r.ProjectID, &r.VersionNum,
			&r.Name, &r.Description, &r.ActorUserID, &r.ChangedFields, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func getProjectRevision(projectID, tenantID, revisionID string) (*projectRevisionRow, error) {
	var r projectRevisionRow
	err := db.QueryRow(
		`SELECT id,tenant_id,project_id,version_num,name,COALESCE(description,''),actor_user_id,changed_fields,created_at
		 FROM project_revision WHERE id=? AND project_id=? AND tenant_id=?`,
		revisionID, projectID, tenantID,
	).Scan(&r.ID, &r.TenantID, &r.ProjectID, &r.VersionNum,
		&r.Name, &r.Description, &r.ActorUserID, &r.ChangedFields, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func projectRevisionListJSON(r projectRevisionRow) map[string]interface{} {
	return map[string]interface{}{
		"id":             r.ID,
		"version_num":    r.VersionNum,
		"name":           r.Name,
		"description":    domain.TruncateRunes(r.Description, 200),
		"actor_user_id":  r.ActorUserID,
		"changed_fields": r.ChangedFields,
		"created_at":     r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func projectRevisionDetailJSON(r projectRevisionRow) map[string]interface{} {
	m := projectRevisionListJSON(r)
	m["description"] = r.Description
	m["tenant_id"] = r.TenantID
	m["project_id"] = r.ProjectID
	return m
}

func logProjectRevisionAuthDenied(traceID, userID, projectID, reason string) {
	tracelog.EmitComponent("warn", "project_revision_auth_denied", "taskProjectService", traceID, map[string]string{
		"user_id":    userID,
		"project_id": projectID,
		"reason":     reason,
	})
}
