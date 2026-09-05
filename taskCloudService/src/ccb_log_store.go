package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"snowflake"
)

func lookupWorkspaceIDByTask(taskID string) string {
	taskID = trim(taskID)
	if taskID == "" || db == nil {
		return ""
	}
	var ws string
	err := db.QueryRow(
		`SELECT workspace_id FROM cloud_server_configs
		 WHERE task_id=? AND workspace_id<>''
		 ORDER BY updated_at DESC LIMIT 1`,
		taskID,
	).Scan(&ws)
	if err != nil {
		return ""
	}
	return trim(ws)
}

func resolveLogWorkspaceID(explicit, taskID string, statusData map[string]interface{}) string {
	if ws := trim(explicit); ws != "" {
		return ws
	}
	if statusData != nil {
		if ws := strField(statusData, "workspace_id"); ws != "" {
			return ws
		}
	}
	return lookupWorkspaceIDByTask(taskID)
}

func bindingLogWorkspaceID(b *CommentContainerBinding) string {
	if b == nil {
		return ""
	}
	return resolveLogWorkspaceID(b.WorkspaceID, b.TaskID, nil)
}

func insertCCBLogRow(workspaceID, companyID, taskID, commentID, bindingID, stage, message, createdAt string) error {
	ws := trim(workspaceID)
	if ws == "" {
		return fmt.Errorf("workspace_id required for log shard")
	}
	table, err := ccbLogTable(ws)
	if err != nil {
		return err
	}
	id := snowflake.GenerateIDString()
	_, err = db.Exec(
		`INSERT INTO `+table+`(
			id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		id, ws, companyID, taskID, commentID, bindingID, stage, message, createdAt,
	)
	if err == nil {
		log.Printf("[taskCloudService] event=ccb_log_shard_write workspace_id=%s table=%s stage=%s",
			ws, table, stage)
		persistCommentStartupLogBestEffort(context.Background(), CommentContainerBindingLog{
			ID: id, WorkspaceID: ws, CompanyID: companyID, TaskID: taskID, CommentID: commentID,
			BindingID: bindingID, Stage: stage, Message: message, CreatedAt: parseCloudUTCDateTime(createdAt),
		})
	}
	return err
}

func countCCBLogShardRows(workspaceID, companyID, taskID string) (int, error) {
	ws := trim(workspaceID)
	if ws == "" {
		return 0, fmt.Errorf("workspace_id required for log shard")
	}
	table, err := ccbLogTable(ws)
	if err != nil {
		return 0, err
	}
	var n int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM `+table+` WHERE workspace_id=? AND company_id=? AND task_id=?`,
		ws, trim(companyID), trim(taskID),
	).Scan(&n)
	return n, err
}

func deleteCCBLogShardRow(workspaceID, logID string) error {
	ws := trim(workspaceID)
	id := trim(logID)
	if ws == "" || id == "" {
		return fmt.Errorf("workspace_id and log id required")
	}
	table, err := ccbLogTable(ws)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM `+table+` WHERE workspace_id=? AND id=?`, ws, id)
	return err
}

func deleteCCBLogShardRowsForComment(workspaceID, companyID, taskID, commentID string) error {
	ws := trim(workspaceID)
	if ws == "" || trim(commentID) == "" {
		return fmt.Errorf("workspace_id and comment_id required")
	}
	table, err := ccbLogTable(ws)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`DELETE FROM `+table+` WHERE workspace_id=? AND company_id=? AND task_id=? AND comment_id=?`,
		ws, trim(companyID), trim(taskID), trim(commentID),
	)
	return err
}

func appendCommentContainerBindingLog(b *CommentContainerBinding, stage string) error {
	if b == nil {
		return nil
	}
	msg := ccbStageMessage(stage)
	if msg == "" {
		return nil
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	return insertCCBLogRow(bindingLogWorkspaceID(b), b.CompanyID, b.TaskID, b.CommentID, b.ID, stage, msg, now)
}

func logCommentContainerBindingStageBestEffort(b *CommentContainerBinding, stage string) {
	if err := appendCommentContainerBindingLog(b, stage); err != nil {
		log.Printf("[taskCloudService] event=comment_container_binding_log_failed binding_id=%s stage=%s err=%v",
			b.ID, stage, err)
	}
}

func listCommentContainerBindingLogsIn(workspaceID, companyID, taskID string) ([]CommentContainerBindingLog, error) {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" {
		return nil, fmt.Errorf("company_id, task_id required")
	}
	ws := resolveLogWorkspaceID(workspaceID, taskID, nil)
	if ws == "" {
		return nil, fmt.Errorf("workspace_id required for log shard")
	}
	table, err := ccbLogTable(ws)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
		FROM `+table+`
		WHERE workspace_id=? AND company_id=? AND task_id=?
		ORDER BY created_at ASC, id ASC`,
		ws, companyID, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CommentContainerBindingLog, 0)
	for rows.Next() {
		var l CommentContainerBindingLog
		var created string
		if err := rows.Scan(
			&l.ID, &l.WorkspaceID, &l.CompanyID, &l.TaskID, &l.CommentID, &l.BindingID, &l.Stage, &l.Message, &created,
		); err != nil {
			return nil, err
		}
		l.CreatedAt = parseCloudUTCDateTime(created)
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mergeStartupLogsFromObjectStore(ws, companyID, taskID, out), nil
}

func listCommentContainerBindingLogs(companyID, taskID string) ([]CommentContainerBindingLog, error) {
	return listCommentContainerBindingLogsIn("", companyID, taskID)
}

func listCommentContainerBindingLogsByComment(companyID, taskID string) (map[string][]CommentContainerBindingLog, error) {
	return listCommentContainerBindingLogsByCommentIn("", companyID, taskID)
}

func listCommentContainerBindingLogsByCommentIn(workspaceID, companyID, taskID string) (map[string][]CommentContainerBindingLog, error) {
	logs, err := listCommentContainerBindingLogsIn(workspaceID, companyID, taskID)
	if err != nil {
		return nil, err
	}
	byComment := make(map[string][]CommentContainerBindingLog)
	for _, l := range logs {
		byComment[l.CommentID] = append(byComment[l.CommentID], l)
	}
	return byComment, nil
}

func commentContainerBindingLogToJSON(l *CommentContainerBindingLog) map[string]interface{} {
	return map[string]interface{}{
		"id":         l.ID,
		"stage":      l.Stage,
		"message":    l.Message,
		"created_at": formatCloudUTCJSON(l.CreatedAt),
	}
}

func commentContainerBindingLogsToJSON(logs []CommentContainerBindingLog) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(logs))
	for i := range logs {
		out = append(out, commentContainerBindingLogToJSON(&logs[i]))
	}
	return out
}
