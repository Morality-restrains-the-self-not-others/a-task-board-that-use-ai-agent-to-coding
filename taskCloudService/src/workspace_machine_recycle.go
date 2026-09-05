package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func setConfigIdleSince(companyID, workspaceID, taskID string, at time.Time) error {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" {
		return nil
	}
	q := `UPDATE cloud_server_configs SET idle_since=?, updated_at=? WHERE company_id=? AND task_id=?`
	args := []interface{}{at.UTC(), at.UTC(), companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	_, err := db.Exec(q, args...)
	return err
}

func clearConfigIdleSince(companyID, workspaceID, taskID string) error {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" {
		return nil
	}
	q := `UPDATE cloud_server_configs SET idle_since=NULL, updated_at=? WHERE company_id=? AND task_id=?`
	args := []interface{}{time.Now().UTC(), companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	_, err := db.Exec(q, args...)
	return err
}

func maybeMarkIdleOnServerURLClear(cfg *CloudServerConfig, prevServerURL string) error {
	if cfg == nil {
		return nil
	}
	prev := strings.TrimSpace(prevServerURL)
	cur := strings.TrimSpace(cfg.ServerURL)
	if prev == "" || cur != "" {
		return nil
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		return nil
	}
	return setConfigIdleSince(cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID, time.Now().UTC())
}

func maybeClearIdleOnServerURLSet(cfg *CloudServerConfig, prevServerURL string) error {
	if cfg == nil {
		return nil
	}
	prev := strings.TrimSpace(prevServerURL)
	cur := strings.TrimSpace(cfg.ServerURL)
	if cur == "" || prev == cur {
		return nil
	}
	return clearConfigIdleSince(cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID)
}

func recycleIdleMachines(now time.Time) (int, error) {
	policyRows, err := db.Query(
		`SELECT company_id, workspace_id, idle_recycle_minutes
		 FROM cloud_workspace_machine_policies WHERE idle_recycle_minutes > 0`,
	)
	if err != nil {
		return 0, err
	}

	type policyRow struct {
		companyID   string
		workspaceID string
		minutes     int
	}
	policies := []policyRow{}
	for policyRows.Next() {
		var companyID, workspaceID string
		var minutes int
		if err := policyRows.Scan(&companyID, &workspaceID, &minutes); err != nil {
			policyRows.Close()
			return 0, err
		}
		policies = append(policies, policyRow{companyID: companyID, workspaceID: workspaceID, minutes: minutes})
	}
	if err := policyRows.Close(); err != nil {
		return 0, err
	}
	if err := policyRows.Err(); err != nil {
		return 0, err
	}

	recycled := 0
	for _, p := range policies {
		n, err := recycleIdleMachinesForWorkspace(p.companyID, p.workspaceID, p.minutes, now)
		if err != nil {
			return recycled, err
		}
		recycled += n
	}
	nInstr, err := recycleInstructionIdleMachines(now)
	if err != nil {
		return recycled, err
	}
	return recycled + nInstr, nil
}

type recycleInstanceGroup struct {
	allIdle       bool
	latestIdle    time.Time
	hasValidSince bool
	tasks         []struct {
		configID string
		taskID   string
	}
}

