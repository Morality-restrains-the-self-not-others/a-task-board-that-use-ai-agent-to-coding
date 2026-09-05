package publish

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"taskEvents/broker"
	"tracelog"
)

// RedisStreamPublisher writes domain events to the Redis stream (Django send_event equivalent).
type RedisStreamPublisher struct {
	client *redis.Client
	stream string
}

func NewRedisStreamPublisher(host string, port, db int, stream string) *RedisStreamPublisher {
	if port == 0 {
		port = 6379
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if stream == "" {
		stream = "domain-events:all"
	}
	return &RedisStreamPublisher{
		client: redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), DB: db}),
		stream: stream,
	}
}

func (p *RedisStreamPublisher) Close() error {
	if p.client == nil {
		return nil
	}
	return p.client.Close()
}

// PublishEvent XADDs one domain event.
func (p *RedisStreamPublisher) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	data = tracelog.EnsureTraceInData(ctx, data)
	return broker.PublishEnvelope(ctx, p.client, p.stream, eventType, data, key)
}
