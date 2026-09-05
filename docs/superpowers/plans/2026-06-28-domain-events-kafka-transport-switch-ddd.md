# DDD Domain Model: Domain Events Transport Redis→Kafka 切换

> Input:
> - Design: `docs/design/consumer-groups-kafka-transport-switch.md`
> - Value Stream: `docs/superpowers/plans/2026-06-28-domain-events-kafka-transport-switch-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-28-domain-events-kafka-transport-switch-nfr.md`
>
> Output consumer: `/6-plans-实施计划`

## Skip Declaration

**This is a pure configuration change.** No new domain concepts, entities, value objects,
aggregates, repositories, or domain events are introduced. The existing domain model
already fully supports both Redis and Kafka transports through well-defined abstractions.

## Existing Domain Abstractions (Confirmed)

### Go Side (taskEvents)

| File | Abstraction | Role |
|------|------------|------|
| `domain/event_broker_port.go` | `EventBrokerPort` interface | Broker abstraction — `KafkaBroker` and `RedisStreamBroker` both implement it |
| `domain/transport_kind.go` | `TransportKind` enum | `TransportKafka` / `TransportRedis` — used by factory to select broker |
| `broker/factory.go` | `New()` function | Factory — reads `cfg.Transport` and returns correct broker implementation |
| `broker/kafka.go` | `KafkaBroker` struct | Full Kafka consumer group implementation (kafka-go) |
| `broker/redis.go` | `RedisStreamBroker` struct | Full Redis Stream consumer group implementation (go-redis) |
| `domain/backoff_policy.go` | `BackoffPolicy` interface | Retry/backoff — shared by both brokers |
| `broker/connection_state.go` | `ConnectionState` struct | Health probe state — shared by both brokers |

### Python Side (Django)

| File | Abstraction | Role |
|------|------------|------|
| `core/services/abstract.py` | `IEventPublisher` ABC | Publisher abstraction — `KafkaEventPublisher` and `RedisStreamEventPublisher` both implement it |
| `core/services/registry.py` | `get_event_publisher()` | Factory — reads `DOMAIN_EVENTS_TRANSPORT` and returns correct publisher |
| `core/services/kafka_event_publisher.py` | `KafkaEventPublisher` | Full Kafka producer implementation (confluent_kafka) |
| `core/services/redis_stream_event_publisher.py` | `RedisStreamEventPublisher` | Full Redis Stream XADD implementation |
| `core/kafka/producer.py` | `KafkaProducer` | Kafka Producer wrapper with delivery callbacks |
| `core/kafka/config.py` | `KAFKA_TOPICS` dict | Event type → Kafka topic mapping |

## No New Domain Files

The transport switch requires:
1. ✅ `conf/domain-events/config.yaml`: `transport: redis` → `transport: kafka`
2. ✅ `conf/infra/docker-infra/config.yaml`: add `kafka.bootstrapServers`
3. ✅ `tests/test_domain_events_port_config.py`: assert `transport == "kafka"`

No domain model changes needed. All abstractions hold.
