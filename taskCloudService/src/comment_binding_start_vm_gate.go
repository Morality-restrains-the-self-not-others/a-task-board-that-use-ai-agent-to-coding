package main

// commentBindingBlocksStartVM 串行 waiting_previous 绑定禁止冷启动。
// binding 尚不存在时放行（由 @镜像 handler 的 has_unfinished_predecessors 门禁负责）。
func commentBindingBlocksStartVM(companyID, taskID, commentID string) (blocked bool, reason string) {
	commentID = trim(commentID)
	if commentID == "" || db == nil {
		return false, ""
	}
	b, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || b == nil {
		return false, ""
	}
	if b.ExecutionMode == ccbExecutionWaitPrevious && b.Status == ccbStatusWaitingPrevious {
		return true, "评论为串行等待前序，拒绝启动云主机"
	}
	return false, ""
}
