package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskrevisionrecorded"
)

func main() {
	eventbin.RunIntent("task_revision_recorded", "1_observe", taskrevisionrecorded.LocalHandler(), consumer.IdempotencyKeyFromEnvelope)
}
