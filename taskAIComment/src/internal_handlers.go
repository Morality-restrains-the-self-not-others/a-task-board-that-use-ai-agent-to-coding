package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleInternalRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/task-ai-comment/")
	path = strings.TrimSuffix(path, "/")
	if path == "import" {
		handleImportAIComments(w, r)
		return
	}
	if strings.HasPrefix(path, "tasks/") {
		handleInternalTaskCommentLists(w, r, path)
		return
	}
	if path == "container-agent-comments" {
		handleInternalCreateContainerAgentComment(w, r)
		return
	}
	if path == "container-agent-comments/active-by-task" {
		handleInternalActiveContainerAgentByTask(w, r)
		return
	}
	if strings.HasPrefix(path, "container-agent-comments/by-parent/") && strings.HasSuffix(path, "/status") {
		parentID := strings.TrimPrefix(path, "container-agent-comments/by-parent/")
		parentID = strings.TrimSuffix(parentID, "/status")
		parentID = strings.Trim(parentID, "/")
		handleInternalPatchContainerAgentStatusByParent(w, r, parentID)
		return
	}
	if strings.HasPrefix(path, "container-agent-comments/") && strings.HasSuffix(path, "/status") {
		id := strings.TrimPrefix(path, "container-agent-comments/")
		id = strings.TrimSuffix(id, "/status")
		id = strings.Trim(id, "/")
		if id != "" && !strings.Contains(id, "/") {
			handleInternalPatchContainerAgentComment(w, r, id)
			return
		}
	}
	if strings.HasPrefix(path, "container-agent-comments/") {
		id := strings.TrimPrefix(path, "container-agent-comments/")
		id = strings.Trim(id, "/")
		if id != "" && !strings.Contains(id, "/") {
			handleInternalPatchContainerAgentComment(w, r, id)
			return
		}
	}
	if strings.HasSuffix(path, "/assistant-response") {
		id := strings.TrimSuffix(path, "/assistant-response")
		id = strings.TrimSuffix(id, "/")
		handlePatchAssistantResponse(w, r, id)
		return
	}
	writeError(w, r, http.StatusNotFound, "not found")
}

func handleInternalTaskCommentLists(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	rest := strings.TrimPrefix(path, "tasks/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	taskID := strings.TrimSpace(parts[0])
	kind := parts[1]
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task_id required")
		return
	}
	page := parseCommentPageQuery(r)
	switch kind {
	case "ai-comments":
		if page.paginate {
			writeAICommentsPage(w, r, taskID, page)
			return
		}
		rows, err := listCommentsByTask(taskID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]map[string]interface{}, 0, len(rows))
		for _, c := range rows {
			item := commentToJSON(c, map[string]userProfile{})
			applyAssistantPreview(item, page.preview, page.full)
			out = append(out, item)
		}
		writeJSON(w, http.StatusOK, out)
	case "container-agent-comments":
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
	default:
		writeError(w, r, http.StatusNotFound, "not found")
	}
}

func handleInternalActiveContainerAgentByTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task_id required")
		return
	}
	c, err := loadLatestActiveContainerAgentByTask(taskID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if c == nil {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, containerAgentCommentToJSON(c))
}

func handleInternalPatchContainerAgentComment(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if id == "" {
		writeError(w, r, http.StatusBadRequest, "id required")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	var runStatusPtr *string
	if _, ok := body["run_status"]; ok {
		rs := strField(body, "run_status")
		switch rs {
		case runStatusStarting, runStatusRunning, runStatusPending, runStatusStreaming:
			runStatusPtr = &rs
		case "":
			writeError(w, r, http.StatusBadRequest, "run_status required when provided")
			return
		default:
			writeError(w, r, http.StatusBadRequest, "unsupported run_status")
			return
		}
	}
	var packJSONPtr *string
	if rawPack, ok := body["context_pack"]; ok && rawPack != nil {
		packMap, ok := rawPack.(map[string]interface{})
		if !ok {
			writeError(w, r, http.StatusBadRequest, "context_pack must be object")
			return
		}
		packMap = enrichContextPackRunIDs(packMap, id)
		packJSON, err := marshalContextPack(packMap)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid context_pack")
			return
		}
		log.Printf("[taskAIComment] event=context_pack_patch id=%s bytes=%d", id, len(packJSON))
		packJSONPtr = &packJSON
	}
	if runStatusPtr == nil && packJSONPtr == nil {
		writeError(w, r, http.StatusBadRequest, "context_pack or run_status required")
		return
	}
	ok, err := updateContainerAgentPatch(id, runStatusPtr, packJSONPtr)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	c, err := loadContainerAgentComment(id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "id": id})
		return
	}
	writeJSON(w, http.StatusOK, containerAgentCommentToJSON(c))
}

func handleInternalPatchContainerAgentStatusByParent(w http.ResponseWriter, r *http.Request, parentCommentID string) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	parentCommentID = strings.TrimSpace(parentCommentID)
	if parentCommentID == "" {
		writeError(w, r, http.StatusBadRequest, "parent_comment_id required")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	status := strField(body, "run_status")
	switch status {
	case runStatusStarting, runStatusRunning, runStatusPending, runStatusStreaming:
	case "":
		writeError(w, r, http.StatusBadRequest, "run_status required")
		return
	default:
		writeError(w, r, http.StatusBadRequest, "unsupported run_status")
		return
	}
	c, err := loadLatestActiveContainerAgentByParent(parentCommentID)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	ok, err := updateContainerAgentRunStatus(c.ID, status)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	updated, err := loadContainerAgentComment(c.ID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok": true, "id": c.ID, "parent_comment_id": parentCommentID, "run_status": status,
		})
		return
	}
	writeJSON(w, http.StatusOK, containerAgentCommentToJSON(updated))
}

func handlePatchAssistantResponse(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if id == "" {
		writeError(w, r, http.StatusBadRequest, "id required")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	text := strField(body, "assistant_response")
	if text == "" {
		if v, ok := body["assistant_text"]; ok && v != nil {
			text = strings.TrimSpace(strings.Trim(fmt.Sprintf("%v", v), " "))
		}
	}
	if text == "" {
		writeError(w, r, http.StatusBadRequest, "assistant_response required")
		return
	}
	ok, err := updateAssistantResponse(id, text)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "id": id})
}

func handleImportAIComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireTaskAICommentInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid body")
		return
	}
    var payload struct {
		Comments []map[string]interface{} `json:"comments"`
		Rows     []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	items := payload.Comments
	if len(items) == 0 {
		items = payload.Rows
	}
	if len(items) == 0 {
		writeError(w, r, http.StatusBadRequest, "comments required")
		return
	}
	rows := make([]AIComment, 0, len(items))
	for _, item := range items {
		c := AIComment{
			ID:          strField(item, "id"),
			Content:     strField(item, "content"),
			TaskID:      strField(item, "task_id"),
			TenantID:    strField(item, "tenant_id"),
			WorkspaceID: strField(item, "workspace_id"),
			CreatedByID: strField(item, "created_by_id"),
		}
		if c.TaskID == "" {
			c.TaskID = strField(item, "task")
		}
		if c.CreatedByID == "" {
			c.CreatedByID = strField(item, "created_by")
		}
		if ar := strField(item, "assistant_response"); ar != "" {
			c.AssistantResponse = sql.NullString{String: ar, Valid: true}
		}
		if ts := strField(item, "created_at"); ts != "" {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				c.CreatedAt = t
			}
		}
		if ts := strField(item, "updated_at"); ts != "" {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				c.UpdatedAt = t
			}
		}
		rows = append(rows, c)
	}
	count, err := importComments(rows)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": count})
}
