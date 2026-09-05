package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/containermigrateawaitready"
	"taskEvents/internal/publish"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("container_migrate_await_ready")
	if err != nil {
		tracelog.Fatal("container_migrate_await_ready", "config", err)
	}
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	poll := containermigrateawaitready.LoadPollConfigFromEnv()
	eventbin.RunIntent(
		"container_migrate_await_ready",
		"1_await_running_heartbeat",
		&containermigrateawaitready.Handler{
			Publisher:   pub,
			Starter:     containermigrateawaitready.NewDefaultContainerStarter(),
			MaxAttempts: poll.MaxAttempts,
			RetryDelay:  poll.RetryDelay,
		},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
