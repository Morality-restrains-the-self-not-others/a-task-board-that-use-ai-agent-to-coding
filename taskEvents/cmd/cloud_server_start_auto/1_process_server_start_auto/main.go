package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/cloudserverstartauto"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("cloud_server_start_auto")
	if err != nil {
		tracelog.Fatal("cloud_server_start_auto", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("cloud_server_start_auto", "1_process_server_start_auto", &cloudserverstartauto.Handler{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
