package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/projectupdated"
)

func main() {
	eventbin.RunIntent("project_updated", "1_process_project_update", &projectupdated.Handler{}, consumer.IdempotencyKeyFromEnvelope)
}
