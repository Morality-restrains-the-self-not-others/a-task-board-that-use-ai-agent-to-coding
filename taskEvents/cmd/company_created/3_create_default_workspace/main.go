package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("company_created", "3_create_default_workspace")
	if err != nil {
		tracelog.Fatal(config.BinaryName("company_created", "3_create_default_workspace"), "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("company_created", "3_create_default_workspace", &companycreated.WorkspaceIntent{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
