package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type startVmFinalizeResult struct {
	EventID   string
	EventData map[string]interface{}
}

// startVmAutoFinalizeResult is an alias kept for call sites during Phase 3i cutover.
type startVmAutoFinalizeResult = startVmFinalizeResult

func finalizeStartVmInGo(ctx context.Context, in startVmFinalizeInput) (*startVmFinalizeResult, int, error) {
	eventData, buildErr := buildStartVmEventData(in)
	if buildErr != "" {
		return nil, http.StatusBadRequest, errors.New(buildErr)
	}

	taskID := strField(in.Body, "task_id")
	authID := strField(eventData, "authorization_id")
	if authID == "" && in.CloudAuth != nil {
		authID = in.CloudAuth.ID
		eventData["authorization_id"] = authID
	}
	if authID == "" {
		return nil, http.StatusBadRequest, errors.New("缺少 authorization_id")
	}

	platformType := strField(eventData, "cloud_platform_type")
	if platformType == "" && in.CloudAuth != nil {
		platformType = in.CloudAuth.PlatformType
	}
	if platformType == "" {
		if cpa, err := loadCloudAuth(in.TenantID, authID); err == nil && cpa != nil {
			platformType = cpa.PlatformType
			eventData["cloud_platform_type"] = platformType
		}
	}

	commentID := strings.TrimSpace(strField(in.Body, "comment_id"))
	if commentID == "" {
		return nil, http.StatusBadRequest, errors.New("缺少评论ID")
	}
	if bindErr := ensureCommentBindingForStartVM(in.TenantID, in.WorkspaceID, taskID, commentID, strField(in.Body, "execution_mode")); bindErr != nil {
		logWarn("finalizeStartVm: ensure comment binding: "+bindErr.Error(), in.TraceID)
	}

	accessToken, bootErr := bootstrapStartVmTokens(
		ctx,
		in.TenantID,
		in.WorkspaceID,
		taskID,
		commentID,
		authID,
		platformType,
		strField(eventData, "region_id"),
		strField(eventData, "zone_id"),
	)
	if bootErr != nil {
		return nil, http.StatusBadGateway, bootErr
	}
	eventData["userdata_access_token"] = accessToken
	fillStartVmEventCommentCSCID(in.TenantID, in.WorkspaceID, taskID, commentID, eventData)

	// Persist 镜像调用人员（评论人员），供审计与回收筛选。
	invoker := strings.TrimSpace(strField(in.Body, "image_invoker_user_id"))
	if invoker == "" {
		invoker = strings.TrimSpace(strField(in.Body, "comment_created_by_id"))
	}
	if invoker == "" {
		invoker = strings.TrimSpace(in.UserID)
	}
	if invoker != "" {
		_ = setCloudServerImageInvokerUserID(in.TenantID, in.WorkspaceID, taskID, invoker)
	}

	eventID, merged, err := insertPendingStartEvent(in.TenantID, in.WorkspaceID, taskID, in.UserID, commentID, eventData)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("持久化启动事件失败: %w", err)
	}
	// Persist verification_secret so server-userdata-verify callback can look it up.
	if s := strings.TrimSpace(strField(eventData, "verification_secret")); s != "" {
		if err := setCloudServerConfigVerificationSecret(in.TenantID, in.WorkspaceID, taskID, s); err != nil {
			logWarn("finalizeStartVm: failed to set verification_secret: "+err.Error(), in.TraceID)
		}
	}
	merged["event_id"] = eventID
	markCommentBindingStartingAfterStartAck(in.TenantID, taskID, commentID, strField(eventData, "csc_id"))

	return &startVmFinalizeResult{EventID: eventID, EventData: merged}, http.StatusOK, nil
}
