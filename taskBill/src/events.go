package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"tracelog"
)

// billingEventTopics maps internal event type names to Kafka topics.
// Must stay in sync with Django core/kafka/config.py KAFKA_TOPICS for billing events.
var billingEventTopics = map[string]string{
	"BILLING_TRANSACTION_CREATED":          "billing-transaction-created",
	"BILLING_CREDIT_EXPIRED":               "billing-credit-expired",
	"BILLING_REFUND_APPLICATION_SUBMITTED": "billing-refund-application-submitted",
	"BILLING_REFUND_APPLICATION_REJECTED":  "billing-refund-application-rejected",
	"BILLING_REFUND_COMPLETED":             "billing-refund-completed",
	"BILLING_RESOURCE_GRANT_EXPIRED":       "billing-resource-grant-expired",
	"BILLING_ORDER_COMMENT_CREATED":        "billing-order-comment-created",
	"PAYPAL_ORDER_CREATED":                 "paypal-order-created",
	"PAYPAL_ORDER_APPROVED":                "paypal-order-approved",
	"PAYPAL_CAPTURE_COMPLETED":             "paypal-capture-completed",
	"PAYMENT_SUCCEEDED":                    "payment-succeeded",
	"GitlabRegionAccessModeChanged":        "gitlab-region-access-mode-changed",
	"GitlabManualNodeFulfillmentQueued":    "gitlab-manual-node-fulfillment-queued",
	"GitlabRegionInfraMarkedReady":         "gitlab-region-infra-marked-ready",
	"FEEDBACK_LINK_GROUP_CREATED":          "feedback-link-group-created",
	"FEEDBACK_LINK_GROUP_UPDATED":          "feedback-link-group-updated",
	"FEEDBACK_LINK_GROUP_DELETED":          "feedback-link-group-deleted",
}

// feedbackEventPublisher is the Kafka publisher for feedback-link-group events; tests may replace it.
var feedbackEventPublisher = publishEvent

// gitlabRegionEventPublisher is the Kafka publisher for GitLab region infra events; tests may replace it.
var gitlabRegionEventPublisher = publishEvent

// publishEvent sends a domain event directly to Kafka (no Django bridge).
func publishEvent(ctx context.Context, eventType string, payload map[string]interface{}, key string) error {
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		log.Printf("[taskBill] kafka bootstrap servers empty, skip publish %s", eventType)
		return nil
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return nil
	}
	topic, ok := billingEventTopics[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	// Inject trace info if available
	if ctx != nil {
		if tid := tracelog.TraceIDFromContext(ctx); tid != "" {
			if _, exists := payload["trace_id"]; !exists {
				payload["trace_id"] = tid
			}
		}
		payload = tracelog.EnsureTraceInData(ctx, payload)
	}
	body, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       payload,
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
	msg := kafka.Message{Value: body}
	if key != "" {
		msg.Key = []byte(key)
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		log.Printf("[taskBill] kafka publish %s failed: %v", eventType, err)
		return err
	}
	tracelog.LogEventPublish(ctx, eventType, topic, map[string]any{
		"event_type": eventType,
	})
	return nil
}

// publishBillingTransactionCreated sends BILLING_TRANSACTION_CREATED via Kafka.
// Replaces the Django emit-billing-event bridge.
func publishBillingTransactionCreated(ctx context.Context, outboxID int64, payload map[string]interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := publishEvent(ctx, "BILLING_TRANSACTION_CREATED", payload, formatID(outboxID)); err != nil {
		log.Printf("[taskBill] emit billing event outbox=%d err=%v", outboxID, err)
	}
}

// publishBillingEvent sends an arbitrary billing lifecycle event via Kafka.
// eventType must match a key in billingEventTopics or will be slugified.
func publishBillingEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
	if strings.TrimSpace(eventType) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := publishEvent(ctx, eventType, payload, ""); err != nil {
		log.Printf("[taskBill] emit event type=%s err=%v", eventType, err)
	}
}

// publishPaypalLifecycleAsync sends PayPal lifecycle events asynchronously.
func publishPaypalLifecycleAsync(ctx context.Context, eventType string, payload map[string]interface{}) {
	go publishBillingEvent(ctx, eventType, payload)
}
