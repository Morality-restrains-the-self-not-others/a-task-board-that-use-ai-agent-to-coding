package main

import (
	"net/http"
	"strings"
	"time"
)

func handleGetQueuedAutoRun(w http.ResponseWriter, r *http.Request, tenantID, userID, topTaskID string) {
	t, err := loadTask(topTaskID)
	if err != nil || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if userID != "internal" && !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	rhythm, _ := loadScheduleRhythm(topTaskID)
	members, err := listMembershipsForTop(topTaskID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		item := map[string]interface{}{
			"task_id":     m.TaskID,
			"top_task_id": m.TopTaskID,
			"depth":       m.Depth,
			"status":      m.Status,
			"enqueued_at": m.EnqueuedAt.UTC().Format(time.RFC3339Nano),
		}
		if m.Status == "deferred" {
			item["deferred_reason"] = deferredReason(rhythm)
		}
		items = append(items, item)
	}
	inWindow := rhythmInWindow(rhythm, time.Now())
	windows, _ := loadScheduleRhythmWindows(topTaskID)
	maxN := 0
	for _, w := range windows {
		maxN += w.MaxQueuedMachines
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"top_task_id":         topTaskID,
		"schedule_rhythm":     rhythmToJSON(rhythm),
		"in_window":           inWindow,
		"queued_slots_used":   countQueuedSlots(topTaskID),
		"max_queued_machines": maxN,
		"members":             items,
		"window_message":      windowMessage(rhythm, inWindow),
	})
}

func windowMessage(r *scheduleRhythm, inWindow bool) string {
	if r == nil || !r.Enabled {
		return "自动调度未启用"
	}
	if inWindow {
		return "当前处于允许运行时段"
	}
	return deferredReason(r)
}

// notifyManualStartDequeues is called from cloud proxy paths when available.
// Front/manual start-vm may also PATCH queued_auto_run=false; TTS also exposes this helper for internal hooks.
func notifyManualStartDequeues(taskID, actorUserID string) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	dequeueOnManualStart(taskID, actorUserID)
}
