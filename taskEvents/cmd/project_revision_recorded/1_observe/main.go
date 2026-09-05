package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/projectrevisionrecorded"
)

func main() {
	eventbin.RunIntent("project_revision_recorded", "1_observe", projectrevisionrecorded.LocalHandler(), consumer.IdempotencyKeyFromEnvelope)
}
