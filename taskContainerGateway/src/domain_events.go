package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"tracelog"
)

const (
	relayLifecycleTopic = "relay-lifecycle"
	sseMessageTopic     = "sse-message"
)

var (
	kafkaWriter     *kafka.Writer
	kafkaWriterOnce sync.Once
)

func kafkaBrokers() string {
	brokers := strings.TrimSpace(cfg.KafkaBootstrapServers)
	if brokers == "" {
		brokers = "localhost:9093"
	}
	return brokers
}

func initKafkaWriter() *kafka.Writer {
	kafkaWriterOnce.Do(func() {
		kafkaWriter = &kafka.Writer{
			Addr:         kafka.TCP(kafkaBrokers()),
			Topic:        relayLifecycleTopic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		}
	})
	return kafkaWriter
}

func publishDomainEventToTopic(ctx context.Context, topic, eventType string, data map[string]any, key string) error {
	if strings.TrimSpace(eventType) == "" {
		return nil
	}
	if data == nil {
		data = map[string]any{}
	}
	if tid := traceIDFromContext(ctx); tid != "" {
		if _, ok := data["trace_id"]; !ok {
			data["trace_id"] = tid
		}
	}
	payload, err := json.Marshal(map[string]any{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}
	writer := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBrokers()),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer writer.Close()
	msg := kafka.Message{Value: payload, Time: time.Now()}
	if key != "" {
		msg.Key = []byte(key)
	}
	pubCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return writer.WriteMessages(pubCtx, msg)
}

func publishRelayLifecycleEvent(ctx context.Context, eventType string, data map[string]any, key string) {
	if strings.TrimSpace(eventType) == "" {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	if tid := traceIDFromContext(ctx); tid != "" {
		if _, ok := data["trace_id"]; !ok {
			data["trace_id"] = tid
		}
	}
	payload, err := json.Marshal(map[string]any{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		slog.Warn("relay lifecycle event marshal failed", "event_type", eventType, "error", err)
		return
	}
	writer := initKafkaWriter()
	if writer == nil {
		return
	}
	msg := kafka.Message{Key: []byte(key), Value: payload, Time: time.Now()}
	pubCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := writer.WriteMessages(pubCtx, msg); err != nil {
		slog.Warn("relay lifecycle event publish failed", "event_type", eventType, "error", err)
		tracelog.LogForwardStage(ctx, "relay_lifecycle_publish", map[string]any{
			"event_type": eventType,
			"ok":         false,
			"detail":     err.Error(),
		})
		return
	}
	tracelog.LogForwardStage(ctx, "relay_lifecycle_publish", map[string]any{
		"event_type": eventType,
		"ok":         true,
		"topic":      relayLifecycleTopic,
	})
}

// publishSSEMessage publishes SSE_MESSAGE to Kafka (task-events → task-sse),
// matching taskCloudService payload shape so Gateway no longer hits Django.
func publishSSEMessage(ctx context.Context, taskID string, statusData map[string]any) error {
	tid := strings.TrimSpace(taskID)
	if tid == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if statusData == nil {
		statusData = map[string]any{}
	}
	if tidTrace := traceIDFromContext(ctx); tidTrace != "" {
		if _, ok := statusData["trace_id"]; !ok {
			statusData["trace_id"] = tidTrace
		}
	}
	err := publishDomainEventToTopic(ctx, sseMessageTopic, "SSE_MESSAGE", map[string]any{
		"task_id":     tid,
		"status_data": statusData,
	}, tid)
	if err != nil {
		tracelog.LogForwardStage(ctx, "sse_message_publish", map[string]any{
			"event_type": "SSE_MESSAGE",
			"ok":         false,
			"detail":     err.Error(),
			"task_id":    tid,
		})
		return err
	}
	tracelog.LogForwardStage(ctx, "sse_message_publish", map[string]any{
		"event_type": "SSE_MESSAGE",
		"ok":         true,
		"topic":      sseMessageTopic,
		"task_id":    tid,
	})
	return nil
}
