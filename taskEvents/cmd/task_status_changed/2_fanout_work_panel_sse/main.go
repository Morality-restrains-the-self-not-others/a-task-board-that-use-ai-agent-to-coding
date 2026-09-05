package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/workpanelfanout"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("task_status_changed", "2_fanout_work_panel_sse")
	if err != nil {
		tracelog.Fatal("task_status_changed", "config", err)
	}
	eventbin.RunIntent(
		"task_status_changed",
		"2_fanout_work_panel_sse",
		&workpanelfanout.Handler{
			RedisHost: appCfg.RedisHost,
			RedisPort: appCfg.RedisPort,
			RedisDB:   appCfg.RedisDB,
		},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
