package main

import (
	"net/http"
	"strings"
)

func handleInternalDequeueQueuedAutoRun(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task id required")
		return
	}
	reason := "manual_start"
	if r.URL.Query().Get("reason") != "" {
		reason = r.URL.Query().Get("reason")
	}
	_ = dequeueQueuedAutoRun(taskID, reason, getAuthUser(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID, "reason": reason})
}
