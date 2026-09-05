package cloudcommon

import (
	"context"
	"fmt"
	"strings"

	"taskEvents/internal/publish"
	"tracelog"
)

// PublishSSE sends one SSE_MESSAGE domain event with trace_id in status_data and structured log.
func PublishSSE(ctx context.Context, pub publish.EventPublisher, taskID string, statusData map[string]interface{}) error {
	if pub == nil {
		return fmt.Errorf("SSE publisher is nil")
	}
	statusData = ensureTraceInStatusData(ctx, statusData)
	tracelog.LogEventConsume(ctx, "sse_publish", "SSE_MESSAGE", map[string]any{
		"task_id": taskID,
		"status":  strField(statusData, "status"),
		"message": truncateMsg(strField(statusData, "message"), 200),
	})
	return pub.PublishEvent(ctx, "SSE_MESSAGE", sseEventPayload(taskID, statusData), taskID)
}

// PublishSSEBatch sends multiple SSE payloads in order.
func PublishSSEBatch(ctx context.Context, pub publish.EventPublisher, events []map[string]interface{}) error {
	for _, ev := range events {
		taskID, _ := ev["task_id"].(string)
		status, _ := ev["status_data"].(map[string]interface{})
		if taskID == "" || status == nil {
			continue
		}
		if err := PublishSSE(ctx, pub, taskID, status); err != nil {
			return err
		}
	}
	return nil
}

func ensureTraceInStatusData(ctx context.Context, statusData map[string]interface{}) map[string]interface{} {
	if statusData == nil {
		statusData = map[string]interface{}{}
	}
	traceID := ""
	if raw, ok := statusData["trace_id"].(string); ok && strings.TrimSpace(raw) != "" {
		traceID = strings.TrimSpace(raw)
	}
	if traceID == "" {
		traceID = tracelog.TraceIDFromContext(ctx)
	}
	if traceID != "" {
		tracelog.EnsureTraceCorrelationInMap(statusData, traceID)
	}
	return statusData
}

func sseEventPayload(taskID string, statusData map[string]interface{}) map[string]interface{} {
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
	return payload
}

func strField(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func truncateMsg(msg string, max int) string {
	if max <= 0 || len(msg) <= max {
		return msg
	}
	return msg[:max] + "..."
}