func recycleIdleMachinesForWorkspace(companyID, workspaceID string, minutes int, now time.Time) (int, error) {
	threshold := now.Add(-time.Duration(minutes) * time.Minute)
	rows, err := db.Query(
		`SELECT id, task_id, COALESCE(instance_id,''), COALESCE(server_url,''),
		        COALESCE(idle_since,''), COALESCE(updated_at,''), COALESCE(last_runtime_status,'')
		 FROM cloud_server_configs
		 WHERE company_id=? AND workspace_id=?
		   AND TRIM(COALESCE(instance_id,'')) != ''`,
		companyID, workspaceID,
	)
	if err != nil {
		return 0, err
	}

	byInstance := map[string]*recycleInstanceGroup{}
	for rows.Next() {
		var id, taskID, instanceID, serverURL, idleSinceRaw, updatedRaw, runtimeStatus string
		if err := rows.Scan(&id, &taskID, &instanceID, &serverURL, &idleSinceRaw, &updatedRaw, &runtimeStatus); err != nil {
			rows.Close()
			return 0, err
		}
		instID := strings.TrimSpace(instanceID)
		if instID == "" {
			continue
		}
		if !machineRuntimeCountsAsStarted(instID, runtimeStatus) {
			continue
		}
		grp, ok := byInstance[instID]
		if !ok {
			grp = &recycleInstanceGroup{allIdle: true}
			byInstance[instID] = grp
		}
		if strings.TrimSpace(serverURL) != "" {
			grp.allIdle = false
		}
		since, ok := resolveIdleSinceTimestamp(idleSinceRaw, updatedRaw)
		if ok && (!grp.hasValidSince || since.After(grp.latestIdle)) {
			grp.latestIdle = since
			grp.hasValidSince = true
		}
		grp.tasks = append(grp.tasks, struct {
			configID string
			taskID   string
		}{configID: id, taskID: taskID})
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	recycled := 0
	for _, grp := range byInstance {
		if !grp.allIdle || !grp.hasValidSince || grp.latestIdle.After(threshold) {
			continue
		}
		for _, t := range grp.tasks {
			if err := recycleIdleMachineConfig(companyID, workspaceID, t.taskID, t.configID); err != nil {
				return recycled, err
			}
			recycled++
		}
	}
	return recycled, nil
}

func resolveIdleSinceTimestamp(idleSinceRaw, updatedRaw string) (time.Time, bool) {
	idleSinceRaw = strings.TrimSpace(idleSinceRaw)
	if idleSinceRaw != "" {
		if t, err := time.Parse(time.RFC3339, idleSinceRaw); err == nil {
			return t.UTC(), true
		}
		if t, err := time.Parse("2006-01-02 15:04:05", idleSinceRaw); err == nil {
			return t.UTC(), true
		}
	}
	updatedRaw = strings.TrimSpace(updatedRaw)
	if updatedRaw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, updatedRaw); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02 15:04:05", updatedRaw); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func recycleIdleMachineConfig(companyID, workspaceID, taskID, configID string) error {
	// OPT-20260823-027：清库前先取 comment_id，供调度行路由到该评论启动日志。
	// idle_recycle 只清库不调云 API，用户仍须能从日志看到「触发：工作区机器空闲回收」。
	var commentID string
	if err := db.QueryRow(`SELECT COALESCE(comment_id,'') FROM cloud_server_configs WHERE id = ?`, configID).Scan(&commentID); err != nil {
		logWarn("event=idle_recycle_load_comment_failed task="+taskID+" config="+configID+" err="+err.Error(), taskID)
	}
	now := time.Now().UTC()
	_, err := db.Exec(
		`UPDATE cloud_server_configs SET
			instance_id='', public_ip='', server_url='', business_api_endpoint='',
			container_vscode_url='', error_reason='', last_runtime_status='', idle_since=NULL, instruction_idle_since=NULL, updated_at=?
		 WHERE id=?`,
		now, configID,
	)
	if err != nil {
		return err
	}
	stopNow := now.Format("2006-01-02 15:04:05")
	_, err = closeOpenCloudServerConfigHistories(companyID, workspaceID, taskID, map[string]interface{}{
		"stopped_at":  stopNow,
		"stop_reason": "idle_recycle",
	}, "")
	if err != nil {
		return err
	}
	// 发布带触发说明的调度行写入评论 binding 启动日志（冷打开可还原）。
	msg := annotateStopServerMessage("工作区机器空闲回收，容器配置已清理", "idle_recycle")
	_ = publishTaskSSE(context.Background(), taskID, commentID, map[string]interface{}{
		"status":            "released",
		"event_name":        "server_status_update",
		"message":           msg,
		"progress":          100,
		"stop_reason":       "idle_recycle",
		"stop_reason_label": stopReasonTriggerLabel("idle_recycle"),
		"runtime_status":    "Stopped",
	})
	logInfo("event=idle_recycle tenant="+companyID+" workspace="+workspaceID+" task="+taskID, taskID)
	return nil
}

func handleInternalRecycleIdleMachines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	recycled, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"recycled": recycled,
	})
}
