package aliyun

import "strings"

// ECSInstanceName 与评论容器名相同（task_{taskId}_{commentId}）。
// 启机与查找都按评论，缺少 task_id 或 comment_id 时返回空串，不回退任务级名。
func ECSInstanceName(taskID, commentID string) string {
	commentID = strings.TrimSpace(commentID)
	taskID = strings.TrimSpace(taskID)
	if commentID == "" || taskID == "" {
		return ""
	}
	if strings.HasPrefix(taskID, "task_") {
		return taskID + "_" + commentID
	}
	return "task_" + taskID + "_" + commentID
}

// ECSInstanceNames 查找用 InstanceName：仅评论级规范名。
func ECSInstanceNames(taskID, commentID string) []string {
	if name := ECSInstanceName(taskID, commentID); name != "" {
		return []string{name}
	}
	return nil
}
