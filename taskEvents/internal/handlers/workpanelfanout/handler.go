package workpanelfanout

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"

	"taskEvents/domain"
	"tracelog"
)

// PublishFunc publishes raw bytes to a Redis channel (injectable for tests).
type PublishFunc func(ctx context.Context, channel string, payload []byte) error

// Handler fans out TASK_STATUS_CHANGED, TASK_CREATED, and TASK_DELETED to workspace-level SSE via Redis.
type Handler struct {
	RedisHost string
	RedisPort int
	RedisDB   int
	Publish   PublishFunc
}

func (h *Handler) client() *redis.Client {
	port := h.RedisPort
	if port == 0 {
		port = 6379
	}
	host := h.RedisHost
	if host == "" {
		host = "127.0.0.1"
	}
	return redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), DB: h.RedisDB})
}

func (h *Handler) publish(ctx context.Context, channel string, payload []byte) error {
	if h.Publish != nil {
		return h.Publish(ctx, channel, payload)
	}
	rdb := h.client()
	defer rdb.Close()
	_, err := rdb.Publish(ctx, channel, payload).Result()
	return err
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	workspaceID := strings.TrimSpace(fieldString(data, "workspace_id"))
	taskID := strings.TrimSpace(fieldString(data, "task_id"))
	if workspaceID == "" || taskID == "" {
		tracelog.LogEventConsume(ctx, "work_panel_fanout_missing_fields", cmd.EventType, map[string]any{
			"workspace_id": workspaceID,
			"task_id":      taskID,
		})
		return domain.DispatchPermanent, fmt.Errorf("missing workspace_id or task_id")
	}

	var eventName string
	var statusData map[string]interface{}
	hubKey := "workspace:" + workspaceID

	switch cmd.EventType {
	case "TASK_STATUS_CHANGED":
		eventName = "task_status_changed"
		statusData = map[string]interface{}{
			"event_name":                  eventName,
			"task_id":                     taskID,
			"tenant_id":                   fieldString(data, "tenant_id"),
			"company_id":                  fieldString(data, "company_id"),
			"workspace_id":                workspaceID,
			"previous_progress_column_id": fieldString(data, "previous_progress_column_id"),
			"progress_column_id":          fieldString(data, "progress_column_id"),
			"previous_completed":          data["previous_completed"],
			"completed":                   data["completed"],
		}
		if name := fieldString(data, "progress_column_name"); name != "" {
			statusData["progress_column_name"] = name
		}
	case "TASK_CREATED":
		eventName = "task_created"
		statusData = map[string]interface{}{
			"event_name":   eventName,
			"task_id":      taskID,
			"tenant_id":    fieldString(data, "tenant_id"),
			"company_id":   fieldString(data, "company_id"),
			"workspace_id": workspaceID,
			"title":        fieldString(data, "title"),
			"user_id":      fieldString(data, "user_id"),
		}
	case "TASK_DELETED":
		eventName = "task_deleted"
		statusData = map[string]interface{}{
			"event_name":   eventName,
			"task_id":      taskID,
			"tenant_id":    fieldString(data, "tenant_id"),
			"company_id":   fieldString(data, "company_id"),
			"workspace_id": workspaceID,
		}
	default:
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}

	if tid := fieldString(data, "trace_id"); tid != "" {
		tracelog.EnsureTraceCorrelationInMap(statusData, tid)
	} else if ctxTid := tracelog.TraceIDFromContext(ctx); ctxTid != "" {
		tracelog.EnsureTraceCorrelationInMap(statusData, ctxTid)
	}
	envelope := map[string]interface{}{
		"task_id":     hubKey,
		"status_data": statusData,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	channel := "sse:" + hubKey
	if err := h.publish(ctx, channel, body); err != nil {
		log.Printf("[work_panel_fanout] redis publish failed channel=%s task_id=%s: %v", channel, taskID, err)
		return domain.DispatchRetryable, err
	}
	tracelog.LogEventConsume(ctx, "work_panel_fanout_published", cmd.EventType, map[string]any{
		"channel":      channel,
		"task_id":      taskID,
		"workspace_id": workspaceID,
	})
	log.Printf("[work_panel_fanout] published channel=%s task_id=%s", channel, taskID)
	return domain.DispatchSuccess, nil
}

func fieldString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
