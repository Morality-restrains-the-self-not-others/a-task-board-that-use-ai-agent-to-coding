package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/billingordercomment"
)

func main() {
	cfg, _, _, err := config.LoadIntent("billing_order_comment_created", "1_notify_other_side")
	if err != nil {
		tracelog.Fatal(config.BinaryName("billing_order_comment_created", "1_notify_other_side"), "config", err)
	}
	h := &billingordercomment.Handler{
		RedisHost: cfg.RedisHost,
		RedisPort: cfg.RedisPort,
		RedisDB:   cfg.RedisDB,
	}
	eventbin.RunIntent(
		"billing_order_comment_created",
		"1_notify_other_side",
		h,
		consumer.IdempotencyKeyFromEnvelope,
	)
}
