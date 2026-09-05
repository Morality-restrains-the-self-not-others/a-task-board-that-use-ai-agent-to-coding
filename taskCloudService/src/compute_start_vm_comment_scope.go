package main

import (
	"database/sql"
	"errors"
	"fmt"
)

// ensureCommentBindingForStartVM inserts a comment_container_binding if missing.
// Auto-run start-vm historically ACK'd before the task-detail page called ensure,
// so a failed/lost RunInstances had no row to drain errors onto and no advance
// retry until the user opened the task — visible as comment time ≪ Aliyun CreationTime.
// Existing rows keep their execution_mode (do not flip independent ↔ wait_previous).
func ensureCommentBindingForStartVM(companyID, workspaceID, taskID, commentID, executionMode string) error {
	commentID = trim(commentID)
	if commentID == "" {
		return nil
	}
	existing, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("load comment binding: %w", err)
	}
	if trim(executionMode) == "" {
		executionMode = ccbExecutionWaitPrevious
	}
	_, err = insertCommentContainerBinding(companyID, taskID, commentID, executionMode, "", workspaceID)
	if err != nil {
		return fmt.Errorf("insert comment binding: %w", err)
	}
	logInfo(fmt.Sprintf("event=start_vm_comment_binding_ensured task_id=%s comment_id=%s workspace_id=%s mode=%s",
		taskID, commentID, workspaceID, executionMode), taskID)
	return nil
}

// fillStartVmEventCommentCSCID copies the comment CSC id into eventData so
// persistStartVmInstanceBinding can pin instance_id to the comment row.
func fillStartVmEventCommentCSCID(companyID, workspaceID, taskID, commentID string, eventData map[string]interface{}) {
	if eventData == nil || trim(commentID) == "" {
		return
	}
	if trim(strField(eventData, "csc_id")) != "" {
		return
	}
	cfg, err := loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil || trim(cfg.ID) == "" {
		return
	}
	eventData["csc_id"] = cfg.ID
}

func markCommentBindingStartingAfterStartAck(companyID, taskID, commentID, cscID string) {
	commentID = trim(commentID)
	if commentID == "" {
		return
	}
	b, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || b == nil {
		return
	}
	if b.Status != ccbStatusPending && b.Status != ccbStatusWaitingPrevious {
		return
	}
	mockName := buildCommentMockContainerName(taskID, commentID)
	cscID = trim(cscID)
	var markErr error
	if cscID != "" {
		markErr = markCommentContainerBindingStartingWithCSC(b.ID, mockName, cscID)
	} else {
		markErr = updateCommentContainerBindingStatus(b.ID, ccbStatusStarting)
	}
	if markErr != nil {
		logWarn(fmt.Sprintf("event=start_vm_comment_binding_starting_failed task_id=%s comment_id=%s err=%v",
			taskID, commentID, markErr), taskID)
		return
	}
	logInfo(fmt.Sprintf("event=start_vm_comment_binding_starting task_id=%s comment_id=%s csc_id=%s",
		taskID, commentID, cscID), taskID)
}
