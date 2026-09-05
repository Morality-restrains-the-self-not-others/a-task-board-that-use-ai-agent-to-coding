package taskstatuschanged

import (
	"context"
	"fmt"
	"log"
	"strings"

	"taskEvents/internal/repository/cloudconfig"
	"tracelog"
)

type releaseKind int

const (
	releaseNone releaseKind = iota
	releaseGraceful
	releaseHard
)

// markCommentTerminalReleasedFn is overridable in tests (OPT-20260817-027).
var markCommentTerminalReleasedFn = cloudconfig.MarkCommentTerminalReleased

func (h *Handler) releaseOne(
	ctx context.Context,
	eventType, tenantID, wsID, taskID string,
	companyID int64,
	terminalKind, stopReason string,
	row *cloudconfig.ConfigRow,
) (releaseKind, error) {
	instanceID := strings.TrimSpace(row.InstanceID)
	serverURL := strings.TrimSpace(row.ServerURL)
	commentID := strings.TrimSpace(row.CommentID)
	launchRequestID := strings.TrimSpace(row.LaunchRequestID)
	mig := h.migrator()

	if instanceID != "" {
		foreign, ferr := mig.ListBusyForeign(ctx, companyID, wsID, instanceID, taskID)
		if ferr != nil {
			log.Printf("[task_status_changed] list busy foreign failed task_id=%s comment_id=%s: %v", taskID, commentID, ferr)
			tracelog.LogEventConsume(ctx, "list_busy_foreign_failed", eventType, map[string]any{
				"task_id": taskID, "comment_id": commentID, "error": ferr.Error(),
			})
			return releaseNone, ferr
		}
		for _, f := range foreign {
			if err := mig.MigrateOff(ctx, tenantID, wsID, f.TaskID, instanceID); err != nil {
				log.Printf("[task_status_changed] migrate foreign failed owner=%s from=%s: %v", f.TaskID, instanceID, err)
				tracelog.LogEventConsume(ctx, "migrate_foreign_failed", eventType, map[string]any{
					"task_id": taskID, "owner_task_id": f.TaskID, "error": err.Error(),
				})
				return releaseNone, err
			}
			if strings.TrimSpace(f.ServerURL) != "" {
				if err := h.stopper().StopContainerOnly(ctx, f.TaskID, stopReason+"_migrate_off"); err != nil {
					log.Printf("[task_status_changed] stop foreign container failed owner=%s: %v", f.TaskID, err)
					tracelog.LogEventConsume(ctx, "stop_foreign_failed", eventType, map[string]any{
						"task_id": taskID, "owner_task_id": f.TaskID, "error": err.Error(),
					})
					return releaseNone, err
				}
			}
			tracelog.LogEventConsume(ctx, "migrated_foreign_container", eventType, map[string]any{
				"task_id": taskID, "owner_task_id": f.TaskID, "from_instance_id": instanceID,
			})
		}
		if err := mig.ClearIdleSiblings(ctx, tenantID, wsID, instanceID, taskID); err != nil {
			log.Printf("[task_status_changed] clear idle siblings failed task_id=%s: %v", taskID, err)
			tracelog.LogEventConsume(ctx, "clear_idle_siblings_failed", eventType, map[string]any{
				"task_id": taskID, "error": err.Error(),
			})
			return releaseNone, err
		}
	}

	if serverURL != "" {
		notifyErr := h.notifier().NotifyShutdown(ctx, serverURL, terminalKind, taskID, stopReason)
		if notifyErr == nil {
			if h.Publisher == nil {
				return releaseNone, fmt.Errorf("event publisher not configured")
			}
			awaitData := map[string]interface{}{
				"task_id":       taskID,
				"comment_id":    commentID,
				"tenant_id":     tenantID,
				"company_id":    companyID,
				"workspace_id":  wsID,
				"terminal_kind": terminalKind,
				"stop_reason":   stopReason,
				"instance_id":   instanceID,
				"server_url":    serverURL,
				"attempt":       1,
				"event_id":      fmt.Sprintf("graceful-await-%s-1", taskID),
			}
			key := stopPublishKey(taskID, commentID)
			if err := h.Publisher.PublishEvent(ctx, GracefulShutdownAwaitEvent, awaitData, key); err != nil {
				log.Printf("[task_status_changed] publish graceful await failed task_id=%s: %v", taskID, err)
				tracelog.LogEventConsume(ctx, "publish_graceful_await_failed", eventType, map[string]any{
					"task_id": taskID, "comment_id": commentID, "error": err.Error(),
				})
				return releaseNone, err
			}
			tracelog.LogEventConsume(ctx, "graceful_shutdown_notified", eventType, map[string]any{
				"task_id": taskID, "comment_id": commentID, "terminal_kind": terminalKind, "server_url": serverURL,
			})
			return releaseGraceful, nil
		}
		log.Printf("[task_status_changed] graceful notify failed task_id=%s comment_id=%s: %v (fallback hard-release)", taskID, commentID, notifyErr)
		tracelog.LogEventConsume(ctx, "graceful_notify_failed_hard_release", eventType, map[string]any{
			"task_id": taskID, "comment_id": commentID, "error": notifyErr.Error(),
		})
		if err := h.stopper().StopLocal(ctx, tenantID, wsID, taskID, serverURL, stopReason); err != nil {
			log.Printf("[task_status_changed] local stop failed task_id=%s: %v", taskID, err)
			tracelog.LogEventConsume(ctx, "local_stop_failed", eventType, map[string]any{
				"task_id": taskID, "comment_id": commentID, "error": err.Error(),
			})
			return releaseNone, err
		}
		tracelog.LogEventConsume(ctx, "local_stop_ok", eventType, map[string]any{
			"task_id": taskID, "comment_id": commentID, "terminal_kind": terminalKind, "stop_reason": stopReason,
		})
	}

	if instanceID != "" {
		if h.Publisher == nil {
			return releaseNone, fmt.Errorf("event publisher not configured")
		}
		eventData := buildStopVmEventData(tenantID, wsID, taskID, companyID, row, stopReason)
		key := stopPublishKey(taskID, commentID)
		if err := h.Publisher.PublishEvent(ctx, "CLOUD_SERVER_STOPPED", eventData, key); err != nil {
			log.Printf("[task_status_changed] publish CLOUD_SERVER_STOPPED failed task_id=%s: %v", taskID, err)
			tracelog.LogEventConsume(ctx, "publish_stop_failed", eventType, map[string]any{
				"task_id": taskID, "comment_id": commentID, "error": err.Error(),
			})
			return releaseNone, err
		}
		tracelog.LogEventConsume(ctx, "published_cloud_server_stopped", eventType, map[string]any{
			"task_id": taskID, "comment_id": commentID, "terminal_kind": terminalKind, "instance_id": instanceID,
		})
	}

	// OPT-20260817-027：RunInstances 已发出但回包前（仅有 launch_request_id，尚无
	// instance_id）任务进入终态 → 标记该评论 CSC Released，防晚到的 VM 成为孤儿。
	if instanceID == "" && serverURL == "" && launchRequestID != "" {
		if err := markCommentTerminalReleasedFn(tenantID, wsID, taskID, commentID); err != nil {
			log.Printf("[task_status_changed] mark comment terminal released failed task_id=%s comment_id=%s: %v", taskID, commentID, err)
			tracelog.LogEventConsume(ctx, "mark_comment_terminal_released_failed", eventType, map[string]any{
				"task_id": taskID, "comment_id": commentID, "error": err.Error(),
			})
			return releaseNone, err
		}
		tracelog.LogEventConsume(ctx, "mark_comment_terminal_released", eventType, map[string]any{
			"task_id": taskID, "comment_id": commentID, "launch_request_id": launchRequestID,
		})
		return releaseHard, nil
	}
	return releaseHard, nil
}
