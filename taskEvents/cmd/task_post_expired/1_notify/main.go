package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskpostexpired"
)

func main() {
	_, _, _, err := config.LoadIntent("task_post_expired", "1_notify")
	if err != nil {
		tracelog.Fatal(config.BinaryName("task_post_expired", "1_notify"), "config", err)
	}
	eventbin.RunIntent(
		"task_post_expired",
		"1_notify",
		&taskpostexpired.Handler{},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
