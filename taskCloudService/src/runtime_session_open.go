package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

// handleInternalRuntimeSessionOpen implements POST /api/internal/runtime-session/open/
// Mirrors Django open_runtime_session_view (ensure cfg + close open histories + create history).
func handleInternalRuntimeSessionOpen(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "forbidden"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	runtimeSource := strField(body, "runtime_source")
	if runtimeSource == "" {
		runtimeSource = "relay_local"
	}
	ensureCfg := true
	if v, ok := body["ensure_cloud_server_config"]; ok {
		switch t := v.(type) {
		case bool:
			ensureCfg = t
		case string:
			ensureCfg = strings.EqualFold(strings.TrimSpace(t), "true") || t == "1"
		}
	}
	if tenantID == "" || taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "tenant_id and task_id required"})
		return
	}

	if ensureCfg {
		existing, err := loadCloudServerConfig(tenantID, workspaceID, taskID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
		if existing == nil || errors.Is(err, sql.ErrNoRows) {
			row := CloudServerConfig{
				ID:              genID("csc"),
				CompanyID:       tenantID,
				WorkspaceID:     workspaceID,
				TaskID:          taskID,
				Platform:        "relay-local",
				Region:          "local",
				ZoneID:          "local-a",
				AuthorizationID: "relay-" + taskID,
			}
			if err := upsertCloudServerConfig(row); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
				return
			}
		}
	}

	_, _ = closeOpenCloudServerConfigHistories(tenantID, workspaceID, taskID, map[string]interface{}{
		"stop_reason": "superseded_by_new_start",
		"stopped_at":  time.Now().UTC().Format("2006-01-02 15:04:05"),
	}, "")

	now := time.Now().UTC()
	h := CloudServerConfigHistory{
		ID:              genID("csh"),
		CompanyID:       tenantID,
		WorkspaceID:     workspaceID,
		TaskID:          taskID,
		Platform:        "mock",
		PlatformID:      1,
		Region:          "mock-local",
		ZoneID:          "mock-local-a",
		AuthorizationID: "mock-auth",
		RuntimeSource:   runtimeSource,
		CpuCores:        1,
		MemoryGB:        1,
		StorageGB:       40,
		StartedAt:       now,
		CreatedAt:       now,
	}
	if err := upsertCloudServerConfigHistory(h); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"session_id": h.ID,
	})
}
