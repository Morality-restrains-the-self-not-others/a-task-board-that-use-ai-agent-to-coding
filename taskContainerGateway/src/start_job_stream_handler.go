package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tracelog"
)

func requireGatewayInternalSecret(r *http.Request) bool {
	if cfg.InternalSecret == "" {
		return true
	}
	return r.Header.Get("X-TaskContainerGateway-Internal-Secret") == cfg.InternalSecret
}

type startJobStreamRequest struct {
	TenantID    string `json:"tenant_id"`
	WorkspaceID string `json:"workspace_id"`
	TaskID      string `json:"task_id"`
	JobID       string `json:"job_id"`
	BaseURL     string `json:"base_url"`
	AccessToken string `json:"access_token"`
	TraceID     string `json:"trace_id"`
}

func handleStartJobStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use POST"})
		return
	}
	if !requireGatewayInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "forbidden"})
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "failed to read body"})
		return
	}
	var req startJobStreamRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json body"})
		return
	}
	sc := scope{
		TenantID:    strings.TrimSpace(req.TenantID),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		TaskID:      strings.TrimSpace(req.TaskID),
	}
	jobID := strings.TrimSpace(req.JobID)
	baseURL := strings.TrimSpace(req.BaseURL)
	accessToken := strings.TrimSpace(req.AccessToken)
	if sc.TenantID == "" || sc.WorkspaceID == "" || sc.TaskID == "" || jobID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail": "tenant_id, workspace_id, task_id, job_id required",
		})
		return
	}
	if baseURL == "" || accessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail": "base_url, access_token required",
		})
		return
	}

	ctx := r.Context()
	if tid := strings.TrimSpace(req.TraceID); tid != "" {
		ctx = tracelog.ContextWithTraceID(ctx, tid)
	}

	target := containerTarget{BaseURL: baseURL, AccessToken: accessToken}
	startJobStream(ctx, sc, target, jobID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "job_id": jobID})
}
