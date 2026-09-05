# NFR Clarification: Domain Events Transport Redis→Kafka 切换

> Input:
> - Design: `docs/design/consumer-groups-kafka-transport-switch.md`
> - Value Stream: `docs/superpowers/plans/2026-06-28-domain-events-kafka-transport-switch-value-stream.md`
>
> Output consumers: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR Overview

| Category | Level | One-line Quantification |
|----------|-------|------------------------|
| Fault Tolerance | L2 | Standard — existing backoff retry + reconnect applies |
| Observability | L2 | Standard — consumer groups visible in Kafka UI + existing health APIs |
| Consistency | L2 | Standard — same envelope format, at-least-once + idempotent consumers |
| Performance | L1 | Basic — same messaging throughput as Redis (no new perf targets) |
| Security | L0 | N/A — no credential changes; internal Kafka on localhost |
| Availability | L1 | Basic — Kafka single-node, no HA guarantee (unchanged) |
| Scalability | L0 | N/A — single-node dev environment |
| Compliance | L0 | N/A — no data locality / regulatory change |
| Maintainability | L1 | Basic — single config line change, trivial rollback |

## Per-Increment NFR Analysis

### Increment 1: Transport Switch (Thin Slice)

#### Fault Tolerance — L2 Standard
- **Existing**: `taskEvents/broker/backoff.go` — exponential backoff (initial 1s, max 30s) on broker fetch failure; `KafkaBroker` sets `connState.SetConnected(false)` on error → health API reflects degraded
- **Change**: Kafka broker connection failure path now exercised (previously Redis was the active path). Both paths already have identical retry/backoff logic.
- **No new code needed**.

#### Observability — L2 Standard
- **Goal**: Consumer groups visible in Kafka UI (`:18080/ui/clusters/local/consumer-groups`)
- **Existing**: health APIs at `:18020-18037/api/health/ready` return `ok|degraded` based on broker connection state
- **Change**: Health check ping changes from Redis to Kafka; `ConnectionState` driven by `KafkaBroker.FetchMessage` success/failure

#### Consistency — L2 Standard
- **Message format**: Same envelope `{event_type, data}` across both transports
- **Ordering**: Kafka preserves per-partition order; single-partition topics = per-topic order
- **Idempotency**: Go consumers use `AckFunc` after successful dispatch — at-least-once + idempotent handler = effectively-once (unchanged)

## Quality Scenarios

### QS-01: Kafka broker disconnect → health degraded → auto-recover

| Element | Content |
|---------|---------|
| Category | Fault Tolerance |
| Level | L2 |
| Stimulus Source | Docker: `docker stop kafka-kafka-1` |
| Stimulus | Kafka broker unreachable |
| Artifact | taskEvents-accounts consumer (port 18025) |
| Environment | Degraded |
| Response | `/api/health/ready` → `{"status":"degraded"}`; consumer retries with backoff |
| Response Measure | Health flips to `degraded` within 5s of disconnect; flips back to `ok` within 10s of broker recovery |

### QS-02: Consumer groups appear in Kafka UI after switch

| Element | Content |
|---------|---------|
| Category | Observability |
| Level | L2 |
| Stimulus Source | Operator browses `http://183.250.1.132:18080/ui/clusters/local/consumer-groups` |
| Stimulus | TaskEvents consumers started with transport=kafka |
| Artifact | Kafka UI consumer groups page |
| Environment | Normal |
| Response | Page lists 6 consumer groups: task-events-accounts, task-events-projects, task-events-cloud, task-events-realtime, task-events-billing, task-events-notifications |
| Response Measure | All 6 groups visible with non-zero member count within 15s of taskEvents startup |

## Domain Model Impact

| NFR Decision | Model Impact | DDD Action |
|-------------|-------------|------------|
| No consistency model change (same envelope, same ack semantics) | No change to aggregate boundaries or event schemas | Keep existing domain event types and handler interfaces |
| Kafka broker health replaces Redis health | `ConnectionState` entity unchanged; only instantiation source changes | No new domain concept — `EventBrokerPort` interface already abstracts transport |
| No new NFR requires new domain abstractions | — | DDD step is lightweight — confirm existing abstractions hold |

## Trade-offs & Boundaries

### Trade-offs
- **Kafka single-node vs Redis single-node**: Both are single-node for dev; Kafka adds Zookeeper dependency but provides consumer group visibility (the goal)

### Explicit non-goals
- No Kafka cluster HA (multi-broker) — dev environment only
- No message schema registry (Avro/Protobuf) — JSON envelope unchanged
- No Kafka Connect / stream processing — not in scope
- No performance benchmarking — L1 basic, no new targets

### Upgrade triggers
- When multi-node Kafka cluster is deployed → Availability from L1 to L2, add broker failover test
- When production traffic > 1000 events/sec → Performance from L1 to L2, add P95 latency SLO

## Skip Declarations
- **Security (L0)**: Kafka runs on localhost:9093, no authentication configured. No change from current state — both Redis and Kafka are localhost-only.
- **Scalability (L0)**: Single-node dev environment. No horizontal scaling in scope.
- **Compliance (L0)**: No regulatory data flows through domain events. No GDPR/PII in event payloads.
- **Performance (L1)**: Config change only — no code path change that would affect throughput. If baseline was acceptable with Redis, Kafka equivalent is acceptable.
