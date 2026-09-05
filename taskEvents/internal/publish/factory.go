package publish

import (
	"taskEvents/config"
	"taskEvents/domain"
)

// ClosablePublisher publishes outbound domain events and can release resources on shutdown.
type ClosablePublisher interface {
	EventPublisher
	Close() error
}

// NewFromConfig selects Redis or Kafka publisher based on domain-events transport.
func NewFromConfig(cfg config.Config) ClosablePublisher {
	switch cfg.Transport {
	case domain.TransportKafka:
		return NewKafkaPublisher(cfg.BootstrapServers)
	default:
		return NewRedisStreamPublisher(cfg.RedisHost, cfg.RedisPort, cfg.RedisDB, cfg.StreamKey)
	}
}
