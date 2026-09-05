package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

func handleCommentContainerBindingsRoutes(w http.ResponseWriter, r *http.Request, tenantID, taskID, workspaceID, subPath string) {
	subPath = strings.Trim(subPath, "/")
	if subPath == "advance" || strings.HasPrefix(subPath, "advance/") {
		handleCommentContainerBindingsAdvance(w, r, tenantID, taskID, workspaceID)
		return
	}
	if commentID, ok := routeCommentContainerBindingCancel(subPath); ok {
		handleCommentContainerBindingCancel(w, r, tenantID, taskID, commentID)
		return
	}
	if strings.HasSuffix(subPath, "/complete") {
		parts := strings.Split(subPath, "/")
		if len(parts) >= 2 && parts[len(parts)-1] == "complete" {
			commentID := strings.Join(parts[:len(parts)-1], "/")
			handleCommentContainerBindingComplete(w, r, tenantID, taskID, commentID)
			return
		}
	}
	if subPath != "" {
		writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{"error": "not found", "path": subPath})
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleCommentContainerBindingsList(w, r, tenantID, taskID, workspaceID)
	case http.MethodPost:
		handleCommentContainerBindingsCreate(w, r, tenantID, taskID)
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleCommentContainerBindingsList(w http.ResponseWriter, r *http.Request, tenantID, taskID, workspaceID string) {
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "task_id required"})
		return
	}
	rows, err := listCommentContainerBindings(tenantID, taskID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	bindingsJSON := commentContainerBindingsToJSON(rows)
	// OPT-20260809-011: 附带服务端权威启动阶段事件时间线（按 comment 归组）
	logsByComment, err := listCommentContainerBindingLogsByCommentIn(workspaceID, tenantID, taskID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	for _, b := range bindingsJSON {
		cid := fmt.Sprint(b["comment_id"])
		if per, ok := logsByComment[cid]; ok {
			b["logs"] = commentContainerBindingLogsToJSON(per)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"bindings": bindingsJSON,
	})
}

func handleCommentContainerBindingsCreate(w http.ResponseWriter, r *http.Request, tenantID, taskID string) {
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "task_id required"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	commentID := strField(body, "comment_id")
	if commentID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "comment_id required"})
		return
	}
	dependsOn := strField(body, "depends_on_comment_id")
	if dependsOn == "" {
		if raw, ok := body["depends_on_comment_ids"]; ok {
			dependsOn = joinDependsOnCommentIDs(raw)
		}
	}
	row, created, err := ensureCommentContainerBinding(
		tenantID, taskID, commentID,
		strField(body, "execution_mode"),
		dependsOn,
		workspaceIDFromComputeRequest(r),
	)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	statusCode := http.StatusOK
	status := "exists"
	if created {
		statusCode = http.StatusCreated
		status = "success"
	}
	writeJSON(w, statusCode, map[string]interface{}{
		"status":  status,
		"binding": commentContainerBindingToJSON(row),
	})
}

func handleCommentContainerBindingsAdvance(w http.ResponseWriter, r *http.Request, tenantID, taskID, workspaceID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "task_id required"})
		return
	}
	result, err := advanceCommentContainerBindings(tenantID, taskID, workspaceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"advanced": commentContainerBindingsToJSON(result.Advanced),
		"blocked":  commentContainerBindingsToJSON(result.Blocked),
	})
}

func handleCommentContainerBindingComplete(w http.ResponseWriter, r *http.Request, tenantID, taskID, commentID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "task_id and comment_id required"})
		return
	}
	row, err := completeCommentContainerBinding(tenantID, taskID, commentID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{"status": "error", "message": "binding not found"})
			return
		}
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"binding": commentContainerBindingToJSON(row),
	})
}
