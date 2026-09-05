package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	maxTaskKindOptions      = 50
	maxTaskKindOptionLength = 64
)

var defaultTaskKindOptions = []string{"bug-fix", "feature"}

func defaultTaskKindOptionsCopy() []string {
	out := make([]string, len(defaultTaskKindOptions))
	copy(out, defaultTaskKindOptions)
	return out
}

func normalizeTaskKindOptions(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return defaultTaskKindOptionsCopy(), nil
	}
	if len(raw) > maxTaskKindOptions {
		return nil, fmt.Errorf("options exceeds max %d", maxTaskKindOptions)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if len(v) > maxTaskKindOptionLength {
			return nil, fmt.Errorf("option %q exceeds max length %d", v, maxTaskKindOptionLength)
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return defaultTaskKindOptionsCopy(), nil
	}
	return out, nil
}

func loadTaskKindOptions(workspaceID string) ([]string, string, error) {
	var raw, updated string
	err := db.QueryRow(
		`SELECT options_json, updated_at FROM project_workspace_task_kind_options WHERE workspace_id=?`,
		workspaceID,
	).Scan(&raw, &updated)
	if err == sql.ErrNoRows {
		return defaultTaskKindOptionsCopy(), "", nil
	}
	if err != nil {
		return nil, "", err
	}
	var opts []string
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		return defaultTaskKindOptionsCopy(), updated, nil
	}
	norm, err := normalizeTaskKindOptions(opts)
	if err != nil {
		return defaultTaskKindOptionsCopy(), updated, nil
	}
	return norm, updated, nil
}

func upsertTaskKindOptions(workspaceID string, opts []string) (string, error) {
	now := time.Now().UTC()
	b, err := json.Marshal(opts)
	if err != nil {
		return "", err
	}
	_, err = db.Exec(
		`INSERT INTO project_workspace_task_kind_options (workspace_id, options_json, updated_at)
		 VALUES (?,?,?)
		 ON DUPLICATE KEY UPDATE
			options_json=VALUES(options_json),
			updated_at=VALUES(updated_at)`,
		workspaceID, string(b), now,
	)
	if err != nil {
		return "", err
	}
	return now.Format(time.RFC3339), nil
}

func taskKindOptionsResponse(opts []string, updatedAt string) map[string]interface{} {
	out := map[string]interface{}{
		"status":  "success",
		"options": opts,
	}
	if updatedAt != "" {
		out["updated_at"] = updatedAt
	}
	return out
}

func handleTaskKindOptions(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	traceID := r.Header.Get("X-Trace-Id")
	userID := strings.TrimSpace(getAuthUser(r))
	if userID == "" {
		logWarn(fmt.Sprintf("task-kind-options unauthorized tenant=%s workspace=%s", tenantID, workspaceID), traceID)
		writeError(w, r, http.StatusUnauthorized, "missing user")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	tenantID = strings.TrimSpace(tenantID)
	if workspaceID == "" || tenantID == "" {
		writeError(w, r, http.StatusBadRequest, "tenant_id and workspace_id required")
		return
	}
	if err := verifyWorkspaceInTenant(workspaceID, tenantID); err != nil {
		logWarn(fmt.Sprintf("task-kind-options workspace missing tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
		writeError(w, r, http.StatusNotFound, "workspace not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		opts, updated, err := loadTaskKindOptions(workspaceID)
		if err != nil {
			logError(fmt.Sprintf("task-kind-options GET error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "load failed")
			return
		}
		logInfo(fmt.Sprintf("task-kind-options GET ok tenant=%s workspace=%s user=%s count=%d", tenantID, workspaceID, userID, len(opts)), traceID)
		writeJSON(w, http.StatusOK, taskKindOptionsResponse(opts, updated))
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid JSON")
			return
		}
		rawOpts, ok := body["options"].([]interface{})
		if !ok {
			writeError(w, r, http.StatusBadRequest, "options must be an array of strings")
			return
		}
		parsed := make([]string, 0, len(rawOpts))
		for _, item := range rawOpts {
			parsed = append(parsed, fmt.Sprintf("%v", item))
		}
		norm, err := normalizeTaskKindOptions(parsed)
		if err != nil {
			logWarn(fmt.Sprintf("task-kind-options PUT validate fail tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := upsertTaskKindOptions(workspaceID, norm)
		if err != nil {
			logError(fmt.Sprintf("task-kind-options PUT error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "save failed")
			return
		}
		logInfo(fmt.Sprintf("task-kind-options PUT ok tenant=%s workspace=%s user=%s count=%d", tenantID, workspaceID, userID, len(norm)), traceID)
		writeJSON(w, http.StatusOK, taskKindOptionsResponse(norm, updated))
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
