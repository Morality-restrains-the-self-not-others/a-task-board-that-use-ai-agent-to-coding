package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/billing"
)

func main() {
	cfg, _, _, err := config.LoadEvent("billing_transaction_created")
	if err != nil {
		tracelog.Fatal("billing_transaction_created", "config", err)
	}
	h := &billing.Handler{
		RedisHost: cfg.RedisHost,
		RedisPort: cfg.RedisPort,
		RedisDB:   cfg.RedisDB,
	}
	eventbin.RunIntent("billing_transaction_created", "1_process_billing_transaction", h, consumer.IdempotencyKeyFromEnvelope)
}
