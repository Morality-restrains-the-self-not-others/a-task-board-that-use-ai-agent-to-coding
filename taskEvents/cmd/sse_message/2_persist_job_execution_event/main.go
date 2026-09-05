package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/jobstreampersist"
	"taskEvents/internal/repository/saas"
)

func main() {
	h := &jobstreampersist.Handler{Repo: saas.New("")}
	eventbin.RunIntent("sse_message", "2_persist_job_execution_event", h, consumer.IdempotencyKeyFromEnvelope)
}
