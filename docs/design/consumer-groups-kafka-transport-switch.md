# 设计文档: Domain Events Transport 切换 — Redis → Kafka

## 背景

当前 `conf/domain-events/config.yaml` 中 `transport: redis`，所有领域事件经 Redis Streams
生产/消费。Kafka 集群虽在运行（dockerInfra/kafka），但无任何消费者连接，导致 Kafka UI
(`http://183.250.1.132:18080/ui/clusters/local/consumer-groups`) 显示
"No active consumer groups found"。

**目标**: 将 domain events transport 从 Redis 切换为 Kafka，使 taskEvents 消费者连接
Kafka 并以 consumer group 形式在 Kafka UI 中可见。

## 现状架构

```
                    transport=redis
 Django (publisher) ──→ Redis Streams ──→ taskEvents Go consumers (18020-18037)
     ↑                    ↑                        ↑
  registry.py       redis_stream_             broker/redis.go
  → RedisStream      event_publisher.py       → RedisStreamBroker
    EventPublisher

 Kafka 集群: 运行但空闲 (无 producer / 无 consumer)
```

## 目标架构

```
                    transport=kafka
 Django (publisher) ──→ Kafka (9093) ──→ taskEvents Go consumers (18020-18037)
     ↑                    ↑                       ↑
  registry.py       kafka_event_             broker/kafka.go
  → KafkaEvent       publisher.py            → KafkaBroker
    Publisher          ↓                        ↓
                   core/kafka/            Consumer Groups
                   producer.py            (Kafka UI 可见)
```

## 变更范围

### 1. 配置文件修改

#### 1.1 `conf/domain-events/config.yaml`

```yaml
# Before
transport: redis

# After  
transport: kafka
```

这是**唯一必需的配置变更**。Kafka bootstrap servers 由 taskEvents Go 代码默认提供
(`localhost:9093`)，与 docker-compose 中 `KAFKA_ADVERTISED_LISTENERS` 的
`PLAINTEXT_HOST://localhost:9093` 对齐。

#### 1.2 `conf/infra/docker-infra/config.yaml` (建议)

当前 `kafka` block:
```yaml
kafka:
  host: 127.0.0.1
  kafkaUiPort: 18080
```

建议增加 `bootstrapServers` 以便 Go config loader 可解析：
```yaml
kafka:
  host: 127.0.0.1
  bootstrapServers: localhost:9093
  kafkaUiPort: 18080
```

> 非阻塞 — Go 默认值 `localhost:9093` 已正确。但显式化有利于运维可读性。

#### 1.3 同步链

```
conf/infra/docker-infra/config.yaml
    ↓ sync.sh
conf/domain-events/docker-infra.yaml
    ↓ conf_yaml.go loadDomainEventsGlobal()
taskEvents Config.BootstrapServers
```

`conf/core/django/domain-events.yaml` 由 `conf/core/django/sync.sh` 从
`conf/domain-events/config.yaml` 同步 → Django `DOMAIN_EVENTS_TRANSPORT` 自动读取新值。

### 2. 代码变更 (无需改动)

**代码层面无需修改**——两端均已支持 Kafka transport：

| 层 | 文件 | Kafka 路径 |
|----|------|------------|
| Go 消费者 | `taskEvents/broker/kafka.go` | `NewKafkaBroker` — 完整实现 |
| Go config | `taskEvents/config/conf_yaml.go` | 已解析 `transport` 并路由到 Kafka |
| Go factory | `taskEvents/broker/factory.go` | `TransportKafka` → `NewKafkaBroker` |
| Django publish | `core/services/kafka_event_publisher.py` | `KafkaEventPublisher` — 完整实现 |
| Django factory | `core/services/registry.py` | transport≠redis 时默认走 Kafka |
| Django producer | `core/kafka/producer.py` | `confluent_kafka.Producer` — 完整实现 |

### 3. Kafka Topics 创建

`task2app/Saas_project/core/kafka/config.py` 定义了完整的 `KAFKA_TOPICS` 映射（14 个 topics）。
Topics 由 `scripts/init/02_02_create_kafka_topics.py` 自动创建。

