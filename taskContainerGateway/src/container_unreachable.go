package main

import (
	"bytes"
	"context"
	"net/http"

	"tracelog"
)

// forwardToContainerService forwards to the online-service container and, when
// the dial fails with connection refused, notifies taskCloudService to clear the
// speculative server_url and demote the comment binding back to starting
// (OPT-20260818-008). Best-effort notify: cloud 侧不可达只记录日志，不改变转发结果。
func forwardToContainerService(ctx context.Context, sc scope, method, upstreamURL, accessToken string, body []byte) (int, []byte, error) {
	status, respBody, err := forwardToOnlineService(ctx, method, upstreamURL, accessToken, body)
	if status == http.StatusBadGateway && isConnRefusedBody(respBody) {
		notifyCloudContainerUnreachable(ctx, sc)
	}
	return status, respBody, err
}

func isConnRefusedBody(body []byte) bool {
	return bytes.Contains(bytes.ToLower(body), []byte("connection refused"))
}

func notifyCloudContainerUnreachable(ctx context.Context, sc scope) {
	payload := map[string]any{
		"tenant_id":    sc.TenantID,
		"workspace_id": sc.WorkspaceID,
		"task_id":      sc.TaskID,
		"comment_id":   sc.CommentID,
	}
	if err := cloudInternalPost(ctx, "/api/internal/cloud-server-config/container-unreachable/", payload); err != nil {
		tracelog.LogForwardStage(ctx, "container_unreachable_notify", map[string]any{
			"task_id": sc.TaskID, "comment_id": sc.CommentID, "ok": false, "detail": err.Error(),
		})
	}
}
