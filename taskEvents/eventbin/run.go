package eventbin

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/domain"
)

// Run starts a single-event consumer binary (v3 compat slug).
func Run(slug string, commands domain.DomainCommandPort, keyFor func(domain.EventEnvelope) domain.IdempotencyKey) {
	def, ok := config.PrimaryIntentForEvent(slug)
	if !ok {
		tracelog.Fatalf("task-events-"+slug, "unknown event slug")
	}
	RunIntent(def.EventSlug, def.IntentSlug, commands, keyFor)
}

// RunIntent starts one v4 intent consumer (cmd/{event}/{intent}/main.go).
func RunIntent(eventSlug, intentSlug string, commands domain.DomainCommandPort, keyFor func(domain.EventEnvelope) domain.IdempotencyKey) {
	serviceName := config.BinaryName(eventSlug, intentSlug)
	if serviceName == "" {
		serviceName = "task-events-" + eventSlug + "-" + intentSlug
	}
	cfg, consumerCfg, _, err := config.LoadIntent(eventSlug, intentSlug)
	if err != nil {
		tracelog.Fatal(serviceName, "config", err)
	}
	consumer.RunWithDelivery(serviceName, cfg, consumerCfg, commands, keyFor)
}
