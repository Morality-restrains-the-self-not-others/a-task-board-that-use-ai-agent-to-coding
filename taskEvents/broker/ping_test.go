package broker_test

import (
	"context"
	"testing"
	"time"

	"taskEvents/broker"
	"taskEvents/config"
	"taskEvents/domain"
)

func TestPingWithRetryFailsOnDeadPort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := config.Config{
		Transport: domain.TransportRedis,
		RedisHost: "127.0.0.1",
		RedisPort: 1,
	}

	err := broker.PingWithRetry(ctx, cfg)
	if err == nil {
		t.Fatal("expected ping failure on closed port")
	}
}

func TestPingWithRetrySucceedsWhenRedisAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("requires local redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.Config{
		Transport: domain.TransportRedis,
		RedisHost: "127.0.0.1",
		RedisPort: 6379,
	}
	if err := broker.PingWithRetry(ctx, cfg); err != nil {
		t.Skipf("redis not available: %v", err)
	}
}
