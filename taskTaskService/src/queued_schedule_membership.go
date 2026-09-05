package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// resolveTopTaskForQueue walks parent_task_id to root; returns top task id and depth from top.
func resolveTopTaskForQueue(taskID string) (topID string, depth int, err error) {
	seen := map[string]struct{}{}
	cur := taskID
	chain := []string{cur}
	for {
		if _, ok := seen[cur]; ok {
			return "", 0, fmt.Errorf("parent cycle at %s", cur)
		}
		seen[cur] = struct{}{}
		t, err := loadTask(cur)
		if err != nil {
			return "", 0, err
		}
		parent := strings.TrimSpace(t.ParentTaskID)
		if parent == "" {
			return cur, len(chain) - 1, nil
		}
		cur = parent
		chain = append([]string{cur}, chain...)
	}
}

func loadMembership(taskID string) (*queuedMembership, error) {
	row := db.QueryRow(`SELECT task_id,tenant_id,workspace_id,top_task_id,depth,status,COALESCE(enabler_user_id,''),enqueued_at,updated_at
		FROM task_queued_auto_run_memberships WHERE task_id=?`, taskID)
	var m queuedMembership
	err := row.Scan(&m.TaskID, &m.TenantID, &m.WorkspaceID, &m.TopTaskID, &m.Depth, &m.Status, &m.EnablerUserID, &m.EnqueuedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func enqueueQueuedAutoRun(t *taskRecord, actorUserID string) (*queuedMembership, error) {
	topID, depth, err := resolveTopTaskForQueue(t.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	status := "queued"
	// 有效节奏：工作空间级优先，未配置时回退任务级（legacy）
	wsRhythm, wsScoped := loadEffectiveWorkspaceRhythm(t.WorkspaceID)
	if wsScoped && !wsRhythm.Enabled {
		return nil, errWorkspaceAutoScheduleDisabled()
	}
	var legacyRhythm *scheduleRhythm
	if wsScoped {
		if !wsRhythm.Enabled || !workspaceRhythmInWindow(wsRhythm, now) {
			status = "deferred"
		}
	} else {
		legacyRhythm, _ = loadScheduleRhythm(topID)
		if legacyRhythm == nil || !legacyRhythm.Enabled || !rhythmInWindow(legacyRhythm, now) {
			status = "deferred"
		}
	}
	_, err = db.Exec(`INSERT INTO task_queued_auto_run_memberships(
		task_id,tenant_id,workspace_id,top_task_id,depth,status,enabler_user_id,enqueued_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			top_task_id=VALUES(top_task_id),
			depth=VALUES(depth),
			status=VALUES(status),
			enabler_user_id=IF(VALUES(enabler_user_id)='', enabler_user_id, VALUES(enabler_user_id)),
			updated_at=VALUES(updated_at)`,
		t.ID, t.TenantID, t.WorkspaceID, topID, depth, status, strings.TrimSpace(actorUserID), now, now)
	if err != nil {
		return nil, err
	}
	_ = publishDomainEvent(context.Background(), "TaskQueuedForAutoRun", map[string]interface{}{
		"task_id":      t.ID,
		"tenant_id":    t.TenantID,
		"workspace_id": t.WorkspaceID,
		"top_task_id":  topID,
		"depth":        depth,
		"status":       status,
	}, t.ID)
	reason := ""
	if status == "deferred" {
		reason = "自动调度未启用"
		if wsScoped {
			reason = deferredReasonWorkspace(wsRhythm)
		} else if legacyRhythm != nil {
			reason = deferredReason(legacyRhythm)
		}
		_ = publishDomainEvent(context.Background(), "QueuedAutoRunDeferred", map[string]interface{}{
			"task_id":   t.ID,
			"tenant_id": t.TenantID,
			"reason":    reason,
		}, t.ID)
	}
	enqMsg := "加入自动调度队列"
	if title := strings.TrimSpace(t.Title); title != "" {
		enqMsg += "：" + title
	}
	if reason != "" {
		enqMsg += "（" + reason + "）"
	}
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: t.TenantID, WorkspaceID: t.WorkspaceID, EventType: scheduleHistoryEventMemberEnqueued,
		TaskID: t.ID, Message: enqMsg, ActorUserID: actorUserID,
	})
	return loadMembership(t.ID)
}

func dequeueQueuedAutoRun(taskID, reason, actorUserID string) error {
	m, err := loadMembership(taskID)
	if err != nil || m == nil {
		return err
	}
	_, _ = db.Exec(`DELETE FROM task_queued_auto_run_memberships WHERE task_id=?`, taskID)
	releaseQueuedSlot(taskID)
	_ = publishDomainEvent(context.Background(), "TaskDequeuedFromAutoRun", map[string]interface{}{
		"task_id":      taskID,
		"tenant_id":    m.TenantID,
		"workspace_id": m.WorkspaceID,
		"top_task_id":  m.TopTaskID,
		"reason":       reason,
	}, taskID)
	deqMsg := "离开自动调度队列"
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: m.TenantID, WorkspaceID: m.WorkspaceID, EventType: scheduleHistoryEventMemberDequeued,
		TaskID: taskID, Message: deqMsg, ActorUserID: actorUserID,
	})
	return nil
}

