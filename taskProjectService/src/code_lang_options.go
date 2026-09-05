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
	maxCodeLangOptions      = 50
	maxCodeLangOptionLength = 64
)

var defaultCodeLangOptions = []string{"go", "rust", "js"}

func defaultCodeLangOptionsCopy() []string {
	out := make([]string, len(defaultCodeLangOptions))
	copy(out, defaultCodeLangOptions)
	return out
}

func normalizeCodeLangOptions(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return defaultCodeLangOptionsCopy(), nil
	}
	if len(raw) > maxCodeLangOptions {
		return nil, fmt.Errorf("options exceeds max %d", maxCodeLangOptions)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if len(v) > maxCodeLangOptionLength {
			return nil, fmt.Errorf("option %q exceeds max length %d", v, maxCodeLangOptionLength)
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return defaultCodeLangOptionsCopy(), nil
	}
	return out, nil
}

func loadCodeLangOptions(workspaceID string) ([]string, string, error) {
	var raw, updated string
	err := db.QueryRow(
		`SELECT options_json, updated_at FROM project_workspace_code_lang_options WHERE workspace_id=?`,
		workspaceID,
	).Scan(&raw, &updated)
	if err == sql.ErrNoRows {
		return defaultCodeLangOptionsCopy(), "", nil
	}
	if err != nil {
		return nil, "", err
	}
	var opts []string
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		return defaultCodeLangOptionsCopy(), updated, nil
	}
	norm, err := normalizeCodeLangOptions(opts)
	if err != nil {
		return defaultCodeLangOptionsCopy(), updated, nil
	}
	return norm, updated, nil
}

func upsertCodeLangOptions(workspaceID string, opts []string) (string, error) {
	now := time.Now().UTC()
	b, err := json.Marshal(opts)
	if err != nil {
		return "", err
	}
	_, err = db.Exec(
		`INSERT INTO project_workspace_code_lang_options (workspace_id, options_json, updated_at)
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

func codeLangOptionsResponse(opts []string, updatedAt string) map[string]interface{} {
	out := map[string]interface{}{
		"status":  "success",
		"options": opts,
	}
	if updatedAt != "" {
		out["updated_at"] = updatedAt
	}
	return out
}

func handleCodeLangOptions(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	traceID := r.Header.Get("X-Trace-Id")
	userID := strings.TrimSpace(getAuthUser(r))
	if userID == "" {
		logWarn(fmt.Sprintf("code-lang-options unauthorized tenant=%s workspace=%s", tenantID, workspaceID), traceID)
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
		logWarn(fmt.Sprintf("code-lang-options workspace missing tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
		writeError(w, r, http.StatusNotFound, "workspace not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		opts, updated, err := loadCodeLangOptions(workspaceID)
		if err != nil {
			logError(fmt.Sprintf("code-lang-options GET error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "load failed")
			return
		}
		logInfo(fmt.Sprintf("code-lang-options GET ok tenant=%s workspace=%s user=%s count=%d", tenantID, workspaceID, userID, len(opts)), traceID)
		writeJSON(w, http.StatusOK, codeLangOptionsResponse(opts, updated))
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
		norm, err := normalizeCodeLangOptions(parsed)
		if err != nil {
			logWarn(fmt.Sprintf("code-lang-options PUT validate fail tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := upsertCodeLangOptions(workspaceID, norm)
		if err != nil {
			logError(fmt.Sprintf("code-lang-options PUT error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "save failed")
			return
		}
		logInfo(fmt.Sprintf("code-lang-options PUT ok tenant=%s workspace=%s user=%s count=%d", tenantID, workspaceID, userID, len(norm)), traceID)
		writeJSON(w, http.StatusOK, codeLangOptionsResponse(norm, updated))
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
