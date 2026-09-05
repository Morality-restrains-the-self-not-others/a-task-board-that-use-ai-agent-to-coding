package publish

import (
	"testing"

	"taskEvents/config"
	"taskEvents/domain"
)

func TestNewFromConfigUsesKafkaWhenTransportKafka(t *testing.T) {
	cfg := config.Config{
		Transport:        domain.TransportKafka,
		BootstrapServers: "localhost:9093",
	}
	pub := NewFromConfig(cfg)
	if _, ok := pub.(*KafkaPublisher); !ok {
		t.Fatalf("expected *KafkaPublisher, got %T", pub)
	}
}

func TestNewFromConfigUsesRedisWhenTransportRedis(t *testing.T) {
	cfg := config.Config{
		Transport: domain.TransportRedis,
		RedisHost: "127.0.0.1",
		RedisPort: 6379,
		StreamKey: "domain-events:all",
	}
	pub := NewFromConfig(cfg)
	if _, ok := pub.(*RedisStreamPublisher); !ok {
		t.Fatalf("expected *RedisStreamPublisher, got %T", pub)
	}
}
