package domain

// TransportKind selects the message broker implementation.
type TransportKind string

const (
	TransportMemory TransportKind = "memory"
	TransportKafka  TransportKind = "kafka"
	TransportRedis  TransportKind = "redis"
)

func (t TransportKind) Valid() bool {
	switch t {
	case TransportMemory, TransportKafka, TransportRedis:
		return true
	default:
		return false
	}
}
