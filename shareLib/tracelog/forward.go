package tracelog

import (
	"context"
	"log/slog"
	"strings"
)

// LogForwardStage emits a structured forward_stage line for Loki trace correlation.
func LogForwardStage(ctx context.Context, stage string, attrs map[string]any) {
	args := []any{
		"service", serviceName,
		"forward_stage", stage,
	}
	args = AppendCorrelationLogArgs(args, ctx)
	for k, v := range attrs {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		args = append(args, k, v)
	}
	slog.InfoContext(ctx, "forward_stage", args...)
}

// LogEventPublish emits a structured event_publish line when a domain event is sent to Kafka.
func LogEventPublish(ctx context.Context, eventType, topic string, attrs map[string]any) {
	args := []any{
		"service", serviceName,
		"event_type", eventType,
		"kafka_topic", topic,
	}
	args = AppendCorrelationLogArgs(args, ctx)
	for k, v := range attrs {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		args = append(args, k, v)
	}
	slog.InfoContext(ctx, "event_publish", args...)
}

// LogEventConsume emits a structured event_consume line for Kafka consumer handlers (Loki trace journey).
func LogEventConsume(ctx context.Context, stage, eventType string, attrs map[string]any) {
	args := []any{
		"service", serviceName,
		"event_type", eventType,
		"consume_stage", stage,
	}
	args = AppendCorrelationLogArgs(args, ctx)
	for k, v := range attrs {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		args = append(args, k, v)
	}
	slog.InfoContext(ctx, "event_consume", args...)
}
