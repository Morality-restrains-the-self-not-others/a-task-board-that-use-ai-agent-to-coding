package main

import (
	"net/http"
	"strings"
	"time"
)

// handleWorkspaceQueueScheduleRoutes — /api/tenant/{tid}/workspace/{wid}/queue-schedule/
// GET/PUT 快照；GET .../history/ 分页调度历史。
func handleWorkspaceQueueScheduleRoutes(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	userID := getAuthUser(r)
	if userID != "internal" && !hasWorkspaceAccess(r.Context(), tenantID, workspaceID, userID) {
		msg := errMsgForbidden
		if r.Method == http.MethodGet {
			msg = errMsgForbiddenRead
		}
		writeError(w, r, http.StatusForbidden, msg)
		return
	}
	if isQueueScheduleHistoryPath(r.URL.Path) {
		if r.Method != http.MethodGet {
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleGetWorkspaceQueueScheduleHistory(w, r, tenantID, workspaceID)
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleGetWorkspaceQueueSchedule(w, r, tenantID, userID, workspaceID)
	case http.MethodPut:
		handlePutWorkspaceQueueSchedule(w, r, tenantID, userID, workspaceID)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func isQueueScheduleHistoryPath(path string) bool {
	p := strings.Trim(path, "/")
	return strings.HasSuffix(p, "/queue-schedule/history") || strings.Contains(p, "/queue-schedule/history/")
}

func buildWorkspaceQueueScheduleSnapshot(tenantID, workspaceID string) (map[string]interface{}, error) {
	rhythm, _ := loadWorkspaceScheduleRhythm(workspaceID)
	members, err := listWorkspaceQueueItems(workspaceID)
	if err != nil {
		return nil, err
	}
	inWindow := workspaceRhythmInWindow(rhythm, time.Now())
	windows, _ := loadWorkspaceScheduleRhythmWindows(workspaceID)
	maxN := 0
	for _, w := range windows {
		maxN += w.MaxQueuedMachines
	}
	recent, _, recentHasMore, histErr := listScheduleHistory(tenantID, workspaceID, "", 8)
	if histErr != nil {
		recent = []map[string]interface{}{}
		recentHasMore = false
	}
	if recent == nil {
		recent = []map[string]interface{}{}
	}
	return map[string]interface{}{
		"workspace_id":            workspaceID,
		"tenant_id":               tenantID,
		"schedule_rhythm":         workspaceRhythmToJSON(rhythm),
		"in_window":               inWindow,
		"window_message":          workspaceWindowMessage(rhythm, inWindow),
		"queued_slots_used":       countWorkspaceQueuedSlots(workspaceID),
		"max_queued_machines":     maxN,
		"members":                 members,
		"recent_history":          recent,
		"recent_history_has_more": recentHasMore,
	}, nil
}

func handleGetWorkspaceQueueSchedule(w http.ResponseWriter, r *http.Request, tenantID, userID, workspaceID string) {
	snap, err := buildWorkspaceQueueScheduleSnapshot(tenantID, workspaceID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func handlePutWorkspaceQueueSchedule(w http.ResponseWriter, r *http.Request, tenantID, userID, workspaceID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	if err := applyWorkspaceScheduleRhythmFromBody(tenantID, workspaceID, userID, body); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	snap, err := buildWorkspaceQueueScheduleSnapshot(tenantID, workspaceID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func handleGetWorkspaceQueueScheduleHistory(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	limit := clampHistoryLimit(r.URL.Query().Get("limit"))
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	items, next, hasMore, err := listScheduleHistory(tenantID, workspaceID, cursor, limit)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":       items,
		"next_cursor": next,
		"has_more":    hasMore,
	})
}
