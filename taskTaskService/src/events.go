package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"tracelog"
)

var eventTopicMap = map[string]string{
	"TASK_CREATED":                   "task-created",
	"TASK_STATUS_CHANGED":            "task-status-changed",
	"TASK_DELETED":                   "task-deleted",
	"TASK_COMMENT_IMAGE_MENTIONED":   "task-comment-image-mentioned",
	"CommentExecutionModeChanged":    "comment-execution-mode-changed",
	"TASK_POST_RENEWED":              "task-post-renewed",
	"TASK_POST_EXPIRED":              "task-post-expired",
	"TASK_GIT_PULL_REQUEST_RECORDED": "task-git-pull-request-recorded",
	"COMMENT_GIT_OAUTH_GRANTED":      "comment-git-oauth-granted",
	"TASK_REVISION_RECORDED":         "task-revision-recorded",
	"SSE_MESSAGE":                    "sse-message",
}

// publishTaskStatusChangedFn is replaceable in tests (no real Kafka).
var publishTaskStatusChangedFn = publishTaskStatusChanged

var publishTaskCreatedFn = publishTaskCreated

var publishTaskDeletedFn = publishTaskDeleted

// publishTaskPostRenewedFn / publishTaskPostExpiredFn are replaceable in tests.
var publishTaskPostRenewedFn = publishTaskPostRenewed
var publishTaskPostExpiredFn = publishTaskPostExpired

// publishDomainEventFn is replaceable in tests (comment @mention notify-before-publish order).
var publishDomainEventFn = publishDomainEvent

var publishTaskRevisionRecordedFn = publishTaskRevisionRecorded

func publishTaskRevisionRecorded(
	ctx context.Context,
	tenantID, workspaceID, taskID, revisionID string,
	versionNum int,
	actorUserID, changedFields string,
) error {
	data := map[string]interface{}{
		"tenant_id":      tenantID,
		"workspace_id":   workspaceID,
		"task_id":        taskID,
		"revision_id":    revisionID,
		"version_num":    versionNum,
		"actor_user_id":  actorUserID,
		"changed_fields": changedFields,
	}
	return publishDomainEvent(ctx, "TASK_REVISION_RECORDED", data, taskID)
}

func publishDomainEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		return nil
	}
	topic, ok := eventTopicMap[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	data = tracelog.EnsureTraceInData(ctx, data)
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBootstrapServers),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer w.Close()
	msg := kafka.Message{Value: payload}
	if key != "" {
		msg.Key = []byte(key)
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		log.Printf("[taskTaskService] kafka publish %s failed: %v", eventType, err)
		return err
	}
	tracelog.LogEventPublish(ctx, eventType, topic, map[string]any{
		"task_id": strField(data, "task_id"),
	})
	return nil
}

func publishTaskCreated(
	ctx context.Context,
	tenantID, workspaceID, taskID, title, userID, postExpiresAt string,
	workspaceSeq int,
) error {
	data := map[string]interface{}{
		"task_id":       taskID,
		"tenant_id":     tenantID,
		"company_id":    tenantID,
		"workspace_id":  workspaceID,
		"title":         title,
		"user_id":       userID,
		"workspace_seq": workspaceSeq,
	}
	if postExpiresAt != "" {
		data["post_expires_at"] = postExpiresAt
	}
	return publishDomainEvent(ctx, "TASK_CREATED", data, taskID)
}

// publishTaskPostRenewed 帖子续存成功（消耗 1 创建帖次数，存续期延长 12 个月）。
func publishTaskPostRenewed(
	ctx context.Context,
	tenantID, workspaceID, taskID, userID, newExpiresAt string,
) error {
	data := map[string]interface{}{
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"company_id":   tenantID,
		"workspace_id": workspaceID,
		"user_id":      userID,
	}
	if newExpiresAt != "" {
		data["post_expires_at"] = newExpiresAt
	}
	return publishDomainEvent(ctx, "TASK_POST_RENEWED", data, taskID)
}

// publishTaskPostExpired 帖子到期下架（每日到期扫描发布，通知用户续存）。
func publishTaskPostExpired(
	ctx context.Context,
	tenantID, workspaceID, taskID string,
) error {
	data := map[string]interface{}{
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"company_id":   tenantID,
		"workspace_id": workspaceID,
	}
	return publishDomainEvent(ctx, "TASK_POST_EXPIRED", data, taskID)
}

func publishTaskDeleted(
	ctx context.Context,
	tenantID, workspaceID, taskID string,
) error {
	data := map[string]interface{}{
		"task_id":      taskID,
		"tenant_id":    tenantID,
		"company_id":   tenantID,
		"workspace_id": workspaceID,
	}
	return publishDomainEvent(ctx, "TASK_DELETED", data, taskID)
}

func publishTaskStatusChanged(
	ctx context.Context,
	tenantID, workspaceID, taskID string,
	prevColumnID, columnID string,
	prevCompleted, completed bool,
	progressColumnName string,
) error {
	data := map[string]interface{}{
		"task_id":                     taskID,
		"tenant_id":                   tenantID,
		"company_id":                  tenantID,
		"workspace_id":                workspaceID,
		"previous_progress_column_id": prevColumnID,
		"progress_column_id":          columnID,
		"previous_completed":          prevCompleted,
		"completed":                   completed,
	}
	if strings.TrimSpace(progressColumnName) != "" {
		data["progress_column_name"] = progressColumnName
	}
	return publishDomainEvent(ctx, "TASK_STATUS_CHANGED", data, taskID)
}
