package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"snowflake"
	"taskCloudService/domain"
	"tracelog"
)

type layerGraphSnapshotRow struct {
	ID          string
	CompanyID   string
	WorkspaceID string
	TaskID      string
	CommentID   string
	GraphJSON   string
}

func upsertLayerGraphSnapshot(row layerGraphSnapshotRow) error {
	if db == nil {
		return fmt.Errorf("db not open")
	}
	id := domain.LayerGraphSnapshotID{
		WorkspaceID: row.WorkspaceID,
		TaskID:      row.TaskID,
		CommentID:   row.CommentID,
	}.Normalize()
	if err := id.Validate(); err != nil {
		return err
	}
	row.WorkspaceID = id.WorkspaceID
	row.TaskID = id.TaskID
	row.CommentID = id.CommentID
	row.CompanyID = strings.TrimSpace(row.CompanyID)
	raw := strings.TrimSpace(row.GraphJSON)
	if raw == "" {
		raw = `{"layers":[],"jobs":[]}`
	}
	if err := domain.ValidateGraphDocument([]byte(raw)); err != nil {
		return err
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = snowflake.GenerateIDString()
	}
	_, err := db.Exec(
		`INSERT INTO cloud_layer_graph_snapshot (
			id, company_id, workspace_id, task_id, comment_id, graph_json
		) VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			graph_json = VALUES(graph_json),
			company_id = VALUES(company_id),
			updated_at = CURRENT_TIMESTAMP`,
		row.ID, row.CompanyID, row.WorkspaceID, row.TaskID, row.CommentID, raw,
	)
	if err != nil {
		return err
	}
	tracelog.LogForwardStage(context.Background(), "layer_graph_snapshot_upsert", map[string]any{
		"workspace_id": row.WorkspaceID,
		"task_id":      row.TaskID,
		"comment_id":   row.CommentID,
	})
	return nil
}

func getLayerGraphSnapshot(workspaceID, taskID, commentID string) (layerGraphSnapshotRow, bool, error) {
	id := domain.LayerGraphSnapshotID{
		WorkspaceID: workspaceID,
		TaskID:      taskID,
		CommentID:   commentID,
	}.Normalize()
	if err := id.Validate(); err != nil {
		return layerGraphSnapshotRow{}, false, err
	}
	if db == nil {
		return layerGraphSnapshotRow{}, false, fmt.Errorf("db not open")
	}
	var row layerGraphSnapshotRow
	err := db.QueryRow(
		`SELECT id, company_id, workspace_id, task_id, comment_id, IFNULL(graph_json, '')
		 FROM cloud_layer_graph_snapshot
		 WHERE workspace_id = ? AND task_id = ? AND comment_id = ?
		 LIMIT 1`,
		id.WorkspaceID, id.TaskID, id.CommentID,
	).Scan(&row.ID, &row.CompanyID, &row.WorkspaceID, &row.TaskID, &row.CommentID, &row.GraphJSON)
	if err == sql.ErrNoRows {
		return layerGraphSnapshotRow{}, false, nil
	}
	if err != nil {
		return layerGraphSnapshotRow{}, false, err
	}
	return row, true, nil
}

func countLayerGraphSnapshots(workspaceID, taskID, commentID string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("db not open")
	}
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM cloud_layer_graph_snapshot
		 WHERE workspace_id = ? AND task_id = ? AND comment_id = ?`,
		strings.TrimSpace(workspaceID), strings.TrimSpace(taskID), strings.TrimSpace(commentID),
	).Scan(&n)
	return n, err
}

func marshalLayerGraphDocument(layers, jobs []any, extra map[string]any) (string, error) {
	doc := map[string]any{
		"layers": layers,
		"jobs":   jobs,
	}
	for k, v := range extra {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		doc[k] = v
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
