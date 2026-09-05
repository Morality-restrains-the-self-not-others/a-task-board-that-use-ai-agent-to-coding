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
		tracelog.Fatal("wechat_identity_conflict", "config", err)
	}
	settings, err := notifcfg.LoadSettings(root)
	if err != nil {
		tracelog.Fatal("wechat_identity_conflict", "settings", err)
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		tracelog.Fatal("wechat_identity_conflict", "delivery", err)
	}
	eventbin.RunIntent("wechat_identity_conflict", "1_audit_alert", &filter.SingleEvent{EventType: "WECHAT_IDENTITY_CONFLICT", Inner: delivery}, notifications.IdempotencyKeyForEnvelope)
}
