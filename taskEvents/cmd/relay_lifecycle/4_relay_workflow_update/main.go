package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/relaylifecycle"
)

func main() {
	eventbin.RunIntent("relay_lifecycle", "4_relay_workflow_update", &relaylifecycle.WorkflowUpdateHandler{}, consumer.IdempotencyKeyFromEnvelope)
}
