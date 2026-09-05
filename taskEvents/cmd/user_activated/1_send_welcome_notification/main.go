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
		tracelog.Fatal("user_activated", "config", err)
	}
	settings, err := notifcfg.LoadSettings(root)
	if err != nil {
		tracelog.Fatal("user_activated", "settings", err)
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		tracelog.Fatal("user_activated", "delivery", err)
	}
	eventbin.RunIntent("user_activated", "1_send_welcome_notification", &filter.SingleEvent{EventType: "USER_ACTIVATED", Inner: delivery}, notifications.IdempotencyKeyForEnvelope)
}
