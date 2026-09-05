package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gatewaycors"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// traceIDForError resolves a trace id for error-body injection from the
// gateway-bridged X-Trace-Id header. Nil-safe for internal helpers without
// a request context.
func traceIDForError(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get("X-Trace-Id"))
}

// writeError writes a uniform {status, error, message, trace_id?} error body.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]interface{}{"status": "error", "error": message, "message": message}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorDetail writes a {status, error, detail, message, trace_id?} error body.
// Frontend reads data.detail first (taskAuth FE error contract), so detail-first.
func writeErrorDetail(w http.ResponseWriter, r *http.Request, status int, detail string) {
	body := map[string]interface{}{
		"status":  "error",
		"error":   detail,
		"detail":  detail,
		"message": detail,
	}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorMap writes a composite error body, injecting trace_id and filling
// in status/message (from error or detail) when absent while preserving any
// extra keys the caller provided.
func writeErrorMap(w http.ResponseWriter, r *http.Request, status int, body map[string]interface{}) {
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	if _, ok := body["status"]; !ok {
		body["status"] = "error"
	}
	if _, ok := body["message"]; !ok {
		if msg, ok := body["error"]; ok {
			body["message"] = msg
		} else if detail, ok := body["detail"]; ok {
			body["message"] = detail
		}
	}
	writeJSON(w, status, body)
}

func readJSONBody(r *http.Request) (map[string]interface{}, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func strField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func rawStrField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "taskAIComment",
	})
}

