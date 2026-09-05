package main

import (
	"strings"
)

// attachMachineOwnerHint adds container_running and machine owner binding fields
// for task-detail image-card UX. Safe no-op when cfg is nil or has no instance.
func attachMachineOwnerHint(resp map[string]interface{}, companyID, workspaceID, viewerTaskID string, cfg *CloudServerConfig) {
	if resp == nil {
		return
	}
	containerRunning := false
	instanceID := ""
	if cfg != nil {
		containerRunning = strings.TrimSpace(cfg.ServerURL) != ""
		instanceID = strings.TrimSpace(cfg.InstanceID)
	}
	if instanceID == "" {
		if v, ok := resp["instance_id"].(string); ok {
			instanceID = strings.TrimSpace(v)
		}
	}
	resp["container_running"] = containerRunning
	if instanceID == "" {
		return
	}

	bindings, err := listInstanceBindings(companyID, workspaceID, instanceID)
	if err != nil {
		logWarn("event=machine_owner_hint_bindings_failed company="+companyID+
			" workspace="+workspaceID+" instance="+instanceID+" err="+err.Error(), viewerTaskID)
		return
	}

	bound := make([]map[string]interface{}, 0, len(bindings))
	owners := make([]string, 0, len(bindings))
	for _, b := range bindings {
		tid := strings.TrimSpace(b.TaskID)
		if tid == "" {
			continue
		}
		released := b.TerminalReleased != 0
		hasContainer := strings.TrimSpace(b.ServerURL) != ""
		bound = append(bound, map[string]interface{}{
			"task_id":            tid,
			"has_container":      hasContainer,
			"terminal_released":  released,
		})
		if !released {
			owners = append(owners, tid)
		}
	}
	if len(bound) == 0 && strings.TrimSpace(viewerTaskID) != "" {
		bound = append(bound, map[string]interface{}{
			"task_id":           strings.TrimSpace(viewerTaskID),
			"has_container":     containerRunning,
			"terminal_released": false,
		})
		owners = append(owners, strings.TrimSpace(viewerTaskID))
	}
	resp["machine_bound_tasks"] = bound
	resp["machine_owner_task_ids"] = owners
}
