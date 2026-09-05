package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/cloudserverstarted"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("cloud_server_started")
	if err != nil {
		tracelog.Fatal("cloud_server_started", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("cloud_server_started", "1_process_server_start", &cloudserverstarted.Handler{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
