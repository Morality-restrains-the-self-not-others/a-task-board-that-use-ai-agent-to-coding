package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/taskstatuschanged"
	"taskEvents/internal/publish"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("task_status_changed", "1_release_servers_on_terminal")
	if err != nil {
		tracelog.Fatal("task_status_changed", "config", err)
	}
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent(
		"task_status_changed",
		"1_release_servers_on_terminal",
		&taskstatuschanged.Handler{Publisher: pub, Migrator: taskstatuschanged.NewHTTPMigrator()},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
