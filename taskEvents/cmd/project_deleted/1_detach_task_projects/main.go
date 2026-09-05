package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/projectdeleted"
)

func main() {
	h := &projectdeleted.Handler{Client: projectdeleted.NewDefaultDetachClient()}
	eventbin.RunIntent("project_deleted", "1_detach_task_projects", h, consumer.IdempotencyKeyFromEnvelope)
}
