package main

import (
	"net/http"
	"strings"
)

// workspaceRuntimeIndicator is the kanban-card signal for one task.
type workspaceRuntimeIndicator struct {
	TaskID                string `json:"task_id"`
	MachineRunning        bool   `json:"machine_running"`
	MachineStarting       bool   `json:"machine_starting"`
	ContainerRunning      bool   `json:"container_running"`
	RunningMachineCount   int    `json:"running_machine_count"`
	RunningContainerCount int    `json:"running_container_count"`
}

func handleWorkspaceRuntimeIndicators(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
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
	items, err := listWorkspaceRuntimeIndicators(tenantID, workspaceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "success",
		"indicators": items,
		"count":      len(items),
	})
}

// listWorkspaceRuntimeIndicators returns per-task machine/container flags for a workspace.
// 与头部摘要计数同源于 computeWorkspaceMachineSnapshot 的一次只读扫描。
func listWorkspaceRuntimeIndicators(companyID, workspaceID string) ([]workspaceRuntimeIndicator, error) {
	snap, err := computeWorkspaceMachineSnapshot(companyID, workspaceID)
	if err != nil {
		return nil, err
	}
	return snap.Indicators(), nil
}

// containerReachabilityCountsAsRunning: kanban「容器运行中」须同时具备可达登记与机器已启动。
func containerReachabilityCountsAsRunning(instanceID, runtimeStatus, serverURL string) bool {
	if strings.TrimSpace(serverURL) == "" {
		return false
	}
	return machineRuntimeCountsAsStarted(instanceID, runtimeStatus)
}

// purgeStaleContainerReachability clears server_url (and related fields) when the machine
// is no longer started — heals orphans that otherwise light the kanban while detail UI
// correctly hides「打开容器页面」after not-serving gate.
func purgeStaleContainerReachability(companyID, workspaceID string) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	if companyID == "" || workspaceID == "" || db == nil {
		return
	}
	rows, err := db.Query(
		`SELECT id, task_id, COALESCE(instance_id,''), COALESCE(server_url,''), COALESCE(last_runtime_status,'')
		 FROM cloud_server_configs
		 WHERE company_id=? AND workspace_id=?
		   AND TRIM(COALESCE(server_url,'')) != ''`,
		companyID, workspaceID,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	type staleRow struct {
		id     string
		taskID string
	}
	var stale []staleRow
	for rows.Next() {
		var id, taskID, instanceID, serverURL, lastStatus string
		if err := rows.Scan(&id, &taskID, &instanceID, &serverURL, &lastStatus); err != nil {
			return
		}
		status := effectiveMachineRuntimeStatus(instanceID, lastStatus)
		if containerReachabilityCountsAsRunning(instanceID, status, serverURL) {
			continue
		}
		id = trim(id)
		taskID = trim(taskID)
		if id == "" || taskID == "" {
			continue
		}
		stale = append(stale, staleRow{id: id, taskID: taskID})
	}
	if err := rows.Err(); err != nil {
		return
	}
	ids := make([]string, 0, len(stale))
	for _, s := range stale {
		ids = append(ids, s.id)
	}
	byID, err := loadCloudServerConfigsByIDs(ids)
	if err != nil {
		return
	}
	for _, s := range stale {
		cfg := byID[s.id]
		if cfg == nil {
			continue
		}
		_, _ = clearContainerReachabilityOnConfig(cfg, companyID, workspaceID, s.taskID)
	}
}
