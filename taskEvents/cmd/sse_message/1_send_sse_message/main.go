package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/sse"
)

func main() {
	cfg, _, _, err := config.LoadEvent("sse_message")
	if err != nil {
		tracelog.Fatal("sse_message", "config", err)
	}
	h := &sse.Handler{
		RedisHost: cfg.RedisHost,
		RedisPort: cfg.RedisPort,
		RedisDB:   cfg.RedisDB,
	}
	eventbin.RunIntent("sse_message", "1_send_sse_message", h, consumer.IdempotencyKeyFromEnvelope)
}
