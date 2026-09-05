package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskpostcreation"
	"taskEvents/internal/publish"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("task_created", "2_create_task_post")
	if err != nil {
		tracelog.Fatal(config.BinaryName("task_created", "2_create_task_post"), "config", err)
	}
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent(
		"task_created",
		"2_create_task_post",
		&taskpostcreation.Handler{Publisher: pub},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
