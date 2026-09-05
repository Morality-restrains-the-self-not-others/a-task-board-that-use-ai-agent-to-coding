package main

import (
	"context"
	"log"
	"time"

	"tracelog"
)

// minutesUntilWindowEnd returns minutes until daily_end while inside the window; -1 if outside.
func minutesUntilWindowEnd(nowLocal time.Time, dailyStart, dailyEnd string) int {
	if !inDailyWindow(nowLocal, dailyStart, dailyEnd) {
		return -1
	}
	eh, em, ok := parseHHMM(dailyEnd)
	if !ok {
		return -1
	}
	endMins := eh*60 + em
	nowMins := nowLocal.Hour()*60 + nowLocal.Minute()
	sh, sm, ok1 := parseHHMM(dailyStart)
	if !ok1 {
		return -1
	}
	startMins := sh*60 + sm
	if startMins < endMins {
		return endMins - nowMins
	}
	// overnight
	if nowMins >= startMins {
		return (24*60 - nowMins) + endMins
	}
	return endMins - nowMins
}

// autoCloseKeyForInWindowEnd is the idempotency key for the window currently ending.
func autoCloseKeyForInWindowEnd(nowLocal time.Time, dailyStart, dailyEnd string) string {
	endAt := windowEndInstant(nowLocal, dailyStart, dailyEnd, true)
	if endAt.IsZero() {
		return ""
	}
	return endAt.Format("2006-01-02") + "|" + dailyEnd
}

// autoCloseKeyForMostRecentEnd is used when outside the window to release leftover slots.
func autoCloseKeyForMostRecentEnd(nowLocal time.Time, dailyStart, dailyEnd string) string {
	endAt := windowEndInstant(nowLocal, dailyStart, dailyEnd, false)
	if endAt.IsZero() {
		return ""
	}
	return endAt.Format("2006-01-02") + "|" + dailyEnd
}

// windowEndInstant returns the end Instant of the relevant window.
// inWindow=true → end of the active window; false → most recently completed end ≤ now.
func windowEndInstant(nowLocal time.Time, dailyStart, dailyEnd string, inWindow bool) time.Time {
	sh, sm, ok1 := parseHHMM(dailyStart)
	eh, em, ok2 := parseHHMM(dailyEnd)
	if !ok1 || !ok2 {
		return time.Time{}
	}
	startMins := sh*60 + sm
	endMins := eh*60 + em
	y, m, d := nowLocal.Date()
	loc := nowLocal.Location()
	nowMins := nowLocal.Hour()*60 + nowLocal.Minute()

	if startMins < endMins || startMins == endMins {
		endToday := time.Date(y, m, d, eh, em, 0, 0, loc)
		if inWindow {
			return endToday
		}
		if !nowLocal.Before(endToday) {
			return endToday
		}
		return endToday.Add(-24 * time.Hour)
	}
	// overnight: end is morning endMins
	endToday := time.Date(y, m, d, eh, em, 0, 0, loc)
	if inWindow {
		if nowMins >= startMins {
			return endToday.Add(24 * time.Hour)
		}
		return endToday
	}
	// outside: between end and start (daytime gap)
	if nowMins >= endMins && nowMins < startMins {
		return endToday
	}
	if nowMins >= startMins {
		// still in evening part — shouldn't release; return tomorrow end for key stability
		return endToday.Add(24 * time.Hour)
	}
	// early morning still in window
	return endToday
}

