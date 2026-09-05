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
	appCfg, _, _, err := config.LoadIntent("company_created", "5_init_tenant_feature_params")
	if err != nil {
		tracelog.Fatal(config.BinaryName("company_created", "5_init_tenant_feature_params"), "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	eventbin.RunIntent("company_created", "5_init_tenant_feature_params", &companycreated.FeatureParamsIntent{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
