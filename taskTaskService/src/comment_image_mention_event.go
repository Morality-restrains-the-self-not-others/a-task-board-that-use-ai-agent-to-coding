package main

import "strings"

// imageMentionHasUnfinishedPredecessors 决定 @镜像 事件是否应声明「仍有未完成前序」。
// independent 不等待；wait_previous 在存在前序评论或显式 depends_on 时为 true。
func imageMentionHasUnfinishedPredecessors(executionMode string, predecessorCount int, dependsOnIDs []string) bool {
	if executionMode == executionModeIndependent {
		return false
	}
	if len(dependsOnIDs) > 0 {
		return true
	}
	return predecessorCount > 0
}

func countOtherTaskComments(taskID, excludeID string) int {
	taskID = strings.TrimSpace(taskID)
	excludeID = strings.TrimSpace(excludeID)
	if taskID == "" || db == nil {
		return 0
	}
	var n int
	q := `SELECT COUNT(*) FROM task_comments WHERE task_id=? AND id<>?`
	if err := db.QueryRow(q, taskID, excludeID).Scan(&n); err != nil {
		return 0
	}
	return n
}

func buildTaskCommentImageMentionedData(
	taskID, tenantID, workspaceID, commentID, imageID, imageName, content, userID, executionMode string,
	dependsOnIDs []string,
) map[string]interface{} {
	n := countOtherTaskComments(taskID, commentID)
	data := map[string]interface{}{
		"task_id":                     taskID,
		"tenant_id":                   tenantID,
		"company_id":                  tenantID,
		"workspace_id":                workspaceID,
		"comment_id":                  commentID,
		"parent_comment_id":           commentID,
		"installed_image_id":          imageID,
		"installed_image_name":        imageName,
		"content":                     content,
		"created_by_id":               userID,
		"execution_mode":              executionMode,
		"has_unfinished_predecessors": imageMentionHasUnfinishedPredecessors(executionMode, n, dependsOnIDs),
	}
	if len(dependsOnIDs) > 0 {
		data["depends_on_comment_ids"] = dependsOnIDs
	}
	return data
}
