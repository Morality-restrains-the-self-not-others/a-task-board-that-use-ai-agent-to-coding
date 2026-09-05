package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"taskCloudService/domain"
	"tracelog"
)

func inboundBodySlice(body map[string]any, key string) ([]any, error) {
	if body == nil {
		return nil, fmt.Errorf("%s 须为 JSON 数组", key)
	}
	raw, ok := body[key]
	if !ok || raw == nil {
		return []any{}, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s 须为 JSON 数组", key)
	}
	return arr, nil
}

func handleJobStepFullPush(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	jobID := inboundBodyString(body, "job_id")
	if jobID == "" {
		writeErrorJSON(w, nil, http.StatusBadRequest, "job_id 必填")
		return
	}
	steps, err := inboundBodySlice(body, "steps")
	if err != nil {
		writeErrorJSON(w, nil, http.StatusBadRequest, err.Error())
		return
	}
	commentID := inboundBodyString(body, "comment_id")
	if cfgRow != nil {
		if c := strings.TrimSpace(cfgRow.CommentID); c != "" {
			commentID = c
		}
		if strings.TrimSpace(workspaceID) == "" {
			workspaceID = cfgRow.WorkspaceID
		}
		if strings.TrimSpace(taskID) == "" {
			taskID = cfgRow.TaskID
		}
	}
	id := domain.StepFullIDs{
		WorkspaceID: workspaceID,
		TaskID:      taskID,
		CommentID:   commentID,
		JobID:       jobID,
		LayerID:     inboundBodyString(body, "layer_id"),
	}
	companyID := strings.TrimSpace(tenantID)
	if companyID == "" && cfgRow != nil {
		companyID = strings.TrimSpace(cfgRow.CompanyID)
	}
	status := inboundBodyString(body, "job_status")
	if status == "" {
		status = inboundBodyString(body, "status")
	}
	key, err := persistStepFullArchive(ctx, id, companyID, status, steps)
	if err != nil {
		if strings.Contains(err.Error(), "exceeds") {
			writeErrorJSON(w, nil, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		writeErrorJSON(w, nil, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "object_key": key, "job_id": jobID})
}

func hydrateJobExecutionLogFromStepFull(ctx context.Context, payload map[string]any, workspaceID, taskID, commentID, jobID string, afterStep, limit int) {
	if payload == nil || strings.TrimSpace(jobID) == "" {
		return
	}
	row, found, err := getStepFullObjectRow(workspaceID, taskID, commentID, jobID)
	if err != nil {
		tracelog.LogForwardStage(ctx, "step_full_hydrate_row_err", map[string]any{"error": err.Error(), "job_id": jobID})
		return
	}
	if !found {
		row, found, err = latestStepFullObjectRowForComment(workspaceID, taskID, commentID, "")
		if err != nil || !found {
			return
		}
	}
	raw, src, err := loadStepFullBundleBytes(ctx, row.ObjectKey, row.PayloadJSON)
	if err != nil || len(raw) == 0 {
		return
	}
	bundle, err := domain.ParseStepFullBundle(raw)
	if err != nil {
		return
	}
	full := domain.StepsForJob(bundle, jobID)
	if full == nil {
		return
	}
	paged, total, nextAfter, hasMore := paginateAnySteps(full, afterStep, limit)
	payload["source"] = src
	if job, ok := payload["job"].(map[string]any); ok {
		job["source"] = src
		job["output_omitted"] = false
	}
	payload["steps"] = map[string]any{
		"steps":           paged,
		"total_steps":     total,
		"after_step":      afterStep,
		"next_after_step": nextAfter,
		"has_more":        hasMore,
		"source":          src,
	}
}

func paginateAnySteps(steps []any, afterStep, limit int) (paged []any, total int, nextAfter any, hasMore bool) {
	if afterStep < 0 {
		afterStep = 0
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	total = len(steps)
	var kept []any
	maxKept := 0
	for _, s := range steps {
		n := stepNumberFromAny(s)
		if n <= afterStep {
			continue
		}
		kept = append(kept, s)
		if n > maxKept {
			maxKept = n
		}
	}
	if len(kept) > limit {
		hasMore = true
		kept = kept[:limit]
		maxKept = 0
		for _, s := range kept {
			if n := stepNumberFromAny(s); n > maxKept {
				maxKept = n
			}
		}
	}
	if hasMore {
		nextAfter = maxKept
	}
	return kept, total, nextAfter, hasMore
}

func stepNumberFromAny(s any) int {
	m, ok := s.(map[string]any)
	if !ok {
		return 0
	}
	switch v := m["step_number"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n := 0
		_, _ = fmt.Sscanf(v, "%d", &n)
		return n
	default:
		return 0
	}
}