func cleanPath(r *http.Request, prefix string) []string {
	p := strings.TrimPrefix(r.URL.Path, prefix)
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func commentToJSON(c AIComment, users map[string]userProfile) map[string]interface{} {
	profile := users[c.CreatedByID]
	if profile.ID == "" {
		profile = userProfile{ID: c.CreatedByID, Username: c.CreatedByID, Email: ""}
	}
	// OPT-20260720-045: include member_name so the frontend can display author name
	// even when the user is not in workspace collaborators list.
	memberName := profile.MemberName
	if memberName == "" {
		memberName = profile.Username
	}
	out := map[string]interface{}{
		"id":              c.ID,
		"content":         c.Content,
		"task_id":         c.TaskID,
		"execution_mode":  c.ExecutionMode,
		"created_by":      map[string]interface{}{"id": profile.ID, "username": profile.Username, "email": profile.Email, "member_name": memberName},
		"created_at":      c.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":      c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if c.AssistantResponse.Valid {
		out["assistant_response"] = c.AssistantResponse.String
	} else {
		out["assistant_response"] = nil
	}
	return out
}

func handleListAIComments(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, userID string) {
	if err := verifyTaskAccess(tenantID, workspaceID, taskID, userID); err != nil {
		if err.Error() == "forbidden" {
			writeError(w, r, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	page := parseCommentPageQuery(r)
	if page.paginate {
		writeAICommentsPage(w, r, taskID, page)
		return
	}
	rows, err := listCommentsByTask(taskID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	userIDs := make([]string, 0, len(rows))
	for _, c := range rows {
		userIDs = append(userIDs, c.CreatedByID)
	}
	users := lookupUsers(userIDs)
	out := make([]map[string]interface{}, 0, len(rows))
	for _, c := range rows {
		item := commentToJSON(c, users)
		applyAssistantPreview(item, page.preview, page.full)
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

func writeAICommentsPage(w http.ResponseWriter, r *http.Request, taskID string, page commentPageParams) {
	rows, hasMore, err := listCommentsByTaskPage(taskID, page)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	userIDs := make([]string, 0, len(rows))
	for _, c := range rows {
		userIDs = append(userIDs, c.CreatedByID)
	}
	users := lookupUsers(userIDs)
	out := make([]map[string]interface{}, 0, len(rows))
	var lastCreatedAt time.Time
	var lastID string
	for _, c := range rows {
		item := commentToJSON(c, users)
		applyAssistantPreview(item, page.preview, page.full)
		out = append(out, item)
		lastCreatedAt = c.CreatedAt
		lastID = c.ID
	}
	writeJSON(w, http.StatusOK, commentPageResponse(out, hasMore, lastCreatedAt, lastID))
}

func forwardTestHeaders(r *http.Request) map[string]string {
	out := map[string]string{}
	if r.Header.Get("X-Task-Test-Skip-AI-Comment-Validate") == "1" {
		out["X-Task-Test-Skip-AI-Comment-Validate"] = "1"
	}
	return out
}

func handleCreateAIComment(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, userID string) {
	if err := verifyTaskAccess(tenantID, workspaceID, taskID, userID); err != nil {
		if err.Error() == "forbidden" {
			writeError(w, r, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	content := strField(body, "content")
	if content == "" {
		writeError(w, r, http.StatusBadRequest, "content required")
		return
	}
	executionMode, err := normalizeCommentExecutionMode(strField(body, "execution_mode"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	validatePayload := map[string]interface{}{
		"content":      content,
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
	}
	for _, key := range []string{"parent_job_id", "repo_layer_id", "command_kind", "agent_auto_iteration_count", "agent_models"} {
		if v, ok := body[key]; ok {
			validatePayload[key] = v
		}
	}
	if traceID := r.Header.Get("X-Trace-Id"); traceID != "" {
		validatePayload["trace_id"] = traceID
	}
	if r.Header.Get("X-Task-Test-Skip-AI-Comment-Validate") != "1" {
		if status, detail, err := validatePostBeforeCreate(r.Context(), validatePayload, forwardTestHeaders(r)); err != nil {
			writeErrorMap(w, r, http.StatusBadGateway, map[string]interface{}{
				"detail": "validation service unavailable",
				"error":  err.Error(),
			})
			return
		} else if status >= 400 {
			if detail == nil {
				detail = map[string]interface{}{"detail": "validation failed"}
			}
			writeJSON(w, status, detail)
			return
		}
	}

	comment := &AIComment{
		Content:       content,
		TaskID:        taskID,
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		CreatedByID:   userID,
		ExecutionMode: executionMode,
	}
	if err := insertComment(comment); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	jobCtx := map[string]interface{}{}
	for _, key := range []string{"parent_job_id", "repo_layer_id", "command_kind", "agent_auto_iteration_count", "agent_models"} {
		if v, ok := body[key]; ok {
			jobCtx[key] = v
		}
	}
	startInstructStreamAsync(instructStreamParams{
		CommentID:           comment.ID,
		TenantID:            tenantID,
		WorkspaceID:         workspaceID,
		TaskID:              taskID,
		UserContent:         content,
		ContainerJobContext: jobCtx,
		TraceID:             r.Header.Get("X-Trace-Id"),
	})

	meta := map[string]interface{}{
		"kind":    "ai_comment_created",
		"id":      comment.ID,
		"todo":    taskID,
		"content": content,
	}
	writeJSON(w, http.StatusOK, meta)
}

func handlePatchAIComment(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, commentID, userID string) {
	if err := verifyTaskAccess(tenantID, workspaceID, taskID, userID); err != nil {
		if err.Error() == "forbidden" {
			writeError(w, r, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	comment, err := loadComment(commentID)
	if err != nil || comment.TaskID != taskID {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	mode, err := normalizeCommentExecutionMode(strField(body, "execution_mode"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	prevMode := comment.ExecutionMode
	if prevMode == "" {
		prevMode = executionModeWaitPrevious
	}
	if mode == prevMode {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id": commentID, "execution_mode": mode,
		})
		return
	}
	if err := updateCommentExecutionMode(commentID, mode); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	logInfo(
		"event=comment_execution_mode_changed task_id="+taskID+
			" comment_id="+commentID+
			" previous="+prevMode+
			" execution_mode="+mode,
		r.Header.Get("X-Trace-Id"),
	)
	_ = publishDomainEvent(r.Context(), "CommentExecutionModeChanged", map[string]interface{}{
		"task_id":                 taskID,
		"tenant_id":               tenantID,
		"company_id":              tenantID,
		"workspace_id":            workspaceID,
		"comment_id":              commentID,
		"previous_execution_mode": prevMode,
		"execution_mode":          mode,
		"comment_kind":            "ai",
	}, taskID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": commentID, "execution_mode": mode,
	})
}

func handleAICommentRoutes(w http.ResponseWriter, r *http.Request) {
	parts := cleanPath(r, "/api/tenant/")
	if len(parts) < 6 || parts[1] != "workspace" || parts[3] != "task-detail" || parts[5] != "ai-comments" {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	tenantID := parts[0]
	workspaceID := parts[2]
	taskID := parts[4]
	userID := getAuthUser(r)
	if getAuthTenant(r) == "" {
		r.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	commentID := ""
	if len(parts) >= 7 {
		commentID = parts[6]
	}
	if commentID != "" {
		switch r.Method {
		case http.MethodPatch:
			handlePatchAIComment(w, r, tenantID, workspaceID, taskID, commentID, userID)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleListAIComments(w, r, tenantID, workspaceID, taskID, userID)
	case http.MethodPost:
		handleCreateAIComment(w, r, tenantID, workspaceID, taskID, userID)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
