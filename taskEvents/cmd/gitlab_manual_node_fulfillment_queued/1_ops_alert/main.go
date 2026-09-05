package main

import (
	"log"

	"tracelog"

	"taskEvents/broker"
	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/gitlabmanualnode"
	notifcfg "taskEvents/notifications/cfg"
	"taskEvents/notifications/smtp"
)

func main() {
	const eventSlug = "gitlab_manual_node_fulfillment_queued"
	const intentSlug = "1_ops_alert"
	serviceName := config.BinaryName(eventSlug, intentSlug)
	if serviceName == "" {
		serviceName = "task-events-" + eventSlug + "-" + intentSlug
	}

	_, root, err := config.Load()
	if err != nil {
		tracelog.Fatal(serviceName, "config", err)
	}
	settings, err := notifcfg.LoadSettings(root)
	if err != nil {
		tracelog.Fatal(serviceName, "settings", err)
	}
	h := &gitlabmanualnode.Handler{
		From: settings.Email.DefaultFrom,
		To:   broker.ResolveDLTAlertEmail(),
	}
	if settings.Email.Host != "" {
		h.Sender = smtp.NewSender(settings.Email)
	} else {
		// SMTP 未配置：仍消费并去重，只保留审计日志，避免事件永久卡在重试。
		log.Printf("[gitlab-manual-node] ops alert smtp disabled (empty host): audit-log only")
	}
	eventbin.RunIntent(eventSlug, intentSlug, h, consumer.IdempotencyKeyFromEnvelope)
}
