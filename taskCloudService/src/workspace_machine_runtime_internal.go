package main

import (
	"fmt"
	"net/http"
)

// handleInternalReconcileWorkspaceMachineRuntimes 供 taskEvents cloud_csc_reconcile
// timer 一次性刷新 last_runtime_status（OPT-20260827-029）。看板 GET 不再 Describe。
func handleInternalReconcileWorkspaceMachineRuntimes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	n, err := reconcileAllWorkspaceMachineRuntimes()
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "success",
		"workspaces": n,
	})
}

func listWorkspacesWithMachineInstances() ([][2]string, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(
		`SELECT DISTINCT company_id, workspace_id
		 FROM cloud_server_configs
		 WHERE TRIM(COALESCE(comment_id,'')) != ''
		   AND TRIM(COALESCE(instance_id,'')) != ''`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][2]string
	for rows.Next() {
		var companyID, workspaceID string
		if err := rows.Scan(&companyID, &workspaceID); err != nil {
			return nil, err
		}
		companyID = trim(companyID)
		workspaceID = trim(workspaceID)
		if companyID == "" || workspaceID == "" {
			continue
		}
		out = append(out, [2]string{companyID, workspaceID})
	}
	return out, rows.Err()
}

func reconcileAllWorkspaceMachineRuntimes() (int, error) {
	pairs, err := listWorkspacesWithMachineInstances()
	if err != nil {
		return 0, err
	}
	for _, p := range pairs {
		purgeStaleContainerReachability(p[0], p[1])
		reconcileWorkspaceMachineRuntimes(p[0], p[1])
	}
	logInfo(fmt.Sprintf("event=workspace_machine_runtime_reconcile workspaces=%d", len(pairs)), "")
	return len(pairs), nil
}