func listQueuedSlotTaskIDs(topID string) ([]string, error) {
	rows, err := db.Query(`SELECT task_id FROM task_queued_machine_slots WHERE top_task_id=?`, topID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func clearQueuedSlotsForTop(topID string) {
	_, _ = db.Exec(`DELETE FROM task_queued_machine_slots WHERE top_task_id=?`, topID)
}

func updateAutoCloseKeysForWindow(windowID, warnKey, releaseKey string) error {
	if warnKey == "" && releaseKey == "" {
		return nil
	}
	if warnKey != "" && releaseKey != "" {
		_, err := db.Exec(`UPDATE task_top_deliverable_schedule_rhythm_windows SET auto_close_warn_key=?, auto_close_release_key=?, updated_at=? WHERE id=?`,
			warnKey, releaseKey, time.Now().UTC(), windowID)
		return err
	}
	if warnKey != "" {
		_, err := db.Exec(`UPDATE task_top_deliverable_schedule_rhythm_windows SET auto_close_warn_key=?, updated_at=? WHERE id=?`,
			warnKey, time.Now().UTC(), windowID)
		return err
	}
	_, err := db.Exec(`UPDATE task_top_deliverable_schedule_rhythm_windows SET auto_close_release_key=?, updated_at=? WHERE id=?`,
		releaseKey, time.Now().UTC(), windowID)
	return err
}

func runAutoCloseForTop(topID string) {
	rhythm, err := loadScheduleRhythm(topID)
	if err != nil || rhythm == nil || !rhythm.Enabled {
		return
	}
	windows, err := loadScheduleRhythmWindows(topID)
	if err != nil || len(windows) == 0 {
		return
	}
	loc, err := time.LoadLocation(rhythm.Timezone)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	nowLocal := time.Now().In(loc)
	slots, err := listQueuedSlotTaskIDs(topID)
	if err != nil || len(slots) == 0 {
		return
	}

	// Find the active window or the most recently ended window
	for _, w := range windows {
		if !w.AutoClose {
			continue
		}
		inWin := inDailyWindow(nowLocal, w.DailyStart, w.DailyEnd)
		if inWin {
			mins := minutesUntilWindowEnd(nowLocal, w.DailyStart, w.DailyEnd)
			lead := w.AutoCloseWarnMinutes
			if lead <= 0 {
				lead = 5
			}
			if mins <= 0 || mins > lead {
				continue
			}
			key := autoCloseKeyForInWindowEnd(nowLocal, w.DailyStart, w.DailyEnd)
			if key == "" || key == w.AutoCloseWarnKey {
				continue
			}
			notifyAutoCloseWarnForWindow(rhythm, w, slots, key)
			return
		}
	}

	// Outside all windows — release if any window just ended
	for _, w := range windows {
		if !w.AutoClose {
			continue
		}
		key := autoCloseKeyForMostRecentEnd(nowLocal, w.DailyStart, w.DailyEnd)
		if key == "" || key == w.AutoCloseReleaseKey {
			continue
		}
		releaseAutoCloseSlotsForWindow(rhythm, w, slots, key)
		return
	}
}

func notifyAutoCloseWarnForWindow(rhythm *scheduleRhythm, w scheduleRhythmWindow, taskIDs []string, key string) {
	ctx := context.Background()
	traceID := tracelog.NormalizeTraceID(rhythm.TaskID)
	if traceID == "" {
		traceID = tracelog.NewTraceID()
	}
	ctx = tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID: traceID,
		SpanID:  tracelog.NewSpanID(),
	})
	for _, taskID := range taskIDs {
		if err := notifyContainerClosingSoonFn(ctx, rhythm.TenantID, rhythm.WorkspaceID, taskID, map[string]interface{}{
			"reason":            "schedule_window_ending",
			"minutes_remaining": w.AutoCloseWarnMinutes,
			"top_task_id":       rhythm.TaskID,
			"daily_end":         w.DailyEnd,
			"timezone":          rhythm.Timezone,
		}); err != nil {
			log.Printf("[taskTaskService] event=schedule_auto_close_warn_failed task_id=%s err=%v", taskID, err)
			tracelog.LogForwardStage(ctx, "schedule_auto_close_warn_error", map[string]any{
				"task_id": taskID, "error": err.Error(),
			})
		}
	}
	_ = updateAutoCloseKeysForWindow(w.ID, key, "")
	_ = publishDomainEvent(ctx, "ScheduleAutoCloseWarned", map[string]interface{}{
		"top_task_id":  rhythm.TaskID,
		"tenant_id":    rhythm.TenantID,
		"workspace_id": rhythm.WorkspaceID,
		"task_ids":     taskIDs,
		"warn_key":     key,
		"window_id":    w.ID,
	}, rhythm.TaskID)
	tracelog.LogForwardStage(ctx, "schedule_auto_close_warn_ok", map[string]any{
		"top_task_id": rhythm.TaskID, "count": len(taskIDs), "warn_key": key,
	})
}

