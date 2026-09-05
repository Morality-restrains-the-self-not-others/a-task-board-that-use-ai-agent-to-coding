package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/filter"
	"taskEvents/notifications"
	notifcfg "taskEvents/notifications/cfg"
)

func main() {
	_, root, err := config.Load()
	if err != nil {
		tracelog.Fatal("invitation_created", "config", err)
	}
	settings, err := notifcfg.LoadSettings(root)
	if err != nil {
		tracelog.Fatal("invitation_created", "settings", err)
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		tracelog.Fatal("invitation_created", "delivery", err)
	}
	eventbin.RunIntent("invitation_created", "1_send_invitation_email", &filter.SingleEvent{EventType: "INVITATION_CREATED", Inner: delivery}, notifications.IdempotencyKeyForEnvelope)
}