Kafka broker 配置 `auto.create.topics.enable` (默认 true) 也会在首次 produce 时自动创建 topic，
但推荐显式运行 init 脚本确保 topics 就绪。

### 4. 重启服务

| 服务 | 影响 | 操作 |
|------|------|------|
| taskEvents consumers (18020-18037) | transport 变更后需重连 | `run.sh restart taskEvents` |
| Django (saas-backend, :8001) | publisher 切换到 Kafka | 重启 Django |
| Kafka 集群 | 无需变更 | 已在运行 |
| Redis | 仍需运行 | pub/sub + 缓存仍依赖 Redis |

## 验证清单

### 功能验证

1. Kafka UI consumer groups 页面显示各 consumer group:
   - `task-events-accounts`
   - `task-events-projects`
   - `task-events-cloud`
   - `task-events-realtime`
   - `task-events-billing`
   - `task-events-notifications`

2. 注册新用户 → Kafka UI 中 `user-created` topic 有消息入站，`task-events-accounts` consumer group 有 lag 变化

3. taskEvents health API 正常:
   - `http://127.0.0.1:18020/api/health/ready` → `{"status": "ok"}`
   - `http://127.0.0.1:18025/api/health/ready` → `{"status": "ok"}`

### 回归验证

4. 已有测试仍通过 — 测试使用 `USE_IN_MEMORY_SERVICES=true`，不受 transport 影响
5. Redis pub/sub 仍正常（taskSSE 等不经过 domain events transport）
6. taskEvents integration tests 通过

## 风险与回滚

### 风险

| 风险 | 级别 | 缓解 |
|------|------|------|
| Kafka consumer group rebalance 导致短暂消息延迟 | 低 | 单节点无 rebalance；消息持久化在 Kafka |
| Redis → Kafka 后消息格式不一致 | 低 | Envelope 格式 (`{event_type, data}`) 在两端一致 |
| Kafka broker 故障 | 中 | taskEvents broker 有退避重连 (backoff.go)，Kafka 已有 7 天数据保留 |

### 回滚

```bash
# 1. 改回 Redis transport
sed -i 's/transport: kafka/transport: redis/' conf/domain-events/config.yaml

# 2. 重启 taskEvents consumers
# runAll UI 或 run.sh restart taskEvents

# 3. 重启 Django
```

## 领域概念清单

| 概念 | 类型 | Bounded Context |
|------|------|-----------------|
| Domain Event | Domain Event | 跨 Bounded Context 异步消息载体 |
| Transport Backend | Value Object | 消息传输后端选择（kafka/redis） |
| Consumer Group | Entity | Kafka consumer group — 含 groupId + 订阅 topic 列表 |
| Event Publisher | Domain Service | 发布领域事件的抽象接口 |
| Event Broker | Domain Service | 消费领域事件的抽象接口 |
| Kafka Topic | Infrastructure | 事件类型 → Kafka topic 映射 |

## 价值流影响

| 价值流 | 影响 |
|--------|------|
| `message-queue-kafka-to-redis` | **反向操作** — `increment1-config-redis-default` (active) 被回退。该价值流的 `increment2-deprecate-memory-mode` (planned) 不受影响 |
| `domain-events-consumer-split` | transport 切换不影响消费者拆分结构，但 health check 从 Redis ping 变为 Kafka ping |
| `runall-cascade-lifecycle` | `docker-kafka` 组从「可选运行」变为「依赖运行」— Kafka 不可用时 taskEvents 无法启动 |

## 推荐实施顺序

1. **配置修改**: 改 `transport: kafka` + 增加 `bootstrapServers`
2. **Topic 预建**: 运行 `02_02_create_kafka_topics.py`
3. **重启消费者**: taskEvents consumers 连接到 Kafka
4. **验证 Kafka UI**: consumer groups 可见
5. **重启 Django**: publisher 切换到 Kafka
6. **端到端验证**: 注册用户 → 追踪事件流
