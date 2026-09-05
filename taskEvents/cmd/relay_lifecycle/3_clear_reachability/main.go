package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/relaylifecycle"
)

func main() {
	eventbin.RunIntent("relay_lifecycle", "3_clear_reachability", &relaylifecycle.ClearReachabilityHandler{}, consumer.IdempotencyKeyFromEnvelope)
}
