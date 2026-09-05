package broker

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/v9"

	"taskEvents/config"
	"taskEvents/domain"
)

// Ping checks whether the configured transport endpoint is reachable.
func Ping(ctx context.Context, cfg config.Config) error {
	switch cfg.Transport {
	case domain.TransportRedis:
		port := cfg.RedisPort
		if port == 0 {
			port = 6379
		}
		addr := fmt.Sprintf("%s:%d", cfg.RedisHost, port)
		client := redis.NewClient(&redis.Options{Addr: addr, DB: cfg.RedisDB})
		defer func() { _ = client.Close() }()
		return client.Ping(ctx).Err()
	case domain.TransportKafka:
		target := cfg.BootstrapServers
		if target == "" {
			target = "localhost:9093"
		}
		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", target)
		if err != nil {
			return err
		}
		return conn.Close()
	default:
		return fmt.Errorf("ping unsupported for transport %q", cfg.Transport)
	}
}

// PingWithRetry pings the broker with exponential backoff until success or attempts are exhausted.
func PingWithRetry(ctx context.Context, cfg config.Config) error {
	policy := domain.DefaultStartupPingPolicy()
	backoff := BackoffFromPolicy(policy)
	var lastErr error
	for attempt := 0; attempt < policy.Attempts; attempt++ {
		if err := Ping(ctx, cfg); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < policy.Attempts-1 {
			if err := backoff.Wait(ctx); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("broker ping failed after %d attempts: %w", policy.Attempts, lastErr)
}
