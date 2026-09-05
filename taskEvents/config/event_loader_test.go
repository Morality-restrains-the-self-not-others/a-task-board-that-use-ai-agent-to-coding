package config

import (
	"testing"
)

func TestLoadEventBilling(t *testing.T) {
	_, consumer, _, err := LoadEvent("billing_transaction_created")
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	if len(consumer.Events) != 1 || consumer.Events[0] != "BILLING_TRANSACTION_CREATED" {
		t.Fatalf("events: %v", consumer.Events)
	}
	if consumer.Port != 18020 {
		t.Fatalf("port: %d", consumer.Port)
	}
	if consumer.GroupID != "task-events-billing-transaction-created-1-process-billing-transaction" {
		t.Fatalf("group: %q", consumer.GroupID)
	}
}

func TestEventBySlug(t *testing.T) {
	def, ok := EventBySlug("sse_message")
	if !ok {
		t.Fatal("missing sse_message")
	}
	if def.EventType != "SSE_MESSAGE" {
		t.Fatalf("event type %q", def.EventType)
	}
}
