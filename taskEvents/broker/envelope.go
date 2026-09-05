package broker

import (
	"encoding/json"
	"fmt"

	"taskEvents/domain"
)

type wireEnvelope struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// ParseEnvelope decodes Kafka/Redis message bytes into domain.EventEnvelope.
func ParseEnvelope(value []byte, key []byte) (domain.EventEnvelope, error) {
	var wire wireEnvelope
	if err := json.Unmarshal(value, &wire); err != nil {
		return domain.EventEnvelope{}, err
	}
	if wire.EventType == "" {
		return domain.EventEnvelope{}, fmt.Errorf("missing event_type")
	}
	k := ""
	if len(key) > 0 {
		k = string(key)
	}
	return domain.EventEnvelope{
		EventType: wire.EventType,
		Data:      wire.Data,
		Key:       k,
	}, nil
}
