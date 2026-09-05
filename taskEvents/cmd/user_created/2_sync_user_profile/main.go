package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/syncprofile"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, _, err := config.LoadEvent("user_created")
	if err != nil {
		tracelog.Fatal("sync_profile", "config", err)
	}
	repo := saas.New(appCfg.InternalSecret)
	defer repo.Close()
	// Does not need a Publisher — this intent only writes to DB, doesn't emit events.
	h := &syncprofile.Handler{Repo: repo}
	eventbin.RunIntent("user_created", "2_sync_user_profile", h, consumer.IdempotencyKeyFromEnvelope)
}
