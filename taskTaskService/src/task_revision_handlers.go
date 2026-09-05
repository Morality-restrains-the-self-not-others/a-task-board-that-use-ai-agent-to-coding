package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

func parseRevisionPage(r *http.Request) (limit, offset int) {
	limit = 20
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func revisionIDFromPath(path string) string {
	i := strings.Index(path, "/revisions/")
	if i < 0 {
		return ""
	}
	rest := strings.Trim(path[i+len("/revisions/"):], "/")
	if rest == "" {
		return ""
	}
	return strings.Split(rest, "/")[0]
}

func handleTaskRevisionRoutes(w http.ResponseWriter, r *http.Request, tenantID, userID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, errMsgMethodNotAllowed)
		return
	}
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t == nil || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if workspaceID != "" && t.WorkspaceID != workspaceID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		logRevisionAuthDenied(tracelog.TraceIDFromContext(r.Context()), userID, taskID, "no_workspace_access")
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	revID := revisionIDFromPath(r.URL.Path)
	if revID == "" {
		handleListTaskRevisions(w, r, tenantID, taskID)
		return
	}
	handleGetTaskRevision(w, r, tenantID, taskID, revID)
}

func handleListTaskRevisions(w http.ResponseWriter, r *http.Request, tenantID, taskID string) {
	limit, offset := parseRevisionPage(r)
	rows, total, err := listTaskRevisions(taskID, tenantID, limit, offset)
	if err != nil {
		log.Printf("[taskTaskService] event=task_revision_list_failed task_id=%s err=%v", taskID, err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	results := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		results = append(results, taskRevisionListJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func handleGetTaskRevision(w http.ResponseWriter, r *http.Request, tenantID, taskID, revisionID string) {
	row, err := getTaskRevision(taskID, tenantID, revisionID)
	if err == sql.ErrNoRows || row == nil {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if err != nil {
		log.Printf("[taskTaskService] event=task_revision_get_failed task_id=%s revision_id=%s err=%v", taskID, revisionID, err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskRevisionDetailJSON(*row))
}