func releaseAutoCloseSlotsForWindow(rhythm *scheduleRhythm, w scheduleRhythmWindow, taskIDs []string, key string) {
	ctx := context.Background()
	traceID := tracelog.NormalizeTraceID(rhythm.TaskID)
	if traceID == "" {
		traceID = tracelog.NewTraceID()
	}
	ctx = tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID: traceID,
		SpanID:  tracelog.NewSpanID(),
	})
	modes := make(map[string]string, len(taskIDs))
	for _, taskID := range taskIDs {
		mode := "shutdown"
		if err := shutdownContainerLifecycleFn(ctx, rhythm.TenantID, rhythm.WorkspaceID, taskID, map[string]interface{}{
			"terminal_kind": "cancelled",
			"reason":        "schedule_window_end",
			"top_task_id":   rhythm.TaskID,
		}); err != nil {
			log.Printf("[taskTaskService] event=schedule_auto_close_shutdown_failed task_id=%s err=%v; fallback stop-vm", taskID, err)
			if err2 := stopVMFn(ctx, rhythm.TenantID, rhythm.WorkspaceID, taskID, map[string]interface{}{
				"task_id": taskID,
				"reason":  "schedule_window_end",
			}); err2 != nil {
				log.Printf("[taskTaskService] event=schedule_auto_close_stop_vm_failed task_id=%s err=%v", taskID, err2)
				mode = "failed"
			} else {
				mode = "stop_vm"
			}
		}
		modes[taskID] = mode
	}
	clearQueuedSlotsForTop(rhythm.TaskID)
	resetStartingMembershipsToQueued(taskIDs)
	_ = updateAutoCloseKeysForWindow(w.ID, "", key)
	_ = publishDomainEvent(ctx, "ScheduleAutoCloseReleased", map[string]interface{}{
		"top_task_id":  rhythm.TaskID,
		"tenant_id":    rhythm.TenantID,
		"workspace_id": rhythm.WorkspaceID,
		"task_ids":     taskIDs,
		"release_key":  key,
		"window_id":    w.ID,
		"modes":        modes,
	}, rhythm.TaskID)
	tracelog.LogForwardStage(ctx, "schedule_auto_close_release_ok", map[string]any{
		"top_task_id": rhythm.TaskID, "count": len(taskIDs), "release_key": key,
	})
}

func runQueuedScheduleAutoCloseOnce() {
	// 任务级（legacy）auto_close
	rows, err := db.Query(`SELECT DISTINCT r.task_id FROM task_top_deliverable_schedule_rhythms r
		INNER JOIN task_top_deliverable_schedule_rhythm_windows w ON w.task_id = r.task_id
		WHERE r.enabled=1 AND w.auto_close=1`)
	if err != nil {
		return
	}
	var tops []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			tops = append(tops, id)
		}
	}
	rows.Close()
	for _, topID := range tops {
		runAutoCloseForTop(topID)
	}

	// 工作空间级 auto_close
	wsRows, err := db.Query(`SELECT DISTINCT r.workspace_id FROM workspace_schedule_rhythms r
		INNER JOIN workspace_schedule_rhythm_windows w ON w.workspace_id = r.workspace_id
		WHERE r.enabled=1 AND w.auto_close=1`)
	if err != nil {
		return
	}
	var workspaces []string
	for wsRows.Next() {
		var id string
		if wsRows.Scan(&id) == nil && id != "" {
			workspaces = append(workspaces, id)
		}
	}
	wsRows.Close()
	for _, wsID := range workspaces {
		runAutoCloseForWorkspace(wsID)
	}
}

