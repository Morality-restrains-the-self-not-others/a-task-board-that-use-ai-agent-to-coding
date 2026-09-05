package main

import "log"

// recoverFailedCommentBindingToStarting 将永久失败的评论 binding 拉回 starting，
// 以便后续 promote / reachability 闭环。failed 一旦收口便不再被 advance 调度，
// 若之后评论级 start-vm 已成功或 CSC 已从 mock 升级为真实云平台，必须显式恢复。
func recoverFailedCommentBindingToStarting(b *CommentContainerBinding, reason string) {
	if b == nil || b.Status != ccbStatusFailed {
		return
	}
	mockName := resolveCommentMockContainerName(b.TaskID, b.CommentID, b.MockContainerName)
	if err := markCommentContainerBindingStartingWithCSC(b.ID, mockName, b.CSCID); err != nil {
		log.Printf("[taskCloudService] event=comment_container_binding_recover_failed binding_id=%s comment_id=%s reason=%s err=%v",
			b.ID, b.CommentID, reason, err)
		return
	}
	b.Status = ccbStatusStarting
	b.MockContainerName = mockName
	logCommentContainerBindingStageBestEffort(b, ccbStatusStarting)
	log.Printf("[taskCloudService] event=comment_container_binding_recovered binding_id=%s comment_id=%s csc_id=%s reason=%s",
		b.ID, b.CommentID, b.CSCID, reason)
}

// markCommentBindingFailedAfterStartError 仅在评论级 start-vm 真正失败时收口 failed。
// 不用于 mock 首轮 bootstrap，也不用于任务级 fan-out（避免误伤并行评论）。
func markCommentBindingFailedAfterStartError(b *CommentContainerBinding) {
	if b == nil {
		return
	}
	switch b.Status {
	case ccbStatusRunning, ccbStatusCompleted, ccbStatusReleased, ccbStatusCancelled:
		return
	}
	if trim(b.CSCID) == "" {
		cfg, err := loadCloudServerConfigForComment(b.CompanyID, b.WorkspaceID, b.TaskID, b.CommentID)
		if err == nil && cfg != nil {
			b.CSCID = cfg.ID
		}
	}
	mockName := resolveCommentMockContainerName(b.TaskID, b.CommentID, b.MockContainerName)
	if err := markCommentContainerBindingFailed(b.ID, mockName, b.CSCID); err != nil {
		log.Printf("[taskCloudService] event=comment_container_binding_start_error_fail_store_err binding_id=%s comment_id=%s err=%v",
			b.ID, b.CommentID, err)
		return
	}
	b.Status = ccbStatusFailed
	b.MockContainerName = mockName
	logCommentContainerBindingStageBestEffort(b, ccbStatusFailed)
	log.Printf("[taskCloudService] event=comment_container_binding_failed_after_start_error binding_id=%s comment_id=%s csc_id=%s",
		b.ID, b.CommentID, b.CSCID)
}

func recoverFailedCommentBindingForComment(companyID, taskID, commentID, cscID, reason string) {
	b, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || b == nil {
		return
	}
	if id := trim(cscID); id != "" {
		b.CSCID = id
	}
	recoverFailedCommentBindingToStarting(b, reason)
}
