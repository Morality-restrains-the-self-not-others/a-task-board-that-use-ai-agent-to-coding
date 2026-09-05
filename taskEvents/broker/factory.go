package broker

import (
	"fmt"

	"taskEvents/config"
	"taskEvents/domain"
)

// Handle pairs a broker port with its live connection state for health probes.
type Handle struct {
	Port  domain.EventBrokerPort
	Conn  *ConnectionState
}

// New returns a broker for the configured transport.
func New(cfg config.Config, groupID string, events []string) (domain.EventBrokerPort, error) {
	handle, err := NewHandle(cfg, groupID, events)
	if err != nil {
		return nil, err
	}
	return handle.Port, nil
}

// NewHandle returns a broker and shared connection state for health reporting.
func NewHandle(cfg config.Config, groupID string, events []string) (*Handle, error) {
	conn := NewConnectionState(false)
	switch cfg.Transport {
	case domain.TransportKafka:
		topics := config.TopicsForEvents(events)
		return &Handle{Port: NewKafkaBroker(cfg.BootstrapServers, groupID, topics, conn), Conn: conn}, nil
	case domain.TransportRedis:
		return &Handle{Port: NewRedisStreamBroker(cfg.RedisHost, cfg.RedisPort, cfg.RedisDB, cfg.StreamKey, groupID, conn), Conn: conn}, nil
	case domain.TransportMemory:
		return nil, fmt.Errorf("memory transport has no broker")
	default:
		return nil, fmt.Errorf("unknown transport %q", cfg.Transport)
	}
}
