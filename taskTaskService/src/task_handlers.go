package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func handleGetTaskByID(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	userID := getAuthUser(r)
	taskID := r.Header.Get("X-Task-Id")
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, errMsgTaskIDRequired)
		return
	}
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if tenantID != "" && t.TenantID != "" && t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if userID != "" && !hasWorkspaceAccess(r.Context(), t.TenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	writeJSON(w, http.StatusOK, taskJSONWithCommentsForDetail(t, t.TenantID))
}

func handleGetTask(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	writeJSON(w, http.StatusOK, taskJSONWithCommentsForDetail(t, tenantID))
}

func handleGetTaskSubtree(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	maxDepth := 2
	if raw := strings.TrimSpace(r.URL.Query().Get("max_depth")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, r, http.StatusBadRequest, "max_depth 须为正整数")
			return
		}
		if n > 20 {
			n = 20
		}
		maxDepth = n
	}
	payload, err := buildSubtreePayload(tenantID, t.WorkspaceID, taskID, maxDepth)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	workspaceID := t.WorkspaceID
	db.Exec(`DELETE FROM task_tasks WHERE id=?`, taskID)
	w.WriteHeader(http.StatusNoContent)
	if pubErr := publishTaskDeletedFn(r.Context(), tenantID, workspaceID, taskID); pubErr != nil {
		log.Printf("[taskTaskService] publish TASK_DELETED failed task_id=%s: %v", taskID, pubErr)
	}
}

func handleBatchDeleteTasks(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	ids := memberIDsFromBody(body, "ids")
	if len(ids) == 0 {
		writeError(w, r, http.StatusBadRequest, errMsgIDsRequired)
		return
	}
	deleted := 0
	for _, id := range ids {
		t, err := loadTask(id)
		if err != nil || t.TenantID != tenantID {
			continue
		}
		if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
			continue
		}
		workspaceID := t.WorkspaceID
		db.Exec(`DELETE FROM task_tasks WHERE id=?`, id)
		deleted++
		if pubErr := publishTaskDeletedFn(r.Context(), tenantID, workspaceID, id); pubErr != nil {
			log.Printf("[taskTaskService] publish TASK_DELETED failed task_id=%s: %v", id, pubErr)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": deleted})
}

// writePostExpiredError 返回创建帖已过期的 402 响应（v15 存续期模式）。
// 过期帖子拒绝执行/编辑/评论，续存后恢复。
func writePostExpiredError(w http.ResponseWriter) {
	writeErrorMap(w, nil, http.StatusPaymentRequired, map[string]interface{}{
		"detail": "任务帖已过期，请续存后继续操作",
		"code":   "TASK_POST_EXPIRED",
	})
}

// requirePostActive 拒绝在已过期帖子上执行编辑/评论/切换等变更操作。
func requirePostActive(w http.ResponseWriter, t *taskRecord) bool {
	if t.postExpired() {
		writePostExpiredError(w)
		return false
	}
	return true
}

func firstProjectID(body map[string]interface{}) string {
	projs, ok := body["projects"].([]interface{})
	if !ok || len(projs) == 0 {
		return ""
	}
	if item, ok := projs[0].(map[string]interface{}); ok {
		return strField(item, "project_id")
	}
	return ""
}

func strFieldDefault(m map[string]interface{}, key, def string) string {
	v := strField(m, key)
	if v == "" {
		return def
	}
	return v
}

func boolField(m map[string]interface{}, key string) bool {
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case float64:
		return v != 0
	default:
		return false
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
