package main

import (
	"net/http"
	"strings"
)

type workspaceMachineCounts struct {
	StartedCount  int
	StartingCount int
	BusyCount     int
	IdleCount     int
}

func computeWorkspaceMachineCounts(companyID, workspaceID string) (workspaceMachineCounts, error) {
	snap, err := computeWorkspaceMachineSnapshot(companyID, workspaceID)
	if err != nil {
		return workspaceMachineCounts{}, err
	}
	return snap.Counts(), nil
}

func handleWorkspaceMachineSummary(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if tenantID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "无法获取租户ID",
		})
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(r.Header.Get("X-Workspace-Id"))
	}
	if workspaceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "无法获取工作区ID",
		})
		return
	}

	policy, err := loadWorkspaceMachinePolicy(tenantID, workspaceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	counts, err := computeWorkspaceMachineCounts(tenantID, workspaceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":               "success",
		"started_count":        counts.StartedCount,
		"starting_count":       counts.StartingCount,
		"idle_count":           counts.IdleCount,
		"busy_count":           counts.BusyCount,
		"idle_recycle_minutes": policy.IdleRecycleMinutes,
	})
}
