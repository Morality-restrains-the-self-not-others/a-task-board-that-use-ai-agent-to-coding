// Package integration holds Redis round-trip tests (build tag integration).
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"taskEvents/broker"
	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/domain"
	"taskEvents/idempotency"
	"taskEvents/routing"
)

// RedisAvailable skips the test when Redis is unreachable.
func RedisAvailable(t *testing.T) (host string, port, db int) {
	t.Helper()
	cfg, _, err := config.Load()
	if err != nil {
		t.Skip("config:", err)
	}
	host = cfg.RedisHost
	port = cfg.RedisPort
	db = cfg.RedisDB
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 6379
	}
	if os.Getenv("SKIP_REDIS_INTEGRATION") == "1" {
		t.Skip("SKIP_REDIS_INTEGRATION=1")
	}
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("redis unavailable:", err)
	}
	_ = rdb.Close()
	return host, port, db
}

// DispatchOnce publishes one event and runs LocalDelivery through the same path as consumer.RunWithDelivery.
func DispatchOnce(
	t *testing.T,
	eventType string,
	data map[string]interface{},
	key string,
	slug string,
	commands domain.DomainCommandPort,
	keyFor func(domain.EventEnvelope) domain.IdempotencyKey,
) domain.DispatchOutcome {
	t.Helper()
	host, port, db := RedisAvailable(t)
	cfg, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := config.EventBySlug(slug); !ok {
		t.Fatalf("unknown slug %s", slug)
	}
	stream := cfg.StreamKey
	if stream == "" {
		stream = "domain-events:all"
	}
	group := fmt.Sprintf("integration-%s-%d", slug, time.Now().UnixNano())
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), DB: db})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rb := broker.NewRedisStreamBrokerAt(host, port, db, stream, group, "$", broker.NewConnectionState(true))
	defer rb.Close()
	ch, err := rb.Subscribe(ctx, []string{eventType})
	if err != nil {
		t.Fatal(err)
	}

	if err := broker.PublishEnvelope(ctx, rdb, stream, eventType, data, key); err != nil {
		t.Fatal(err)
	}

	serviceName := slug
	router := routing.DispatchRouter(serviceName, []string{eventType})
	dispatch := domain.IdempotentDispatchService{
		Commands:    commands,
		Idempotency: idempotency.NewMemoryStore(),
		KeyFor:      keyFor,
	}
	if keyFor == nil {
		dispatch.KeyFor = consumer.IdempotencyKeyFromEnvelope
	}

	var msg domain.BrokerMessage
	select {
	case m, ok := <-ch:
		if !ok {
			t.Fatal("subscribe closed")
		}
		msg = m
	case <-ctx.Done():
		t.Fatal("timeout waiting for redis message")
	}

	cmd, ok := router.CommandFor(msg.Envelope)
	if !ok {
		t.Fatalf("no route for %s", msg.Envelope.EventType)
	}
	out, err := dispatch.Handle(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if err := rb.Ack(ctx, msg); err != nil {
		t.Fatal(err)
	}
	return out
}
