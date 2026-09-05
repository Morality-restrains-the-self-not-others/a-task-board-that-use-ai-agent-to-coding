//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"taskEvents/domain"
	"taskEvents/integration"
	"taskEvents/internal/handlers/sse"
)

func TestSSEMessageRedisPublishRoundTrip(t *testing.T) {
	host, port, db := integration.RedisAvailable(t)
	h := &sse.Handler{RedisHost: host, RedisPort: port, RedisDB: db}
	taskID := fmt.Sprintf("sse-int-%d", time.Now().UnixNano())
	status := map[string]interface{}{"status": "ok", "event_name": "server_status_update"}

	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), DB: db})
	defer rdb.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sub := rdb.Subscribe(ctx, "sse:"+taskID)
	defer sub.Close()
	// Allow subscription to register before handler publishes.
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}

	out := integration.DispatchOnce(
		t,
		"SSE_MESSAGE",
		map[string]interface{}{"task_id": taskID, "status_data": status},
		taskID,
		"sse_message",
		h,
		nil,
	)
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Payload == "" {
		t.Fatal("empty sse payload")
	}
}
