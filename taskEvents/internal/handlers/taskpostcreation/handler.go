package taskpostcreation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"taskEvents/internal/publish"
	"tracelog"
)

// Handler consumes TASK_CREATED events and generates (confirms) the task post.
// This is the async "task post generation" step — the task record is already
// created by taskTaskService, and this consumer performs post-creation
// processing: validation, audit logging, and notification publishing.
type Handler struct {
	Publisher publish.EventPublisher
}

const TaskPostCreatedEvent = "TASK_POST_CREATED"

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "TASK_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		tracelog.LogEventConsume(ctx, "task_post_creation_bad_payload", cmd.EventType, map[string]any{
			"error": err.Error(),
		})
		return domain.DispatchPermanent, err
	}

	taskID := fieldStr(data, "task_id")
	tenantID := fieldStr(data, "tenant_id")
	workspaceID := fieldStr(data, "workspace_id")
	title := fieldStr(data, "title")
	userID := fieldStr(data, "user_id")
	postExpiresAt := fieldStr(data, "post_expires_at") // v15 存续期：创建帖 12 个月有效期

	if taskID == "" {
		tracelog.LogEventConsume(ctx, "task_post_creation_missing_task_id", cmd.EventType, nil)
		return domain.DispatchPermanent, fmt.Errorf("missing task_id in TASK_CREATED event")
	}

	// Log the task post generation — this is the "生成一个任务帖子" step.
	log.Printf("[task_post_creation] task post generated: task_id=%s tenant=%s workspace=%s title=%s user=%s expires_at=%s",
		taskID, tenantID, workspaceID, title, userID, postExpiresAt)

	tracelog.LogEventConsume(ctx, "task_post_created", cmd.EventType, map[string]any{
		"task_id":        taskID,
		"tenant_id":      tenantID,
		"workspace_id":   workspaceID,
		"title":          title,
		"user_id":        userID,
		"post_expires_at": postExpiresAt,
	})

	// 发送任务帖创建通知事件（站内通知、Webhook 回调等下游消费者可订阅）
	h.emitTaskPostCreatedEvent(ctx, taskID, tenantID, workspaceID, title, userID, postExpiresAt)

	return domain.DispatchSuccess, nil
}

// emitTaskPostCreatedEvent 发布 TASK_POST_CREATED 事件，供通知、Webhook
// 等下游消费者订阅。通知发送失败不影响主流程成功。
func (h *Handler) emitTaskPostCreatedEvent(ctx context.Context, taskID, tenantID, workspaceID, title, userID, postExpiresAt string) {
	if h.Publisher == nil {
		log.Printf("[task_post_creation] publisher not configured, skipping TASK_POST_CREATED event for task_id=%s", taskID)
		return
	}

	notifData := map[string]interface{}{
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"title":        title,
		"user_id":      userID,
	}
	if postExpiresAt != "" {
		notifData["post_expires_at"] = postExpiresAt
	}

	if err := h.Publisher.PublishEvent(ctx, TaskPostCreatedEvent, notifData, taskID); err != nil {
		// 通知发送是 best-effort：失败不影响任务帖创建确认
		log.Printf("[task_post_creation] TASK_POST_CREATED event publish failed for task_id=%s: %v", taskID, err)
		tracelog.LogEventConsume(ctx, "task_post_created_notify_failed", TaskPostCreatedEvent, map[string]any{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return
	}

	tracelog.LogEventConsume(ctx, "task_post_created_notify_sent", TaskPostCreatedEvent, map[string]any{
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
	})
}

func fieldStr(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
