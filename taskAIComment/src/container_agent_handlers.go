package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"
)

func containerAgentCommentToJSON(c *ContainerAgentComment) map[string]interface{} {
	out := map[string]interface{}{
		"id":                 c.ID,
		"tenant_id":          c.TenantID,
		"workspace_id":       c.WorkspaceID,
		"task_id":            c.TaskID,
		"parent_comment_id":  c.ParentCommentID,
		"installed_image_id": c.InstalledImageID,
		"run_status":         c.RunStatus,
		"created_at":         c.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":         c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if c.Content.Valid {
		out["content"] = c.Content.String
	} else {
		out["content"] = nil
	}
	if c.AssistantResponse.Valid {
		out["assistant_response"] = c.AssistantResponse.String
	} else {
		out["assistant_response"] = nil
	}
	if c.ContextPackJSON.Valid && c.ContextPackJSON.String != "" {
		if pack, err := parseContextPackJSON(c.ContextPackJSON.String); err == nil && pack != nil {
			out["context_pack"] = pack
		}
	}
	return out
}

func createPendingContainerAgentComment(body map[string]interface{}) (*ContainerAgentComment, int, map[string]string) {
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	parentCommentID := strField(body, "parent_comment_id")
	installedImageID := strField(body, "installed_image_id")
	content := strField(body, "content")

	if tenantID == "" || workspaceID == "" || taskID == "" {
		return nil, http.StatusBadRequest, map[string]string{"error": "tenant_id, workspace_id, task_id required"}
	}
	if parentCommentID == "" {
		return nil, http.StatusBadRequest, map[string]string{"error": "parent_comment_id required"}
	}
	if installedImageID == "" {
		return nil, http.StatusBadRequest, map[string]string{"error": "installed_image_id required"}
	}

	active, err := countActiveContainerAgentRunsByParent(parentCommentID)
	if err != nil {
		return nil, http.StatusInternalServerError, map[string]string{"error": err.Error()}
	}
	if err := ValidateOneActiveRun(active); err != nil {
		return nil, http.StatusConflict, map[string]string{"error": err.Error()}
	}

	comment := &ContainerAgentComment{
		TenantID:         tenantID,
		WorkspaceID:      workspaceID,
		TaskID:           taskID,
		ParentCommentID:  parentCommentID,
		InstalledImageID: installedImageID,
		RunStatus:        runStatusPending,
	}
	if content != "" {
		comment.Content = sql.NullString{String: content, Valid: true}
	}
	var pack map[string]interface{}
	if rawPack, ok := body["context_pack"].(map[string]interface{}); ok && rawPack != nil {
		pack = rawPack
	}
	// Pre-assign ID so context_pack can embed run_id = self before insert.
	comment.ID = newCommentID()
	if pack != nil {
		pack = enrichContextPackRunIDs(pack, comment.ID)
		packJSON, err := marshalContextPack(pack)
		if err != nil {
			return nil, http.StatusBadRequest, map[string]string{"error": "invalid context_pack"}
		}
		if packJSON != "" {
			comment.ContextPackJSON = sql.NullString{String: packJSON, Valid: true}
			log.Printf("[taskAIComment] event=context_pack_set id=%s bytes=%d", comment.ID, len(packJSON))
		}
	}
	if err := insertContainerAgentComment(comment); err != nil {
		return nil, http.StatusInternalServerError, map[string]string{"error": err.Error()}
	}
	log.Printf("[taskAIComment] event=container_agent_created id=%s task_id=%s parent_comment_id=%s run_status=%s",
		comment.ID, comment.TaskID, comment.ParentCommentID, comment.RunStatus)
	return comment, http.StatusCreated, nil
}

func handleInternalCreateContainerAgentComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	comment, status, errBody := createPendingContainerAgentComment(body)
	if errBody != nil {
		writeJSON(w, status, errBody)
		return
	}
	writeJSON(w, status, containerAgentCommentToJSON(comment))
}

