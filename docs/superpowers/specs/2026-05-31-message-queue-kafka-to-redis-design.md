# 设计：消息队列基础设施 Kafka → Redis 切换

**日期**：2026-05-31  
**状态**：已实施（2026-05-31）  

| 驱动力 | 说明 |
|--------|------|
| **运维简化** | 本地/部署环境统一依赖 Redis，减少 Kafka 容器与主题管理 |
| **已有抽象** | 领域事件与邮件队列均已实现 kafka / redis 双实现，切换仅需配置 |
| **对齐先例** | `taskSSE.transport` 已为 `redis`；本次补齐其余队列 |

---

## 1. 背景与现状

项目将 **Redis** 与 **Kafka** 作为可互换的消息队列基础设施。传输层选型集中在 `task2app/conf/port_config.json`，代码侧已实现工厂/注册表切换，**无需改业务代码**。

### 1.1 当前 `port_config.json` 快照

| 配置路径 | 当前值 | 实际生效 |
|----------|--------|----------|
| `django.messageQueue.memory` | `true` | **内存同步**派发领域事件（优先级最高，覆盖 transport） |
| `domainEvents.transport` | `kafka` | 仅当 `memory=false` 时走 Kafka |
| `email_queue.type` | `kafka` | 邮件消费者走 Kafka（若进程已启动） |
| `taskSSE.transport` | `redis` | 已为 Redis，**无需改动** |
| `dockerInfra.redisPort` | `6379` | Redis 基础设施端口 |

### 1.2 代码入口（只读确认，本变更不修改）

| 子系统 | 配置键 | 实现 |
|--------|--------|------|
| 领域事件 Producer | `domainEvents.transport` | `registry.py` → `KafkaEventPublisher` / `RedisStreamEventPublisher` / `InMemoryEventPublisher` |
| 领域事件 Consumer | 同上 | `taskEvents/broker/factory.go` → `KafkaBroker` / `RedisStreamBroker` |
| 邮件队列 | `email_queue.type` | `Saas_email` → `kafka_consumer` / `redis_consumer` |
| 实时 SSE | `taskSSE.transport` | `taskSSE` Node 服务（已为 redis） |
| Pub/Sub（非领域事件） | — | 非 memory 模式固定 `RedisPubSubClient` |

---

## 2. 目标与非目标

### 2.1 目标

1. 将 **领域事件** 与 **邮件队列** 的默认传输从 Kafka 切换为 Redis。
2. 仅修改 `task2app/conf/port_config.json`（及同步文档 `port_config.json.md`）。
3. 切换后各域 `taskEvents` 消费者与 Django Producer 均使用 **Redis Streams**；邮件使用 **Redis channel**。

### 2.2 非目标

- 删除 Kafka 相关代码或 docker-compose 中的 Kafka 服务（可后续清理，本次不动）。
- 修改 `django.messageQueue.memory` 的语义（见 §4 决策点）。
- 变更 `Saas_Ai_Provider` 镜像 hash 消费者（当前仍硬编码 Kafka，独立域，不在本次范围）。

---

## 3. 配置变更方案

### 3.1 必改项

```json
{
  "domainEvents": {
    "transport": "redis"
  },
  "email_queue": {
    "type": "redis"
  }
}
```

`domainEvents.redis` 子块已存在且与 `dockerInfra.redisPort` 一致，无需新增字段：

```json
"redis": {
  "host": "127.0.0.1",
  "port": 6379,
  "db": 0,
  "streamKeyPrefix": "domain-events:"
}
```

### 3.2 文档同步（ companion 约束）

同一变更须更新 `task2app/conf/port_config.json.md`：

- 在 `domainEvents` / `email_queue` 说明中注明 **默认/推荐值为 redis**。
- 补充切换检查清单（见 §5）。

### 3.3 切换规则（与现有设计一致）

| 条件 | Producer | Consumer |
|------|----------|----------|
| `django.messageQueue.memory=true` | `InMemoryEventPublisher`（同步） | 不启动 taskEvents |
| `memory=false` + `transport=redis` | `RedisStreamEventPublisher` | taskEvents + Redis Streams |
| `memory=false` + `transport=kafka` | `KafkaEventPublisher` | taskEvents + Kafka |

**邮件队列**不受 `django.messageQueue.memory` 影响，仅看 `email_queue.type`。

---

## 4. 决策点：去掉 memory 模式（已批准）

**用户决定**：去掉 `django.messageQueue.memory` 方案；消息队列基础设施**仅**采用 **redis** 或 **kafka**，不再保留进程内内存同步路径作为生产/本地默认。

### 4.1 配置变更（相对原方案扩展）

除 §3.1 外，须：

```json
"django": {
  "messageQueue": {
    "memory": false,
    "kafka": false
  }
}
```

或直接删除 `memory` 键，由代码默认走 `domainEvents.transport`。

### 4.2 代码影响（超出「仅改 JSON」）

