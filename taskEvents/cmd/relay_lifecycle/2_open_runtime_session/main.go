package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/relaylifecycle"
)

func main() {
	eventbin.RunIntent("relay_lifecycle", "2_open_runtime_session", &relaylifecycle.OpenRuntimeSessionHandler{}, consumer.IdempotencyKeyFromEnvelope)
}
