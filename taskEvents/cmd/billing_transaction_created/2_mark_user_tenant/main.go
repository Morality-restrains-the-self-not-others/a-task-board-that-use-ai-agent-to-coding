package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/billing"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadIntent("billing_transaction_created", "2_mark_user_tenant")
	if err != nil {
		tracelog.Fatal(config.BinaryName("billing_transaction_created", "2_mark_user_tenant"), "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	eventbin.RunIntent("billing_transaction_created", "2_mark_user_tenant", &billing.IsTenantIntent{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
