package broker_test

import (
	"context"
	"testing"
	"time"

	"taskEvents/broker"
	"taskEvents/domain"
)

// QS-01: subscribe must not tight-loop when Redis is unreachable.
func TestRedisSubscribeBacksOffOnConnectionError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	conn := broker.NewConnectionState(true)
	rb := broker.NewRedisStreamBrokerAt("127.0.0.1", 1, 0, "domain-events:test", "test-group", "0", conn)
	defer func() { _ = rb.Close() }()

	ch, err := rb.Subscribe(ctx, []string{"USER_CREATED"})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	for range ch {
	}
	elapsed := time.Since(start)

	policy := domain.DefaultRuntimeReconnectPolicy()
	if elapsed < policy.Initial-time.Millisecond*20 {
		t.Fatalf("subscribe finished in %v, expected at least one backoff wait (~%v)", elapsed, policy.Initial)
	}
	if conn.Connected() {
		t.Fatal("expected mq disconnected during subscribe errors")
	}
}
