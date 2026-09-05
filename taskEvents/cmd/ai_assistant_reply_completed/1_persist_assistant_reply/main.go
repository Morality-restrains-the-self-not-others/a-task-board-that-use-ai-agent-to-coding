package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/aiassistant"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("ai_assistant_reply_completed")
	if err != nil {
		tracelog.Fatal("ai_assistant_reply_completed", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	eventbin.RunIntent("ai_assistant_reply_completed", "1_persist_assistant_reply", &aiassistant.Handler{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
