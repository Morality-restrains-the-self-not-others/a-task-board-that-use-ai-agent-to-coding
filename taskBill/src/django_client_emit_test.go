package main

import (
	"context"
	"testing"
)

func TestDjangoEmitEventNoKafkaNoPanic(t *testing.T) {
	// Kafka not configured in test env → publishEvent returns nil without error
	prev := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prev })

	// Should not panic when Kafka is unavailable
	djangoEmitEvent(context.Background(), "PAYPAL_ORDER_CREATED", map[string]interface{}{
		"order_id": "OID-1", "amount": 10,
	})
	// No assertion needed — just verifying no panic

	djangoEmitBillingEvent(context.Background(), 1, map[string]interface{}{
		"trace_id": "test-trace", "amount": 100,
	})
	// No assertion needed — just verifying no panic

	emitPaypalLifecycleAsync(context.Background(), "PAYPAL_ORDER_APPROVED", map[string]interface{}{
		"order_id": "OID-2",
	})
	// No assertion needed — just verifying no panic (async, let it run)
}
