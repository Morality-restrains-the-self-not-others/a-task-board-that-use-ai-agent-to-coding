package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskgracefulshutdownawait"
	"taskEvents/internal/handlers/taskstatuschanged"
	"taskEvents/internal/publish"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("task_graceful_shutdown_await", "1_hard_release_on_timeout")
	if err != nil {
		tracelog.Fatal("task_graceful_shutdown_await", "config", err)
	}
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent(
		"task_graceful_shutdown_await",
		"1_hard_release_on_timeout",
		&taskgracefulshutdownawait.Handler{
			Publisher: pub,
			Migrator:  taskstatuschanged.NewHTTPMigrator(),
			Stopper:   taskstatuschanged.NewDefaultLocalStopper(),
		},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
