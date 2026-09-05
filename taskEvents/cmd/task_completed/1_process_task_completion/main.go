package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskcompleted"
)

func main() {
	eventbin.RunIntent("task_completed", "1_process_task_completion", &taskcompleted.Handler{}, consumer.IdempotencyKeyFromEnvelope)
}
