package publish

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/segmentio/kafka-go"

	"taskEvents/config"
	"tracelog"
)

// KafkaPublisher publishes domain events to Kafka topics (aligned with Django send_event).
type KafkaPublisher struct {
	brokers string
}

func NewKafkaPublisher(bootstrapServers string) *KafkaPublisher {
	return &KafkaPublisher{brokers: bootstrapServers}
}

func (p *KafkaPublisher) Close() error {
	return nil
}

func (p *KafkaPublisher) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	data = tracelog.EnsureTraceInData(ctx, data)
	topic, ok := config.EventTopic(eventType)
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}
	brokers := strings.TrimSpace(p.brokers)
	if brokers == "" {
		brokers = "localhost:9093"
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers),
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
		return err
	}
	tracelog.LogEventPublish(ctx, eventType, topic, map[string]any{
		"task_id": strField(data, "task_id"),
		"event_id": strField(data, "event_id"),
	})
	return nil
}

func strField(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
