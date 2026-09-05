# Value Stream: Domain Events Transport — Redis → Kafka 切换

> Derived from design: `docs/design/consumer-groups-kafka-transport-switch.md`

## Value Summary

**运维/开发** 在 Kafka UI (`:18080`) 中可观察全部 domain event consumer group 状态
(lag, rebalance, 活跃成员)；切换回 Kafka 作为消息基础设施后，消息持久化与消费者可观测性
均优于 Redis Streams。

## Related Value Streams

| 既有流 | 关系 |
|--------|------|
| **message-queue-kafka-to-redis** | **reversion** — `increment1-config-redis-default` (active) 被回退，transport 恢复为 kafka。`increment2-deprecate-memory-mode` (planned) 不受影响 |
| **domain-events-consumer-split** | **modification** — Increment 1 中 transport=redis → transport=kafka；Go broker factory 路由 switch 不变，但实例化为 KafkaBroker；health check 从 Redis ping 变为 Kafka ping |
| **runall-cascade-lifecycle** | **extension** — `docker-kafka` 组从「可选」变为「依赖」运行；Kafka 不可用时 taskEvents 无法 ready |
| **runall-docker-infra-split** | **dependency** — 依赖已完成的 `docker-kafka` 独立栈与 `18080` 就绪验证 |

**类型**: 平台基础设施 **reversion** — 配置 + 测试更新，无新代码。

## End-to-End Flow

```
[conf/domain-events/config.yaml: transport=kafka]
  → [Django KafkaEventPublisher produce 到 Kafka topic]
  → [Kafka broker (localhost:9093)]
  → [taskEvents Go KafkaBroker 消费 (18020-18037)]
  → [POST /api/internal/task-events/<domain>/]
  → [Django ORM 写库 / Redis SSE 推送]
  → [用户: 注册/公司创建/SSE 正常]
  → [Kafka UI :18080: consumer groups 可见 + lag 监控]
```

## Value Increments

### Increment 1: Transport 切换 — Redis → Kafka (Thin Slice)

**Value to user:** Kafka UI consumer groups 页面显示 6 个活跃 consumer group，
运维可监控 lag 与成员状态。

**Scope:**
- `conf/domain-events/config.yaml`: `transport: redis` → `transport: kafka`
- `conf/infra/docker-infra/config.yaml`: 增加 `kafka.bootstrapServers: localhost:9093`
- `tests/test_domain_events_port_config.py`: assert transport == "kafka"
- 重启 taskEvents consumers + Django

**Depends on:** `docker-kafka` 组已运行（已验证 — Kafka 容器 Up）

**验证:**

| Step | test_file |
|------|-----------|
| transport-kafka-default | `tests/test_domain_events_port_config.py` |
| kafka-broker-connect | `../../taskEvents/integration/registration_chain_test.go` |
| kafka-consumer-health | `tests/test_health_endpoint.py` |
