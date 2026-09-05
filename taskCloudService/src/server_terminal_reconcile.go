package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// reconcileTerminalTaskCSCs releases machines whose tasks have moved to
// 已取消/已完成 but CSC was left running (TASK_STATUS_CHANGED DLT / cloud down).
func reconcileTerminalTaskCSCs(now time.Time) (cleaned int, err error) {
	_ = now
	rows, err := db.Query(
		`SELECT id, company_id, workspace_id, task_id
		 FROM cloud_server_configs
		 WHERE COALESCE(instance_id,'') != ''
		   AND COALESCE(terminal_released,0) != 1
		   AND COALESCE(last_runtime_status,'') != 'Released'
		   AND TRIM(COALESCE(comment_id,'')) != ''`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type cand struct {
		ID, CompanyID, WorkspaceID, TaskID string
	}
	var candidates []cand
	seen := map[string]bool{}
	taskIDs := make([]string, 0)
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.WorkspaceID, &c.TaskID); err != nil {
			continue
		}
		c.TaskID = strings.TrimSpace(c.TaskID)
		if c.TaskID == "" {
			continue
		}
		candidates = append(candidates, c)
		if !seen[c.TaskID] {
			seen[c.TaskID] = true
			taskIDs = append(taskIDs, c.TaskID)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	kinds, err := lookupTaskTerminalKindsCached(taskIDs)
	if err != nil {
		return 0, err
	}

	cscIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		cscIDs = append(cscIDs, c.ID)
	}
	byID, loadErr := loadCloudServerConfigsByIDs(cscIDs)
	if loadErr != nil {
		return 0, loadErr
	}

	for _, c := range candidates {
		kind := strings.TrimSpace(kinds[c.TaskID])
		if !isTaskTerminalKind(kind) {
			continue
		}
		cfgRow := byID[c.ID]
		if cfgRow == nil {
			logWarn(fmt.Sprintf("event=reconcile_terminal_csc_load_failed id=%s task_id=%s err=%v", c.ID, c.TaskID, "missing"), c.TaskID)
			continue
		}
		if _, relErr := releaseMachineForTerminal(context.Background(), cfgRow, c.CompanyID, c.WorkspaceID, c.TaskID, kind, "reconcile_task_terminal"); relErr != nil {
			logWarn(fmt.Sprintf("event=reconcile_terminal_csc_release_failed id=%s task_id=%s err=%v", c.ID, c.TaskID, relErr), c.TaskID)
			continue
		}
		logInfo(fmt.Sprintf("event=reconcile_terminal_csc task_id=%s kind=%s id=%s", c.TaskID, kind, c.ID), c.TaskID)
		cleaned++
	}
	return cleaned, nil
}
