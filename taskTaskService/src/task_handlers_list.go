package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// resolveSearchAllowedWorkspaces returns workspace IDs the caller may search.
// Fail-closed: only workspaces from project-service mine=1; never scan all task rows on lookup failure.
func resolveSearchAllowedWorkspaces(ctx context.Context, tenantID, userID, workspaceID string) (allowed []string, status int, errMsg string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, http.StatusUnauthorized, errMsgUnauthorized
	}
	accessible, err := listAccessibleWorkspaceIDs(ctx, tenantID, userID)
	if err != nil {
		log.Printf("[taskTaskService] tasks/search access lookup failed tenant=%s user=%s: %v", tenantID, userID, err)
		return nil, http.StatusServiceUnavailable, errMsgSearchAccessLookupFailed
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return accessible, 0, ""
	}
	for _, id := range accessible {
		if id == workspaceID {
			return []string{workspaceID}, 0, ""
		}
	}
	// Confirm existence for 404 vs 403 (do not leak tasks; only workspace metadata).
	if _, verr := verifyWorkspace(ctx, tenantID, workspaceID); verr != nil {
		return nil, http.StatusNotFound, errMsgWorkspaceNotFound
	}
	return nil, http.StatusForbidden, errMsgWorkspaceAccessUnavailable
}

// handleSearchTasks — tenant-scoped task search for billing filters and navbar.
// Paths (equivalent):
//
//	GET /api/tenant/{tid}/tasks/search/?q=&workspace_id=&limit=&assignee_ids=
//	GET /api/tasks/search/tenant_id/{tid}/?q=&workspace_id=&limit=&assignee_ids=
//
// Results are limited to workspaces the authenticated user can access (mine=1).
func handleSearchTasks(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, errMsgMethodNotAllowed)
		return
	}
	userID := getAuthUser(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	assigneeIDs := parseAssigneeIDsCSV(r.URL.Query().Get("assignee_ids"))
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, r, http.StatusBadRequest, errMsgLimitPositiveInteger)
			return
		}
		limit = n
	}
	if limit > 100 {
		limit = 100
	}

	allowed, status, errMsg := resolveSearchAllowedWorkspaces(r.Context(), tenantID, userID, workspaceID)
	if status != 0 {
		writeError(w, r, status, errMsg)
		return
	}

	log.Printf("[taskTaskService] tasks/search tenant=%s user=%s q_len=%d assignee_ids=%d workspaces=%d limit=%d",
		tenantID, userID, len(q), len(assigneeIDs), len(allowed), limit)

	hits, err := searchTasksInWorkspaces(tenantID, allowed, q, assigneeIDs, limit)
	if err != nil {
		log.Printf("[taskTaskService] tasks/search failed tenant=%s: %v", tenantID, err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	results := make([]map[string]interface{}, 0, len(hits))
	for _, hit := range hits {
		results = append(results, map[string]interface{}{
			"id":            hit.ID,
			"title":         hit.Title,
			"workspace_id":  hit.WorkspaceID,
			"workspace_seq": hit.WorkspaceSeq,
			"owner":         nilIfEmpty(hit.OwnerID),
			"operator":      nilIfEmpty(hit.OperatorID),
			"assignees":     hit.Assignees,
			"comment_id":    nilIfEmpty(hit.CommentID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func handleListTasks(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if workspaceID == "" {
		writeError(w, r, http.StatusBadRequest, errMsgWorkspaceIDRequired)
		return
	}
	if _, err := verifyWorkspace(r.Context(), tenantID, workspaceID); err != nil {
		writeError(w, r, http.StatusNotFound, err.Error())
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, workspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	rows, err := db.Query(`SELECT `+taskSelectCols+` FROM task_tasks WHERE tenant_id=? AND workspace_id=? ORDER BY order_num, created_at`, tenantID, workspaceID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var t taskRecord
		if err := scanTaskValues(rows.Scan, &t); err != nil {
			continue
		}
		out = append(out, taskToJSON(&t, tenantID))
	}
	enrichTasksCreatedBy(tenantID, out)
	writeJSON(w, http.StatusOK, out)
}

func handleListWorkspaceTasks(w http.ResponseWriter, r *http.Request, tenantID, userID, workspaceID string) {
	if _, err := verifyWorkspace(r.Context(), tenantID, workspaceID); err != nil {
		writeError(w, r, http.StatusNotFound, err.Error())
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, workspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	q := r.URL.Query()
	q.Set("workspace_id", workspaceID)
	r2 := *r
	r2.URL = &url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
	handleListTasks(w, &r2, tenantID, userID)
}
