package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/cloudserverstopped"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("cloud_server_stopped")
	if err != nil {
		tracelog.Fatal("cloud_server_stopped", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("cloud_server_stopped", "1_process_server_stop", &cloudserverstopped.Handler{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
