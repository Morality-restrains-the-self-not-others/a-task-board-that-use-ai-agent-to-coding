package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/relaylifecycle"
)

func main() {
	eventbin.RunIntent("relay_lifecycle", "1_append_token_audit", relaylifecycle.LocalHandler(), consumer.IdempotencyKeyFromEnvelope)
}