| 区域 | 变更 |
|------|------|
| `registry.py` | 移除 `_get_use_in_memory()` 对 `memory` 的优先分支；pytest 改用 `transport=redis` + mock/fixture |
| `port_config.py` | 废弃 `use_memory_message_queue()` 或改为恒 `false` |
| `run.sh` | 删除「内存消息队列」分支；Redis 健康检查替代 Kafka 主题创建 |
| 测试 | `test_in_memory_services.py` 迁移或 deprecated；CI 依赖 Redis 容器 |
| 文档 | `IN_MEMORY_SERVICES.md`、`port_config.json.md` 更新 |

**本设计仍分两阶段交付**：

1. **Increment 1（配置切换）**：`transport=redis`、`email_queue.type=redis`，`memory=false` — 验证 Redis 端到端。
2. **Increment 2（去 memory）**：删除 memory 代码路径与相关测试/文档 — 完成「仅 redis/kafka」约束。

### 4.3 测试策略替代

| 原 memory 用途 | 替代 |
|----------------|------|
| pytest 同步 handler | `@pytest.fixture` + `InMemoryEventPublisher` 显式注入，或 `transport=redis` + fakeredis |
| dev 热重载 | 接受 taskEvents 独立进程；Django 热重载不再同步派发领域事件 |

---

## 5. 切换后验证清单

1. **Redis 可达**：`docker compose up redis` 或 `task2app/Saas_project/docker-compose.yml` 中 redis 服务运行。
2. **重启进程**（配置热加载不完整）：
   - Django / runAll 主站
   - `taskEvents` 五域消费者（runAll 组或 `taskEvents/run.sh`）
   - `Saas_email`：`manage.py start_email_consumer`（读 `email_queue.type`）
3. **pytest**（已有覆盖）：
   ```bash
   cd task2app/Saas_project && DJANGO_SETTINGS_MODULE=saas_project.settings_test \
     pytest tests/test_domain_events_port_config.py \
            tests/test_task_events_redis_transport.py -q
   cd taskEvents && go test ./broker/... ./config/... -q
   ```
4. **冒烟**：注册触发 `USER_CREATED` → accounts 消费者 → 公司创建；发邮件任务 → redis channel 消费。

### 环境变量覆盖（可选，不改 JSON 时）

| 变量 | 作用 |
|------|------|
| `USE_IN_MEMORY_SERVICES=true` | 强制内存模式 |
| `DOMAIN_EVENTS_TRANSPORT=redis` | 覆盖 domainEvents.transport |
| `QUEUE_TYPE=redis` | 覆盖 email_queue.type |

---

## 6. 价值流影响

读取 `value-stream.yaml` 后，本变更**不新增**价值流，仅激活已有 Redis 传输步骤：

| 价值流 | 步骤 | 影响 |
|--------|------|------|
| `domain-events-consumer-split` | `increment1-accounts-thin-slice` | `saas-backend.config.domain_events_transport` 取值变为 `redis` |
| 同上 | `increment2-transport-redis-memory` | 已有 Redis/memory 联动测试，**无需新测** |
| 同上 | `increment3-sse-realtime` | taskSSE 已为 redis，无变化 |

**字段**：无新字段；`domain_events_transport` 枚举值从 `kafka` → `redis`。  
**测试**：现有 `test_task_events_redis_transport.py` 等已覆盖，价值流 mapping 在 step 3 仅需确认 status 仍为 `active`。

---

## 7. 领域概念清单（供 `/5-ddd`）

| 类型 | 候选 |
|------|------|
| **限界上下文** | 领域事件（domainEvents）、邮件投递（email）、实时推送（taskSSE） |
| **实体/值对象** | DomainEvent `{event_type, data, key}`；Redis Stream 键 `domain-events:all` |
| **聚合** | 各域 Consumer 进程（accounts / projects / cloud / realtime / billing） |
| **领域事件** | 现有 14+ event_type 不变；仅传输层替换 |
| **跨上下文** | SSE_MESSAGE：realtime 消费者写 Redis channel → taskSSE 订阅（已设计） |

---

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| memory=true 时改 transport 无运行时效果 | 文档与 §4 明确；验收时按选定选项测 |
| 旧 Kafka 中未消费消息 | 切换前 drain 或接受丢失（dev 可忽略） |
| run.sh 仍启动 Kafka 容器 | 无害冗余；后续可从 docker-compose 移除 kafka 服务 |
| Redis 单点 / 无持久化 | 与现 taskSSE 一致；生产需 AOF/RDB 策略（非本次） |

---

## 9. 实施步骤（批准后）

1. 修改 `task2app/conf/port_config.json`（§3.1；按 §4 决定是否改 `memory`）。
2. 同步 `task2app/conf/port_config.json.md`。
3. 重启相关服务并执行 §5 验证。
4. （可选）更新 `value-stream.yaml` 中 `domain_events_transport` 的 description 示例值为 redis。

**预估工作量**：配置 + 文档 < 30 分钟；无代码变更。

---

## 10. 批准记录

| 审阅人 | 日期 | 结论 |
|--------|------|------|
| | | 待填 |
