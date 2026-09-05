package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
)

// Template for cmd/{event}/{intent}/main.go — replace EVENT_SLUG, INTENT_SLUG, and handler.
func main() {
	const eventSlug = "EVENT_SLUG"
	const intentSlug = "INTENT_SLUG"
	eventbin.RunIntent(eventSlug, intentSlug, nil, consumer.IdempotencyKeyFromEnvelope)
}
