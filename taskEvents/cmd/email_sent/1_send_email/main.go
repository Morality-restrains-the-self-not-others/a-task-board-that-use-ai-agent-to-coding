package main

import (
	"tracelog"

	"taskEvents/eventbin"
	"taskEvents/internal/handlers/filter"
	"taskEvents/notifications"
	notifcfg "taskEvents/notifications/cfg"
	"taskEvents/config"
)

func main() {
	appCfg, root, err := config.Load()
	if err != nil {
		tracelog.Fatal("email_sent", "config", err)
	}
	settings, err := notifcfg.LoadSettings(root)
	if err != nil {
		tracelog.Fatal("email_sent", "settings", err)
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		tracelog.Fatal("email_sent", "delivery", err)
	}
	_ = appCfg
	eventbin.RunIntent("email_sent", "1_send_email", &filter.SingleEvent{EventType: "EMAIL_SENT", Inner: delivery}, notifications.IdempotencyKeyForEnvelope)
}
