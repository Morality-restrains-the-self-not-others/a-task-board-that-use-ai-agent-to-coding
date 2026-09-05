package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskcommentimagementioned"
	"taskEvents/internal/publish"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("task_comment_image_mentioned")
	if err != nil {
		tracelog.Fatal("task_comment_image_mentioned", "config", err)
	}
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent(
		"task_comment_image_mentioned",
		"1_start_vm_for_at_mention",
		&taskcommentimagementioned.Handler{Publisher: pub},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
