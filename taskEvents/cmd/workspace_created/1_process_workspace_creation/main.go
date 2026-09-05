package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/workspacecreated"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("workspace_created")
	if err != nil {
		tracelog.Fatal("workspace_created", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	eventbin.RunIntent("workspace_created", "1_process_workspace_creation", &workspacecreated.Handler{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
