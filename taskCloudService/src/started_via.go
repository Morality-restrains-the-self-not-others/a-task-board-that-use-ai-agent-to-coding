package main

import (
	"net/http"
	"strings"
)

// resolveStartedVia returns the persisted start source.
// queued_schedule is only accepted from internal/TTS callers (X-Internal-Secret or internal user).
func resolveStartedVia(r *http.Request, body map[string]interface{}) string {
	via := strings.TrimSpace(strField(body, "started_via"))
	if via == "" {
		return "manual"
	}
	if via == "queued_schedule" && !isTrustedStartedViaCaller(r) {
		return "manual"
	}
	switch via {
	case "queued_schedule", "auto_run", "manual", "reuse":
		return via
	default:
		return "manual"
	}
}

func isTrustedStartedViaCaller(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.TrimSpace(r.Header.Get("X-Internal-Secret")) != "" &&
		strings.TrimSpace(cfg.InternalSecret) != "" &&
		r.Header.Get("X-Internal-Secret") == cfg.InternalSecret {
		return true
	}
	uid := strings.TrimSpace(getAuthUser(r))
	return uid == "internal"
}

func persistStartedVia(companyID, taskID, via string) {
	via = strings.TrimSpace(via)
	companyID = strings.TrimSpace(companyID)
	taskID = strings.TrimSpace(taskID)
	if companyID == "" || taskID == "" || via == "" {
		return
	}
	_, _ = db.Exec(
		`UPDATE cloud_server_configs SET started_via=?, updated_at=CURRENT_TIMESTAMP WHERE company_id=? AND task_id=?`,
		via, companyID, taskID,
	)
}

func countQueuedScheduleRunning(companyID, workspaceID string, taskIDs []string) int {
	if len(taskIDs) == 0 {
		return 0
	}
	// MVP helper: count configs marked queued_schedule that look started/starting
	n := 0
	for _, tid := range taskIDs {
		var status string
		err := db.QueryRow(
			`SELECT COALESCE(last_runtime_status,'') FROM cloud_server_configs
			 WHERE company_id=? AND workspace_id=? AND task_id=? AND started_via='queued_schedule'`,
			companyID, workspaceID, tid,
		).Scan(&status)
		if err != nil {
			continue
		}
		s := strings.ToLower(strings.TrimSpace(status))
		if s == "" || s == "running" || s == "starting" || s == "pending" || s == "initializing" {
			n++
		}
	}
	return n
}
