package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/cloudplatformauth"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("cloud_platform_authorization_created")
	if err != nil {
		tracelog.Fatal("cloud_platform_authorization_created", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("cloud_platform_authorization_created", "1_process_cloud_platform_authorization", &cloudplatformauth.Handler{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