func countQueuedSlots(topTaskID string) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM task_queued_machine_slots WHERE top_task_id=?`, topTaskID).Scan(&n)
	return n
}

func hasQueuedSlot(taskID string) bool {
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM task_queued_machine_slots WHERE task_id=?`, taskID).Scan(&n)
	return n > 0
}

func acquireQueuedSlot(taskID, topTaskID string) error {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)
		ON DUPLICATE KEY UPDATE top_task_id=VALUES(top_task_id), acquired_at=VALUES(acquired_at)`,
		taskID, topTaskID, now)
	return err
}

func releaseQueuedSlot(taskID string) {
	_, _ = db.Exec(`DELETE FROM task_queued_machine_slots WHERE task_id=?`, taskID)
}

// resetStartingMembershipsToQueued 槽位释放后把仍为 starting 的成员收回 queued，避免「调度启服中」且占用 0。
func resetStartingMembershipsToQueued(taskIDs []string) {
	for _, id := range taskIDs {
		m, err := loadMembership(id)
		if err != nil || m == nil || m.Status != "starting" {
			continue
		}
		log.Printf("[taskTaskService] event=queued_starting_reset_after_slot_release task_id=%s workspace_id=%s", id, m.WorkspaceID)
		setMembershipStatus(id, "queued")
	}
}

func listMembershipsForTop(topTaskID string) ([]queuedMembership, error) {
	rows, err := db.Query(`SELECT task_id,tenant_id,workspace_id,top_task_id,depth,status,COALESCE(enabler_user_id,''),enqueued_at,updated_at
		FROM task_queued_auto_run_memberships WHERE top_task_id=?
		ORDER BY depth ASC, enqueued_at ASC`, topTaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []queuedMembership
	for rows.Next() {
		var m queuedMembership
		if err := rows.Scan(&m.TaskID, &m.TenantID, &m.WorkspaceID, &m.TopTaskID, &m.Depth, &m.Status, &m.EnablerUserID, &m.EnqueuedAt, &m.UpdatedAt); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func countQueuedAhead(taskID string) int {
	m, err := loadMembership(taskID)
	if err != nil || m == nil {
		return 0
	}
	members, err := listMembershipsForTop(m.TopTaskID)
	if err != nil {
		return 0
	}
	n := 0
	for _, x := range members {
		if x.TaskID == taskID {
			break
		}
		switch x.Status {
		case "queued", "deferred", "starting":
			n++
		}
	}
	return n
}

func enrichTaskJSONWithQueue(out map[string]interface{}, taskID string) {
	if out == nil {
		return
	}
	r, _ := loadScheduleRhythm(taskID)
	if r != nil {
		out["schedule_rhythm"] = rhythmToJSON(r)
	} else {
		out["schedule_rhythm"] = nil
	}
	if topID, depth, err := resolveTopTaskForQueue(taskID); err == nil {
		out["queued_top_task_id"] = topID
		out["queued_depth"] = depth
	}
	m, _ := loadMembership(taskID)
	if m != nil {
		out["queued_auto_run"] = true
		out["queued_auto_run_status"] = m.Status
		out["queued_top_task_id"] = m.TopTaskID
		out["queued_depth"] = m.Depth
		out["queued_ahead_count"] = countQueuedAhead(taskID)
		out["queued_deferred_reason"] = ""
		if m.Status == "deferred" {
			// 有效节奏：工作空间级优先，未配置时回退任务级（legacy）
			if wsRhythm, wsScoped := loadEffectiveWorkspaceRhythm(m.WorkspaceID); wsScoped {
				out["queued_deferred_reason"] = deferredReasonWorkspace(wsRhythm)
			} else {
				rhythm, _ := loadScheduleRhythm(m.TopTaskID)
				out["queued_deferred_reason"] = deferredReason(rhythm)
			}
		}
	} else {
		out["queued_auto_run"] = false
		out["queued_auto_run_status"] = ""
		out["queued_ahead_count"] = 0
		out["queued_deferred_reason"] = ""
	}
}

func applyQueuedAutoRunFromBody(t *taskRecord, body map[string]interface{}, actorUserID string) error {
	if _, ok := body["queued_auto_run"]; !ok {
		return nil
	}
	want := boolField(body, "queued_auto_run")
	if want {
		_, err := enqueueQueuedAutoRun(t, actorUserID)
		return err
	}
	return dequeueQueuedAutoRun(t.ID, "user_disabled", actorUserID)
}

// deferImmediateAutoRunStart is true when the task is joining the auto-schedule
// queue and the caller is not force-restarting. Immediate start-vm would race
// the queued dispatcher and start a second machine.
func deferImmediateAutoRunStart(body map[string]interface{}, forceAutoRun bool) bool {
	return boolField(body, "queued_auto_run") && !forceAutoRun
}

// dequeueOnManualStart removes queue membership when user manually starts a machine.
func dequeueOnManualStart(taskID, actorUserID string) {
	if err := dequeueQueuedAutoRun(taskID, "manual_start", actorUserID); err != nil {
		log.Printf("[taskTaskService] dequeue on manual start task_id=%s: %v", taskID, err)
	}
}
