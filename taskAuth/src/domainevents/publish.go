//go:build integration

package domainevents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// PublishEvent writes one domain event envelope to a Redis stream (Django-compatible).
func PublishEvent(ctx context.Context, client *redis.Client, stream, eventType string, data map[string]interface{}, key string) error {
	if client == nil {
		return fmt.Errorf("redis client required")
	}
	if stream == "" {
		stream = "domain-events:all"
	}
	data = EnsureTraceInData(ctx, data)
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}
	fields := map[string]interface{}{"payload": string(payload)}
	if key != "" {
		fields["key"] = key
	}
	return client.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: fields}).Err()
}
