package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// cancelCommentContainerBinding 用户终止「等待前序」的评论容器绑定。
// 仅允许 waiting_previous → cancelled；不启机、不释放云资源（尚未分配）。
func cancelCommentContainerBinding(companyID, taskID, commentID string) (*CommentContainerBinding, error) {
	b, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil {
		return nil, err
	}
	if b.Status != ccbStatusWaitingPrevious {
		return nil, fmt.Errorf("binding status %s cannot cancel (want waiting_previous)", b.Status)
	}
	if err := updateCommentContainerBindingStatus(b.ID, ccbStatusCancelled); err != nil {
		return nil, err
	}
	b.Status = ccbStatusCancelled
	b.UpdatedAt = time.Now().UTC()
	logCommentContainerBindingStageBestEffort(b, ccbStatusCancelled)
	log.Printf("[taskCloudService] event=comment_container_binding_cancelled binding_id=%s comment_id=%s task_id=%s",
		b.ID, b.CommentID, b.TaskID)
	return b, nil
}

func handleCommentContainerBindingCancel(w http.ResponseWriter, r *http.Request, tenantID, taskID, commentID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "task_id and comment_id required"})
		return
	}
	row, err := cancelCommentContainerBinding(tenantID, taskID, commentID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{"status": "error", "message": "binding not found"})
			return
		}
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	publishCommentContainerBindingAdvanced(tenantID, taskID, row)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"binding": commentContainerBindingToJSON(row),
	})
}

// routeCommentContainerBindingCancel 从 subPath 解析 commentId/cancel。
func routeCommentContainerBindingCancel(subPath string) (commentID string, ok bool) {
	subPath = strings.Trim(subPath, "/")
	if !strings.HasSuffix(subPath, "/cancel") {
		return "", false
	}
	parts := strings.Split(subPath, "/")
	if len(parts) < 2 || parts[len(parts)-1] != "cancel" {
		return "", false
	}
	return strings.Join(parts[:len(parts)-1], "/"), true
}