func handleContainerAgentCommentRoutes(w http.ResponseWriter, r *http.Request) {
	parts := cleanPath(r, "/api/tenant/")
	if len(parts) < 6 || parts[1] != "workspace" || parts[3] != "task" || parts[5] != "container-agent-comments" {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	tenantID := parts[0]
	workspaceID := parts[2]
	taskID := parts[4]

	if len(parts) == 6 {
		switch r.Method {
		case http.MethodGet:
			handleListContainerAgentComments(w, r, tenantID, workspaceID, taskID)
		case http.MethodPost:
			handlePublicCreateContainerAgentComment(w, r, tenantID, workspaceID, taskID)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if len(parts) == 7 {
		id := parts[6]
		if r.Method == http.MethodGet {
			handleGetContainerAgentComment(w, r, tenantID, workspaceID, taskID, id)
			return
		}
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(parts) == 8 {
		id := parts[6]
		action := parts[7]
		switch action {
		case "stream":
			handleContainerAgentStream(w, r, tenantID, workspaceID, taskID, id)
		case "complete":
			handleContainerAgentComplete(w, r, tenantID, workspaceID, taskID, id)
		case "fail":
			handleContainerAgentFail(w, r, tenantID, workspaceID, taskID, id)
		default:
			writeError(w, r, http.StatusNotFound, "not found")
		}
		return
	}
	writeError(w, r, http.StatusNotFound, "not found")
}

func handlePublicCreateContainerAgentComment(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	// 鉴权：internal secret → 容器 X-Access-Token（与 stream/complete 对齐）→ 用户会话
	if !requireTaskAICommentInternalSecret(r) && !requireContainerAgentAuth(r, tenantID, workspaceID, taskID) {
		userID := getAuthUser(r)
		if userID == "" {
			writeError(w, r, http.StatusForbidden, "forbidden")
			return
		}
		if err := verifyTaskAccess(tenantID, workspaceID, taskID, userID); err != nil {
			if err.Error() == "forbidden" {
				writeError(w, r, http.StatusForbidden, "forbidden")
				return
			}
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	if strField(body, "tenant_id") == "" {
		body["tenant_id"] = tenantID
	}
	if strField(body, "workspace_id") == "" {
		body["workspace_id"] = workspaceID
	}
	if strField(body, "task_id") == "" {
		body["task_id"] = taskID
	}
	comment, status, errBody := createPendingContainerAgentComment(body)
	if errBody != nil {
		writeJSON(w, status, errBody)
		return
	}
	writeJSON(w, status, containerAgentCommentToJSON(comment))
}

func handleListContainerAgentComments(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	userID := getAuthUser(r)
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
		writeContainerAgentCommentsPage(w, r, taskID, page)
		return
	}
	rows, err := listContainerAgentCommentsByTask(taskID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for i := range rows {
		item := containerAgentCommentToJSON(&rows[i])
		applyAssistantPreview(item, page.preview, page.full)
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

func writeContainerAgentCommentsPage(w http.ResponseWriter, r *http.Request, taskID string, page commentPageParams) {
	rows, hasMore, err := listContainerAgentCommentsByTaskPage(taskID, page)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]interface{}, 0, len(rows))
	var lastCreatedAt time.Time
	var lastID string
	for i := range rows {
		item := containerAgentCommentToJSON(&rows[i])
		applyAssistantPreview(item, page.preview, page.full)
		out = append(out, item)
		lastCreatedAt = rows[i].CreatedAt
		lastID = rows[i].ID
	}
	writeJSON(w, http.StatusOK, commentPageResponse(out, hasMore, lastCreatedAt, lastID))
}

func handleGetContainerAgentComment(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, id string) {
	userID := getAuthUser(r)
	if err := verifyTaskAccess(tenantID, workspaceID, taskID, userID); err != nil {
		if err.Error() == "forbidden" {
			writeError(w, r, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	c, err := loadContainerAgentComment(id)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if c.TaskID != taskID || c.TenantID != tenantID || c.WorkspaceID != workspaceID {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, containerAgentCommentToJSON(c))
}

func loadContainerAgentForAction(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, id string) (*ContainerAgentComment, bool) {
	if !requireContainerAgentAuth(r, tenantID, workspaceID, taskID) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return nil, false
	}
	c, err := loadContainerAgentComment(id)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not found")
		return nil, false
	}
	if c.TaskID != taskID || c.TenantID != tenantID || c.WorkspaceID != workspaceID {
		writeError(w, r, http.StatusNotFound, "not found")
		return nil, false
	}
	return c, true
}

func handleContainerAgentStream(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, id string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c, ok := loadContainerAgentForAction(w, r, tenantID, workspaceID, taskID, id)
	if !ok {
		return
	}
	if isTerminalRunStatus(c.RunStatus) {
		writeError(w, r, http.StatusConflict, "run already terminal")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	chunk := rawStrField(body, "chunk")
	merged, updated, err := appendContainerAgentChunkBatched(id, chunk)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !updated {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if chunk != "" {
		_ = touchContainerAgentStreamingStatus(id)
	}
	traceID := r.Header.Get("X-Trace-Id")
	if chunk != "" {
		publishContainerAgentStreamSSE(r.Context(), taskID, id, "chunk", chunk, traceID)
	}
	log.Printf("[taskAIComment] event=container_agent_stream id=%s task_id=%s chunk_len=%d batched=1",
		id, taskID, len(chunk))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                 true,
		"id":                 id,
		"run_status":         runStatusStreaming,
		"assistant_response": merged,
	})
}

func handleContainerAgentComplete(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, id string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c, ok := loadContainerAgentForAction(w, r, tenantID, workspaceID, taskID, id)
	if !ok {
		return
	}
	if err := flushContainerAgentBatcher(id); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	finalText := strField(body, "assistant_response")
	if finalText == "" {
		if c.AssistantResponse.Valid {
			finalText = c.AssistantResponse.String
		}
	}
	if finalText != "" {
		if _, err := setContainerAgentAssistantResponse(id, finalText); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if _, err := updateContainerAgentRunStatus(id, runStatusCompleted); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	traceID := r.Header.Get("X-Trace-Id")
	publishContainerAgentStreamSSE(r.Context(), taskID, id, "done", "", traceID)
	log.Printf("[taskAIComment] event=container_agent_complete id=%s task_id=%s response_len=%d",
		id, taskID, len(finalText))
	removeContainerAgentBatcher(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                 true,
		"id":                 id,
		"run_status":         runStatusCompleted,
		"assistant_response": finalText,
	})
}

func handleContainerAgentFail(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, id string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	_, ok := loadContainerAgentForAction(w, r, tenantID, workspaceID, taskID, id)
	if !ok {
		return
	}
	if err := flushContainerAgentBatcher(id); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	errMsg := strField(body, "error")
	if errMsg == "" {
		errMsg = strField(body, "message")
	}
	if errMsg != "" {
		if _, err := setContainerAgentAssistantResponse(id, errMsg); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if _, err := updateContainerAgentRunStatus(id, runStatusFailed); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	traceID := r.Header.Get("X-Trace-Id")
	publishContainerAgentStreamSSE(r.Context(), taskID, id, "error", errMsg, traceID)
	log.Printf("[taskAIComment] event=container_agent_fail id=%s task_id=%s error_len=%d",
		id, taskID, len(errMsg))
	removeContainerAgentBatcher(id)
	writeErrorMap(w, r, http.StatusOK, map[string]interface{}{
		"ok":         true,
		"id":         id,
		"run_status": runStatusFailed,
		"error":      errMsg,
	})
}
