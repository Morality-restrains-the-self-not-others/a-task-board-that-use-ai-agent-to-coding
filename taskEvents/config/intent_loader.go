package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadIntent loads Config + ConsumerConfig for one event/intent (v4).
func LoadIntent(eventSlug, intentSlug string) (Config, ConsumerConfig, string, error) {
	def, ok := IntentByPath(eventSlug, intentSlug)
	if !ok {
		return Config{}, ConsumerConfig{}, "", os.ErrNotExist
	}
	cfg, root, err := Load()
	if err != nil {
		return Config{}, ConsumerConfig{}, "", err
	}
	consumer := ConsumerConfig{
		Host:    "0.0.0.0",
		Port:    def.Port,
		GroupID: def.GroupID,
		Events:  []string{def.EventType},
	}
	if merged, ok := intentFromPortConfig(root, eventSlug, intentSlug); ok {
		mergeConsumer(&consumer, merged)
	} else if legacy, ok := consumerFromPortConfig(root, eventSlug); ok {
		// v3 flat port_config is per-event; only apply to primary intent to avoid port collisions on fan-out.
		if primary, ok := PrimaryIntentForEvent(eventSlug); ok && primary.IntentSlug == intentSlug {
			mergeConsumer(&consumer, legacy)
		}
	}
	if overlay, ok := loadIntentOverlay(root, eventSlug, intentSlug); ok {
		if overlay.Host != "" {
			consumer.Host = overlay.Host
		}
		if overlay.Port != 0 {
			consumer.Port = overlay.Port
		}
		if overlay.GroupID != "" {
			consumer.GroupID = overlay.GroupID
		}
	}
	return cfg, consumer, root, nil
}

func intentFromPortConfig(root, eventSlug, intentSlug string) (consumerBlock, bool) {
	return intentFromEventYAML(root, eventSlug, intentSlug)
}

func loadIntentOverlay(root, eventSlug, intentSlug string) (eventOverlay, bool) {
	for _, name := range []string{"config.yaml", "config.json"} {
		path := filepath.Join(root, "taskEvents", "bin", eventSlug, intentSlug, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var o eventOverlay
		if err := json.Unmarshal(data, &o); err != nil {
			continue
		}
		return o, true
	}
	return eventOverlay{}, false
}
