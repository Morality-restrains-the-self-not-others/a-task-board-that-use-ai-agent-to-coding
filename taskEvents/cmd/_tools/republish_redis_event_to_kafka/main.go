// One-off helper: republish the latest CLOUD_SERVER_STARTED from Redis stream to Kafka.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"taskEvents/config"
)

func main() {
	ctx := context.Background()
	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort), DB: cfg.RedisDB})
	defer rdb.Close()

	entries, err := rdb.XRevRangeN(ctx, cfg.StreamKey, "+", "-", 20).Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "redis: %v\n", err)
		os.Exit(1)
	}
	var target string
	for _, e := range entries {
		raw, _ := e.Values["payload"].(string)
		if strings.Contains(raw, `"event_type":"CLOUD_SERVER_STARTED"`) {
			target = raw
			break
		}
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "no CLOUD_SERVER_STARTED in redis stream")
		os.Exit(1)
	}
	var wire struct {
		EventType string                 `json:"event_type"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(target), &wire); err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}
	payload, _ := json.Marshal(map[string]interface{}{"event_type": wire.EventType, "data": wire.Data})
	key := ""
	if wire.Data != nil {
		if tid, ok := wire.Data["task_id"].(string); ok {
			key = tid
		}
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.BootstrapServers),
		Topic:        "cloud-server-started",
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
	defer w.Close()
	msg := kafka.Message{Value: payload}
	if key != "" {
		msg.Key = []byte(key)
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		fmt.Fprintf(os.Stderr, "kafka: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("republished CLOUD_SERVER_STARTED task=%s to kafka topic cloud-server-started\n", key)
}
