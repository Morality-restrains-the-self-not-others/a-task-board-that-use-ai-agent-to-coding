package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
)

var eventTopicMap = map[string]string{
	"AI_ASSISTANT_REPLY_COMPLETED": "ai-assistant-reply-completed",
	"CommentExecutionModeChanged":  "comment-execution-mode-changed",
}

func publishDomainEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		return nil
	}
	topic, ok := eventTopicMap[eventType]
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
		log.Printf("[taskAIComment] kafka publish %s failed: %v", eventType, err)
		return err
	}
	return nil
}
