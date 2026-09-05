package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// Topics aligned with taskEvents EventTopic map / Django KAFKA_TOPICS.
var eventTopicMap = map[string]string{
	"INVITATION_CREATED": "invitation-created",
	"MEMBER_JOINED":      "member-joined",
}

func kafkaBootstrapServers() string {
	if v := strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS")); v != "" {
		return v
	}
	return strings.TrimSpace(cfg.KafkaBootstrapServers)
}

// publishEventHook is a test seam. When non-nil it records the event payload
// and short-circuits Kafka publish. Set only from tests; reset in t.Cleanup.
var publishEventHook func(eventType string, data map[string]interface{})

func publishEvent(eventType string, data map[string]interface{}) {
	payload := map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	}
	b, _ := json.Marshal(payload)
	log.Printf("[taskTenantService] event %s %s", eventType, string(b))
	logInfo("published_"+eventType, "")
	if publishEventHook != nil {
		publishEventHook(eventType, data)
		return
	}

	bootstrap := kafkaBootstrapServers()
	if bootstrap == "" {
		return
	}
	topic, ok := eventTopicMap[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	key := strField(data, "invitation_id")
	if key == "" {
		key = strField(data, "member_id")
	}
	if key == "" {
		key = strField(data, "company_id")
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(bootstrap),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer w.Close()
	msg := kafka.Message{Value: b}
	if key != "" {
		msg.Key = []byte(key)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.WriteMessages(ctx, msg); err != nil {
		log.Printf("[taskTenantService] kafka publish %s topic=%s failed: %v", eventType, topic, err)
		return
	}
	log.Printf("[taskTenantService] kafka published event=%s topic=%s key=%s", eventType, topic, key)
}

// publishMemberJoined emits MEMBER_JOINED for downstream auto git-identity creation.
func publishMemberJoined(memberID, userID, companyID, memberName, role, workspaceID, source string, extra map[string]interface{}) {
	evt := map[string]interface{}{
		"member_id":    memberID,
		"user_id":      userID,
		"company_id":   companyID,
		"member_name":  memberName,
		"role":         role,
		"workspace_id": workspaceID,
		"source":       source,
	}
	for k, v := range extra {
		if _, exists := evt[k]; !exists {
			evt[k] = v
		}
	}
	publishEvent("MEMBER_JOINED", evt)
}
