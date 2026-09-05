package config

import "os"

// eventOverlay is optional bin/{event}/{intent}/config.yaml (JSON-compatible subset).
type eventOverlay struct {
	Host    string `json:"host" yaml:"host"`
	Port    int    `json:"port" yaml:"port"`
	GroupID string `json:"groupId" yaml:"groupId"`
}

// LoadEvent loads shared Config plus ConsumerConfig for one event slug (v3 compat → primary intent).
func LoadEvent(slug string) (Config, ConsumerConfig, string, error) {
	def, ok := PrimaryIntentForEvent(slug)
	if !ok {
		return Config{}, ConsumerConfig{}, "", os.ErrNotExist
	}
	return LoadIntent(def.EventSlug, def.IntentSlug)
}

func consumerFromPortConfig(root, slug string) (consumerBlock, bool) {
	primary, ok := PrimaryIntentForEvent(slug)
	if !ok {
		return consumerBlock{}, false
	}
	return intentFromEventYAML(root, slug, primary.IntentSlug)
}
