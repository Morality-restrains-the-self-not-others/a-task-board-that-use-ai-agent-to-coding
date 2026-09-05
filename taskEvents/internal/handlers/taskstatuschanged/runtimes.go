package taskstatuschanged

import (
	"strings"

	"taskEvents/internal/repository/cloudconfig"
)

// RuntimeLister lists comment-scoped CSCs for a task (injectable for tests).
type RuntimeLister interface {
	ListForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.TaskRuntimeList, error)
}

type defaultRuntimeLister struct{}

func (defaultRuntimeLister) ListForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.TaskRuntimeList, error) {
	return cloudconfig.ListByTask(companyID, workspaceID, taskID)
}

func (h *Handler) resolveRuntimes(companyID int64, workspaceID, taskID string) (*cloudconfig.TaskRuntimeList, error) {
	if h.Lister != nil {
		return h.Lister.ListForTask(companyID, workspaceID, taskID)
	}
	if h.Loader != nil {
		row, err := h.Loader.LoadForTask(companyID, workspaceID, taskID)
		if err != nil {
			return nil, err
		}
		out := &cloudconfig.TaskRuntimeList{Comments: []cloudconfig.ConfigRow{}}
		if row != nil {
			out.Comments = []cloudconfig.ConfigRow{*row}
		}
		return out, nil
	}
	return defaultRuntimeLister{}.ListForTask(companyID, workspaceID, taskID)
}

func filterReleaseable(rows []cloudconfig.ConfigRow) []cloudconfig.ConfigRow {
	out := make([]cloudconfig.ConfigRow, 0, len(rows))
	for _, row := range rows {
		if isReleaseable(row) {
			out = append(out, row)
		}
	}
	return out
}

// isReleaseable: terminal status must stop every comment-bound cloud node that still
// has a live binding — non-empty server_url, or any instance_id that is not already
// stopped/released. Starting/Pending WITH instance_id must release (RunInstances
// returns id before status becomes Running). A row with only launch_request_id
// (RunInstances in flight, no instance_id yet) is also releaseable so it is marked
// Released and a late-arriving VM is not left as an orphan (OPT-20260817-027).
func isReleaseable(row cloudconfig.ConfigRow) bool {
	if strings.TrimSpace(row.ServerURL) != "" {
		return true
	}
	if machineRuntimeHasInstanceToRelease(row.InstanceID, row.LastRuntimeStatus) {
		return true
	}
	return strings.TrimSpace(row.LaunchRequestID) != ""
}

// machineRuntimeHasInstanceToRelease is the terminal-release predicate (stricter than
// work-panel「已启动」计数：Starting/Pending with instance still need CLOUD_SERVER_STOPPED).
func machineRuntimeHasInstanceToRelease(instanceID, runtimeStatus string) bool {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(runtimeStatus))
	switch status {
	case "stopped", "stopping", "released", "terminated", "shutting-down", "shuttingdown":
		return false
	}
	return true
}

func stopPublishKey(taskID, commentID string) string {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return taskID
	}
	return taskID + ":" + commentID
}
