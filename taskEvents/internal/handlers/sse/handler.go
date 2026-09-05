package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"taskEvents/domain"
	"tracelog"
)

// Handler publishes SSE payloads to Redis (no in-process server_startup_status dict).
type Handler struct {
	RedisHost string
	RedisPort int
	RedisDB   int
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

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "SSE_MESSAGE" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	taskID := fieldString(data, "task_id")
	statusRaw, ok := data["status_data"]
	if !ok || taskID == "" {
		tracelog.LogEventConsume(ctx, "sse_missing_fields", "SSE_MESSAGE", nil)
		return domain.DispatchPermanent, fmt.Errorf("missing task_id or status_data")
	}
	statusData, ok := statusRaw.(map[string]interface{})
	if !ok {
		return domain.DispatchPermanent, fmt.Errorf("status_data must be object")
	}
	traceID := fieldString(data, "trace_id")
	if traceID == "" {
		traceID = fieldString(statusData, "trace_id")
	}
	if _, has := statusData["event_name"]; !has {
		statusData["event_name"] = "server_status_update"
	}
	if traceID == "" {
		traceID = tracelog.TraceIDFromContext(ctx)
	}
	if traceID != "" {
		tracelog.EnsureTraceCorrelationInMap(statusData, traceID)
	}
	payload := map[string]interface{}{
		"task_id":     taskID,
		"status_data": statusData,
	}
	if tid, ok := statusData["trace_id"].(string); ok && strings.TrimSpace(tid) != "" {
		payload["trace_id"] = strings.TrimSpace(tid)
	}
	if otel, ok := statusData["otel_trace_id"].(string); ok && strings.TrimSpace(otel) != "" {
		payload["otel_trace_id"] = strings.TrimSpace(otel)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	channel := "sse:" + taskID
	rdb := h.client()
	defer rdb.Close()
	n, err := rdb.Publish(ctx, channel, body).Result()
	if err != nil {
		tracelog.LogEventConsume(ctx, "sse_redis_error", "SSE_MESSAGE", map[string]any{
			"task_id": taskID,
			"channel": channel,
			"error":   err.Error(),
		})
		return domain.DispatchRetryable, err
	}
	tracelog.LogEventConsume(ctx, "sse_redis_published", "SSE_MESSAGE", map[string]any{
		"task_id":     taskID,
		"channel":     channel,
		"subscribers": n,
		"status":      fieldString(statusData, "status"),
		"message":     truncateSSEMsg(fieldString(statusData, "message"), 200),
	})
	return domain.DispatchSuccess, nil
}

func truncateSSEMsg(msg string, max int) string {
	if max <= 0 || len(msg) <= max {
		return msg
	}
	return msg[:max] + "..."
}

func fieldString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
