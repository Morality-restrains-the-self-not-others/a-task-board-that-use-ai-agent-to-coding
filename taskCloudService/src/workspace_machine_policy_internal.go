package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleInternalWorkspaceMachinePolicy is the Credential-facing policy read.
// GET /api/internal/cloud/workspace-machine-policy/?company_id=&workspace_id=
func handleInternalWorkspaceMachinePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		logInfo("event=workspace_machine_policy_internal auth=denied", "")
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if companyID == "" || workspaceID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "company_id and workspace_id required")
		return
	}
	started := time.Now()
	p, err := loadWorkspaceMachinePolicy(companyID, workspaceID)
	if err != nil {
		logInfo("event=workspace_machine_policy_internal status=error company="+companyID+" err="+err.Error(), "")
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	body := workspaceMachinePolicyToJSON(p)
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	commentID := strings.TrimSpace(r.URL.Query().Get("comment_id"))
	if sts := maybeMachineReleaseSTS(companyID, workspaceID, taskID, commentID); sts != nil {
		body["machine_release_sts"] = sts
	}
	logInfo("event=workspace_machine_policy_internal status=ok company="+companyID+
		" minutes="+strconv.Itoa(p.IdleRecycleMinutes)+
		" duration_ms="+strconv.Itoa(int(time.Since(started).Milliseconds())), "")
	writeJSON(w, http.StatusOK, body)
}
