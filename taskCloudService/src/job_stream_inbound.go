package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

var publishJobStreamSSE = publishTaskSSE

func jobStreamBodyString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	v := strings.TrimSpace(fmt.Sprintf("%v", body[key]))
	if v == "" || v == "<nil>" {
		return ""
	}
	return v
}

func jobStreamBodyInt(body map[string]any, key string) int {
	if body == nil {
		return 0
	}
	switch v := body[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprintf("%v", v)))
		return n
	}
}

func eventMapFromBody(body map[string]any) map[string]any {
	if body == nil {
		return nil
	}
	raw, ok := body["event"]
	if !ok || raw == nil {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	return m
}

// buildJobStreamSSEStatusData 组装 Kafka SSE_MESSAGE.status_data（不写库）。
func buildJobStreamSSEStatusData(body map[string]any, jobID string) map[string]any {
	phase := jobStreamBodyString(body, "phase")
	if phase == "" {
		phase = "chunk"
	}
	statusData := map[string]any{
		"status":     "container_job_stream",
		"event_name": "server_status_update",
		"job_id":     jobID,
		"phase":      phase,
		"message":    jobStreamBodyString(body, "message"),
		"seq":        jobStreamBodyInt(body, "seq"),
	}
	if ev := eventMapFromBody(body); ev != nil {
		statusData["event"] = ev
		if sn := jobStreamBodyInt(ev, "step_number"); sn > 0 {
			statusData["step_number"] = sn
		}
		if ds := jobStreamBodyString(ev, "delivery_summary"); ds != "" {
			statusData["delivery_summary"] = ds
		}
		if st := jobStreamBodyString(ev, "state"); st != "" {
			statusData["step_state"] = st
		}
	}
	if sn := jobStreamBodyInt(body, "step_number"); sn > 0 {
		statusData["step_number"] = sn
	}
	if ds := jobStreamBodyString(body, "delivery_summary"); ds != "" {
		statusData["delivery_summary"] = ds
	}
	if st := jobStreamBodyString(body, "state"); st != "" {
		statusData["step_state"] = st
	}
	if js := jobStreamBodyString(body, "job_status"); js != "" {
		statusData["job_status"] = js
	}
	if lid := jobStreamBodyString(body, "layer_id"); lid != "" {
		statusData["layer_id"] = lid
	}
	return statusData
}

func handleJobStreamPush(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	jobID := jobStreamBodyString(body, "job_id")
	if jobID == "" {
		writeErrorJSON(w, nil, http.StatusBadRequest, "job_id 必填")
		return
	}
	statusData := buildJobStreamSSEStatusData(body, jobID)
	commentID := jobStreamBodyString(body, "comment_id")
	companyID := strings.TrimSpace(tenantID)
	wsID := strings.TrimSpace(workspaceID)
	if cfgRow != nil {
		if c := strings.TrimSpace(cfgRow.CommentID); c != "" {
			commentID = c
		}
		if c := strings.TrimSpace(cfgRow.CompanyID); c != "" {
			companyID = c
		}
		if w := strings.TrimSpace(cfgRow.WorkspaceID); w != "" {
			wsID = w
		}
	}
	if commentID != "" {
		statusData["comment_id"] = commentID
	}
	if companyID != "" {
		statusData["company_id"] = companyID
	}
	if wsID != "" {
		statusData["workspace_id"] = wsID
	}
	start := tracelog.TraceIDFromContext(ctx)
	err := publishJobStreamSSE(ctx, taskID, commentID, statusData)
	tracelog.LogForwardStage(ctx, "job_stream_push_publish", map[string]any{
		"ok":         err == nil,
		"task_id":    taskID,
		"job_id":     jobID,
		"phase":      statusData["phase"],
		"comment_id": commentID,
		"trace_id":   start,
	})
	if err != nil {
		writeErrorJSON(w, nil, http.StatusBadGateway, "kafka publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"task_id": taskID,
		"job_id":  jobID,
	})
}
