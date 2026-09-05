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
	appCfg, _, _, err := config.LoadIntent("company_created", "6_grant_initial_resources")
	if err != nil {
		tracelog.Fatal(config.BinaryName("company_created", "6_grant_initial_resources"), "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("company_created", "6_grant_initial_resources", &companycreated.ResourceGrantIntent{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
