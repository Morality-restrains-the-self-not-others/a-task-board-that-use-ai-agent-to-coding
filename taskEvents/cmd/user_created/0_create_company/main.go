package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/usercreated"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("user_created")
	if err != nil {
		tracelog.Fatal("user_created", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	h := &usercreated.Handler{Repo: repo, Publisher: pub}
	eventbin.RunIntent("user_created", "0_create_company", h, consumer.IdempotencyKeyFromEnvelope)
}
