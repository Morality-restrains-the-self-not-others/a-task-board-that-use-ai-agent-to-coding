package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/workpanelfanout"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("task_deleted", "1_fanout_work_panel_sse")
	if err != nil {
		tracelog.Fatal("task_deleted", "config", err)
	}
	eventbin.RunIntent(
		"task_deleted",
		"1_fanout_work_panel_sse",
		&workpanelfanout.Handler{
			RedisHost: appCfg.RedisHost,
			RedisPort: appCfg.RedisPort,
			RedisDB:   appCfg.RedisDB,
		},
		consumer.IdempotencyKeyFromEnvelope,
	)
}
