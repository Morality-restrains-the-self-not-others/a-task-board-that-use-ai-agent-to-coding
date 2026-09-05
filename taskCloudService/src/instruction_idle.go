package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	eventContainerInstructionIdleMarked  = "CONTAINER_INSTRUCTION_IDLE_MARKED"
	eventContainerInstructionIdleCleared = "CONTAINER_INSTRUCTION_IDLE_CLEARED"
)

func optionalBoolPresent(body map[string]any, key string) (val bool, present bool) {
	if body == nil {
		return false, false
	}
	v, ok := body[key]
	if !ok || v == nil {
		return false, false
	}
	return parseBoolish(v, false), true
}

func applyInstructionIdleFromHeartbeat(ctx context.Context, cfgRow *CloudServerConfig, body map[string]any) {
	if cfgRow == nil {
		return
	}
	idle, present := optionalBoolPresent(body, "instruction_idle")
	if !present {
		return
	}
	if idle {
		if err := markInstructionIdle(ctx, cfgRow); err != nil {
			logInfo("event=instruction_idle_mark status=error cfg="+cfgRow.ID+" err="+err.Error(), cfgRow.TaskID)
		}
		return
	}
	if err := clearInstructionIdle(ctx, cfgRow, "heartbeat"); err != nil {
		logInfo("event=instruction_idle_clear status=error cfg="+cfgRow.ID+" err="+err.Error(), cfgRow.TaskID)
	}
}

func markInstructionIdle(ctx context.Context, cfgRow *CloudServerConfig) error {
	if cfgRow == nil || strings.TrimSpace(cfgRow.ID) == "" || db == nil {
		return nil
	}
	policy, err := loadWorkspaceMachinePolicy(cfgRow.CompanyID, cfgRow.WorkspaceID)
	if err != nil {
		return err
	}
	if policy.IdleRecycleMinutes <= 0 {
		logInfo("event=instruction_idle_mark status=skip minutes=0 cfg="+cfgRow.ID, cfgRow.TaskID)
		return nil
	}
	now := time.Now().UTC()
	res, err := db.Exec(
		`UPDATE cloud_server_configs SET instruction_idle_since=?, updated_at=?
		 WHERE id=? AND instruction_idle_since IS NULL`,
		now, now, cfgRow.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}
	logInfo("event=instruction_idle_mark status=ok cfg="+cfgRow.ID+" minutes="+fmt.Sprintf("%d", policy.IdleRecycleMinutes), cfgRow.TaskID)
	_ = publishDomainEvent(ctx, eventContainerInstructionIdleMarked, map[string]interface{}{
		"company_id":   cfgRow.CompanyID,
		"workspace_id": cfgRow.WorkspaceID,
		"task_id":      cfgRow.TaskID,
		"comment_id":   cfgRow.CommentID,
		"config_id":    cfgRow.ID,
		"minutes":      policy.IdleRecycleMinutes,
	}, cfgRow.TaskID)
	return nil
}

func clearInstructionIdle(ctx context.Context, cfgRow *CloudServerConfig, reason string) error {
	if cfgRow == nil || strings.TrimSpace(cfgRow.ID) == "" || db == nil {
		return nil
	}
	now := time.Now().UTC()
	res, err := db.Exec(
		`UPDATE cloud_server_configs SET instruction_idle_since=NULL, updated_at=?
		 WHERE id=? AND instruction_idle_since IS NOT NULL`,
		now, cfgRow.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}
	logInfo("event=instruction_idle_clear status=ok cfg="+cfgRow.ID+" reason="+reason, cfgRow.TaskID)
	_ = publishDomainEvent(ctx, eventContainerInstructionIdleCleared, map[string]interface{}{
		"company_id":   cfgRow.CompanyID,
		"workspace_id": cfgRow.WorkspaceID,
		"task_id":      cfgRow.TaskID,
		"comment_id":   cfgRow.CommentID,
		"config_id":    cfgRow.ID,
		"reason":       reason,
	}, cfgRow.TaskID)
	return nil
}

func loadInstructionIdleSince(configID string) (time.Time, bool) {
	if db == nil || strings.TrimSpace(configID) == "" {
		return time.Time{}, false
	}
	var raw sql.NullString
	err := db.QueryRow(`SELECT instruction_idle_since FROM cloud_server_configs WHERE id=?`, configID).Scan(&raw)
	if err != nil || !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return time.Time{}, false
	}
	return resolveIdleSinceTimestamp(raw.String, "")
}

func instructionJobIsActive(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "pending", "start", "started":
		return true
	default:
		return false
	}
}

func instructionJobIsTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "failed", "interrupted":
		return true
	default:
		return false
	}
}

func latestInstructionJobStatusAndTime(cfg *CloudServerConfig) (status string, at time.Time, ok bool) {
	if cfg == nil || strings.TrimSpace(cfg.CommentID) == "" {
		return "", time.Time{}, false
	}
	jobID, err := latestJobIDForComment(cfg.WorkspaceID, cfg.TaskID, cfg.CommentID)
	if err != nil || strings.TrimSpace(jobID) == "" {
		return "", time.Time{}, false
	}
	events, err := listJobExecutionEvents(cfg.WorkspaceID, cfg.TaskID, cfg.CommentID, jobID)
	if err != nil || len(events) == 0 {
		return "", time.Time{}, false
	}
	for _, ev := range events {
		ph := strings.ToLower(strings.TrimSpace(ev.Phase))
		if s := strings.ToLower(strings.TrimSpace(ev.JobStatus)); s != "" {
			status = s
		} else if ph == "running" || ph == "start" || ph == "completed" || ph == "failed" || ph == "interrupted" {
			status = ph
		}
		if ev.CreatedAt.After(at) {
			at = ev.CreatedAt
		}
	}
	return status, at, status != "" || !at.IsZero()
}

func instructionIdleAnchorTime(cfg *CloudServerConfig, now time.Time) (time.Time, bool) {
	if cfg == nil {
		return time.Time{}, false
	}
	if since, ok := loadInstructionIdleSince(cfg.ID); ok {
		return since, true
	}
	status, at, ok := latestInstructionJobStatusAndTime(cfg)
	if ok && instructionJobIsActive(status) {
		return time.Time{}, false
	}
	if ok && instructionJobIsTerminal(status) && !at.IsZero() {
		return at, true
	}
	if ok && !instructionJobIsTerminal(status) {
		return time.Time{}, false
	}
	return parseIdleClockPreferPast(neverInstructedIdleRaw(cfg), now)
}

func recycleInstructionIdleMachines(now time.Time) (int, error) {
	if db == nil {
		return 0, nil
	}
	rows, err := db.Query(
		`SELECT id FROM cloud_server_configs
		 WHERE TRIM(COALESCE(instance_id,'')) != ''
		   AND COALESCE(terminal_released,0) != 1
		   AND TRIM(COALESCE(comment_id,'')) != ''
		   AND (
		     instruction_idle_since IS NOT NULL
		     OR (userdata_run_verified IS NOT NULL AND TRIM(COALESCE(userdata_run_verified,'')) != '')
		     OR created_at IS NOT NULL
		   )`,
	)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	recycled := 0
	ctx := context.Background()
	byID, err := loadCloudServerConfigsByIDs(ids)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		cfg := byID[id]
		if cfg == nil {
			continue
		}
		policy, err := loadWorkspaceMachinePolicy(cfg.CompanyID, cfg.WorkspaceID)
		if err != nil {
			return recycled, err
		}
		if policy.IdleRecycleMinutes <= 0 {
			continue
		}
		since, ok := instructionIdleAnchorTime(cfg, now)
		if !ok {
			continue
		}
		deadline := since.Add(time.Duration(policy.IdleRecycleMinutes) * time.Minute)
		if now.Before(deadline) {
			continue
		}
		// OPT-20260825-001：无 instruction_idle_since 的候选按 userdata_run_verified/created_at
		// 兜底判闲置，长克隆可能超过 idle 窗口被误回收。先探测容器 clone_done，
		// 克隆未完成或探测失败一律跳过（保守不拆机）。
		if _, hasIdleSince := loadInstructionIdleSince(cfg.ID); !hasIdleSince {
			if !probeCloneDone(ctx, cfg) {
				logInfo("event=instruction_idle_recycle status=skip_clone_pending cfg="+cfg.ID+" instance="+cfg.InstanceID, cfg.TaskID)
				continue
			}
		}
		logInfo("event=instruction_idle_recycle status=start cfg="+cfg.ID+" instance="+cfg.InstanceID, cfg.TaskID)
		_, err = releaseMachineForTerminal(ctx, cfg, cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID, "cancelled", "instruction_idle")
		if err != nil {
			logInfo("event=instruction_idle_recycle status=error cfg="+cfg.ID+" err="+err.Error(), cfg.TaskID)
			return recycled, err
		}
		recycled++
		logInfo("event=instruction_idle_recycle status=ok cfg="+cfg.ID, cfg.TaskID)
	}
	return recycled, nil
}
