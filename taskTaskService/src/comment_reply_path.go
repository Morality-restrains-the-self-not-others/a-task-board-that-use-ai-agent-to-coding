package main

import (
	"context"
	"net/http"
	"strings"
)

type commentReplyPathHints struct {
	workspaceID     string
	parentCommentID string
}

type commentReplyPathHintsKey struct{}

func parseTenantIDCommentReplyPath(urlPath string) (tenantID, workspaceID, taskID, parentID string, ok bool) {
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	// api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}
	if len(parts) < 9 {
		return
	}
	if parts[0] != "api" || parts[1] != "tenant_id" || parts[3] != "workspaceId" ||
		parts[5] != "tasks" || parts[7] != "comments" {
		return
	}
	tenantID = strings.TrimSpace(parts[2])
	workspaceID = strings.TrimSpace(parts[4])
	taskID = strings.TrimSpace(parts[6])
	parentID = strings.TrimSpace(parts[8])
	ok = tenantID != "" && workspaceID != "" && taskID != "" && parentID != ""
	return
}

func withCommentReplyPathHints(r *http.Request, workspaceID, parentCommentID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), commentReplyPathHintsKey{}, commentReplyPathHints{
		workspaceID:     workspaceID,
		parentCommentID: parentCommentID,
	}))
}

func commentReplyPathHintsFrom(r *http.Request) (commentReplyPathHints, bool) {
	h, ok := r.Context().Value(commentReplyPathHintsKey{}).(commentReplyPathHints)
	return h, ok
}

func commentBelongsToTask(taskID, commentID string) bool {
	taskID = strings.TrimSpace(taskID)
	commentID = strings.TrimSpace(commentID)
	if taskID == "" || commentID == "" {
		return false
	}
	var found string
	err := db.QueryRow(`SELECT id FROM task_comments WHERE id=? AND task_id=? LIMIT 1`, commentID, taskID).Scan(&found)
	return err == nil && found == commentID
}

func handleTenantIDPrefixedRoutes(w http.ResponseWriter, r *http.Request) {
	tid, wid, taskID, parentID, ok := parseTenantIDCommentReplyPath(r.URL.Path)
	if !ok {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, errMsgMethodNotAllowed)
		return
	}
	if getAuthTenant(r) == "" {
		r.Header.Set("X-Auth-Tenant-Id", tid)
	}
	if getAuthTenant(r) != tid {
		logInfo("event=comment_reply_tenant_mismatch path_tenant="+tid, taskID)
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !requireTenantMember(w, r, getAuthUser(r), tid) {
		return
	}
	handleCreateComment(w, withCommentReplyPathHints(r, wid, parentID), tid, getAuthUser(r), taskID)
}
