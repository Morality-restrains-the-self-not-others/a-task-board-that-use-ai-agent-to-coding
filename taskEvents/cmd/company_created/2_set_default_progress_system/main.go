package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("company_created", "2_set_default_progress_system")
	if err != nil {
		tracelog.Fatal(config.BinaryName("company_created", "2_set_default_progress_system"), "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	eventbin.RunIntent("company_created", "2_set_default_progress_system", &companycreated.ProgressIntent{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