// ---- 工作空间级 auto_close（OPT：工作空间自动调度）----

func updateWorkspaceAutoCloseKeysForWindow(windowID, warnKey, releaseKey string) error {
	if warnKey == "" && releaseKey == "" {
		return nil
	}
	if warnKey != "" && releaseKey != "" {
		_, err := db.Exec(`UPDATE workspace_schedule_rhythm_windows SET auto_close_warn_key=?, auto_close_release_key=?, updated_at=? WHERE id=?`,
			warnKey, releaseKey, time.Now().UTC(), windowID)
		return err
	}
	if warnKey != "" {
		_, err := db.Exec(`UPDATE workspace_schedule_rhythm_windows SET auto_close_warn_key=?, updated_at=? WHERE id=?`,
			warnKey, time.Now().UTC(), windowID)
		return err
	}
	_, err := db.Exec(`UPDATE workspace_schedule_rhythm_windows SET auto_close_release_key=?, updated_at=? WHERE id=?`,
		releaseKey, time.Now().UTC(), windowID)
	return err
}

func listWorkspaceQueuedSlotTaskIDs(workspaceID string) ([]string, error) {
	rows, err := db.Query(`SELECT s.task_id FROM task_queued_machine_slots s
		INNER JOIN task_queued_auto_run_memberships m ON m.task_id = s.task_id
		WHERE m.workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func runAutoCloseForWorkspace(workspaceID string) {
	rhythm, err := loadWorkspaceScheduleRhythm(workspaceID)
	if err != nil || rhythm == nil || !rhythm.Enabled {
		return
	}
	windows, err := loadWorkspaceScheduleRhythmWindows(workspaceID)
	if err != nil || len(windows) == 0 {
		return
	}
	loc := resolveScheduleLocation(rhythm.Timezone)
	nowLocal := time.Now().In(loc)
	slots, err := listWorkspaceQueuedSlotTaskIDs(workspaceID)
	if err != nil || len(slots) == 0 {
		return
	}

	for _, w := range windows {
		if !w.AutoClose {
			continue
		}
		inWin := inDailyWindow(nowLocal, w.DailyStart, w.DailyEnd)
		if inWin {
			mins := minutesUntilWindowEnd(nowLocal, w.DailyStart, w.DailyEnd)
			lead := w.AutoCloseWarnMinutes
			if lead <= 0 {
				lead = 5
			}
			if mins <= 0 || mins > lead {
				continue
			}
			key := autoCloseKeyForInWindowEnd(nowLocal, w.DailyStart, w.DailyEnd)
			if key == "" || key == w.AutoCloseWarnKey {
				continue
			}
			notifyWorkspaceAutoCloseWarnForWindow(rhythm, w, slots, key)
			return
		}
	}

	for _, w := range windows {
		if !w.AutoClose {
			continue
		}
		key := autoCloseKeyForMostRecentEnd(nowLocal, w.DailyStart, w.DailyEnd)
		if key == "" || key == w.AutoCloseReleaseKey {
			continue
		}
		releaseWorkspaceAutoCloseSlotsForWindow(rhythm, w, slots, key)
		return
	}
}

func notifyWorkspaceAutoCloseWarnForWindow(rhythm *workspaceScheduleRhythm, w workspaceScheduleRhythmWindow, taskIDs []string, key string) {
	ctx := context.Background()
	traceID := tracelog.NormalizeTraceID(rhythm.WorkspaceID)
	if traceID == "" {
		traceID = tracelog.NewTraceID()
	}
	ctx = tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID: traceID,
		SpanID:  tracelog.NewSpanID(),
	})
	for _, taskID := range taskIDs {
		if err := notifyContainerClosingSoonFn(ctx, rhythm.TenantID, rhythm.WorkspaceID, taskID, map[string]interface{}{
			"reason":            "schedule_window_ending",
			"minutes_remaining": w.AutoCloseWarnMinutes,
			"workspace_id":      rhythm.WorkspaceID,
			"daily_end":         w.DailyEnd,
			"timezone":          rhythm.Timezone,
		}); err != nil {
			log.Printf("[taskTaskService] event=schedule_auto_close_warn_failed task_id=%s workspace_id=%s err=%v", taskID, rhythm.WorkspaceID, err)
			tracelog.LogForwardStage(ctx, "schedule_auto_close_warn_error", map[string]any{
				"task_id": taskID, "error": err.Error(),
			})
		}
	}
	_ = updateWorkspaceAutoCloseKeysForWindow(w.ID, key, "")
	_ = publishDomainEvent(ctx, "ScheduleAutoCloseWarned", map[string]interface{}{
		"workspace_id": rhythm.WorkspaceID,
		"tenant_id":    rhythm.TenantID,
		"task_ids":     taskIDs,
		"warn_key":     key,
		"window_id":    w.ID,
	}, rhythm.WorkspaceID)
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: rhythm.TenantID, WorkspaceID: rhythm.WorkspaceID, EventType: scheduleHistoryEventAutoCloseWarned,
		Message: "窗口即将结束，已发出关闭预告",
	})
	tracelog.LogForwardStage(ctx, "schedule_auto_close_warn_ok", map[string]any{
		"workspace_id": rhythm.WorkspaceID, "count": len(taskIDs), "warn_key": key,
	})
}

func releaseWorkspaceAutoCloseSlotsForWindow(rhythm *workspaceScheduleRhythm, w workspaceScheduleRhythmWindow, taskIDs []string, key string) {
	ctx := context.Background()
	traceID := tracelog.NormalizeTraceID(rhythm.WorkspaceID)
	if traceID == "" {
		traceID = tracelog.NewTraceID()
	}
	ctx = tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID: traceID,
		SpanID:  tracelog.NewSpanID(),
	})
	modes := make(map[string]string, len(taskIDs))
	for _, taskID := range taskIDs {
		mode := "shutdown"
		if err := shutdownContainerLifecycleFn(ctx, rhythm.TenantID, rhythm.WorkspaceID, taskID, map[string]interface{}{
			"terminal_kind": "cancelled",
			"reason":        "schedule_window_end",
			"workspace_id":  rhythm.WorkspaceID,
		}); err != nil {
			log.Printf("[taskTaskService] event=schedule_auto_close_shutdown_failed task_id=%s err=%v; fallback stop-vm", taskID, err)
			if err2 := stopVMFn(ctx, rhythm.TenantID, rhythm.WorkspaceID, taskID, map[string]interface{}{
				"task_id": taskID,
				"reason":  "schedule_window_end",
			}); err2 != nil {
				log.Printf("[taskTaskService] event=schedule_auto_close_stop_vm_failed task_id=%s err=%v", taskID, err2)
				mode = "failed"
			} else {
				mode = "stop_vm"
			}
		}
		modes[taskID] = mode
	}
	clearWorkspaceQueuedSlots(rhythm.WorkspaceID)
	resetStartingMembershipsToQueued(taskIDs)
	_ = updateWorkspaceAutoCloseKeysForWindow(w.ID, "", key)
	_ = publishDomainEvent(ctx, "ScheduleAutoCloseReleased", map[string]interface{}{
		"workspace_id": rhythm.WorkspaceID,
		"tenant_id":    rhythm.TenantID,
		"task_ids":     taskIDs,
		"release_key":  key,
		"window_id":    w.ID,
		"modes":        modes,
	}, rhythm.WorkspaceID)
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: rhythm.TenantID, WorkspaceID: rhythm.WorkspaceID, EventType: scheduleHistoryEventAutoCloseReleased,
		Message: "窗口结束，已释放排队占用",
	})
	tracelog.LogForwardStage(ctx, "schedule_auto_close_release_ok", map[string]any{
		"workspace_id": rhythm.WorkspaceID, "count": len(taskIDs), "release_key": key,
	})
}
