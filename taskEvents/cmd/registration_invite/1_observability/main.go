package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/registrationinvite"
)

func main() {
	eventbin.RunIntent("registration_invite", "1_observability", registrationinvite.LocalHandler(), consumer.IdempotencyKeyFromEnvelope)
}
