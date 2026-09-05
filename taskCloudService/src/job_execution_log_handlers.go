package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

func handleInternalJobExecutionEvents(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "failed to read body")
		return
	}
	body := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	row := jobExecutionEventFromMap(body)
	created, err := insertJobExecutionEvent(row)
	if err != nil {
		tracelog.LogForwardStage(r.Context(), "job_execution_event_persist_err", map[string]any{
			"error": err.Error(), "task_id": row.TaskID, "job_id": row.JobID,
		})
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "created": created})
}

func jobExecutionEventFromMap(body map[string]any) jobExecutionEventRow {
	row := jobExecutionEventRow{
		CompanyID:       jobStreamBodyString(body, "company_id"),
		WorkspaceID:     jobStreamBodyString(body, "workspace_id"),
		TaskID:          jobStreamBodyString(body, "task_id"),
		CommentID:       jobStreamBodyString(body, "comment_id"),
		JobID:           jobStreamBodyString(body, "job_id"),
		Seq:             jobStreamBodyInt(body, "seq"),
		Phase:           jobStreamBodyString(body, "phase"),
		Message:         jobStreamBodyString(body, "message"),
		StepNumber:      jobStreamBodyInt(body, "step_number"),
		DeliverySummary: jobStreamBodyString(body, "delivery_summary"),
		StepState:       jobStreamBodyString(body, "step_state"),
		JobStatus:       jobStreamBodyString(body, "job_status"),
		LayerID:         jobStreamBodyString(body, "layer_id"),
		EventJSON:       eventJSONString(body["event"]),
	}
	if row.StepState == "" {
		row.StepState = jobStreamBodyString(body, "state")
	}
	return row
}

func handleContainerJobExecutionLogFromDB(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed, use GET")
		return
	}
	q := r.URL.Query()
	jobID := strings.TrimSpace(q.Get("job_id"))
	commentID := commentIDFromComputeRequest(r, nil)
	if taskID == "" {
		taskID = strings.TrimSpace(q.Get("task_id"))
	}
	workspaceID = strings.TrimSpace(pathKV(r.URL.Path, "workspace_id"))
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(q.Get("workspace_id"))
	}
	if workspaceID == "" || taskID == "" {
		// 与层图快照 GET 对齐：必须带 workspace 才允许按 job 读执行日志，避免跨 workspace IDOR。
		writeErrorJSON(w, r, http.StatusBadRequest, "workspace_id and task_id are required")
		return
	}
	if jobID == "" && commentID != "" && taskID != "" {
		latest, err := latestJobIDForComment(workspaceID, taskID, commentID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		jobID = latest
	}
	if jobID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "job_id is required")
		return
	}
	afterStep := parseNonNegIntQuery(q.Get("after_step"), 0)
	limit := parseNonNegIntQuery(q.Get("limit"), 20)
	events, err := listJobExecutionEvents(workspaceID, taskID, commentID, jobID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	payload := reconstructJobExecutionLog(events, jobID, afterStep, limit)
	if lid := strings.TrimSpace(q.Get("layer_id")); lid != "" {
		if job, ok := payload["job"].(map[string]any); ok && job["layer_id"] == nil {
			job["layer_id"] = lid
		}
	}
	hydrateJobExecutionLogFromStepFull(r.Context(), payload, workspaceID, taskID, commentID, jobID, afterStep, limit)
	_ = tenantID
	writeJSON(w, http.StatusOK, payload)
}

func parseNonNegIntQuery(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func isContainerJobExecutionLogSub(sub string) bool {
	s := strings.Trim(sub, "/")
	return s == "compute/container-job-execution-log" ||
		strings.HasPrefix(s, "compute/container-job-execution-log/") ||
		strings.Contains(s, "/container-job-execution-log")
}
