# 价值流：消息队列 Kafka → Redis + 去除 memory 模式

> 设计：`docs/superpowers/specs/2026-05-31-message-queue-kafka-to-redis-design.md`

## 价值摘要

**运维与开发**统一依赖 Redis 作为消息基础设施，减少 Kafka 容器与主题管理；**终端用户**的注册链、邮件投递、领域事件异步副作用在 Django 重启后仍由独立消费者处理，且配置层不再存在「内存同步 / Kafka / Redis」三岔路的歧义。

## Related Value Streams

| 既有流 | 关系 |
|--------|------|
| **domain-events-consumer-split** | **modification** — Increment 2 原「redis + memory 对称」改为「redis 为默认、废弃 memory」；`increment2-in-memory-sync` 步骤待 deprecated |
| **domain-events-consumer-split** | **extension** — 复用 `increment2-transport-redis-memory`、`test_task_events_redis_transport.py` 作为 Redis 验收 |
| **user-auth** | **dependency** — 注册 `USER_CREATED` 链在 `memory=false` 后必须 taskEvents-accounts 在线 |
| **runall-cascade-*** | **dependency** — health 探针从「跳过 kafka」改为「redis 必检、kafka 可选」 |

**类型**：平台基础设施 **refactor**（配置 + 代码清理），非新用户功能。

## 端到端流

```
[运维改 port_config：transport=redis, email_queue.type=redis, memory=false]
  → [Redis 容器就绪]
  → [Django RedisStreamEventPublisher 发布]
  → [taskEvents-* / Saas_email 消费]
  → [Internal API / SMTP]
  → [用户：注册后公司创建、邮件送达、SSE 推送]
```

## 价值增量

### Increment 1：配置切换 — Redis 为默认传输（Thin Slice）

**用户/运维价值：** 改配置并重启后，领域事件与邮件队列均走 Redis，无需 Kafka 主题。

**范围：**

- `port_config.json`：`domainEvents.transport=redis`、`email_queue.type=redis`、`django.messageQueue.memory=false`
- 同步 `port_config.json.md`
- 重启 runAll / taskEvents / Saas_email consumer

**依赖：** `dockerInfra.redis`、既有 `RedisStreamEventPublisher` / `RedisStreamBroker`

**验证：**

| 步骤 | test_file |
|------|-----------|
| port-config-redis-default | `tests/test_domain_events_port_config.py` |
| redis-transport-publish | `tests/test_task_events_redis_transport.py` |
| email-register-e2e | `accounts/view_test/UserViewSet_email_register_test.py` |

**字段：**

- `saas-backend.config.domain_events_transport` → `redis`
- `saas-backend.config.email_queue_type` → `redis`（新增字段名建议）
- `task-events-accounts.health.status`

---

### Increment 2：去除 memory 代码路径

**价值：** 配置与运行时仅 redis/kafka 二选一；消除 dev/prod 行为分叉与文档歧义。

**范围：**

- 删除或废弃 `use_memory_message_queue()`、`InMemoryEventPublisher` 默认路径
- `run.sh` 移除 memory 分支与 Kafka 主题创建（redis 模式）
- health：`kafka` skipped、`redis` required
- 迁移 `test_in_memory_services.py` → explicit fixture 或 deprecated

**验证：**

| 步骤 | test_file |
|------|-----------|
| no-memory-default | 新增或扩展 `tests/test_port_config_merge.py` |
| health-redis-required | `tests/test_health_endpoint.py` |
| register-without-memory | `accounts/view_test/UserViewSet_email_register_test.py`（需 taskEvents） |

**字段：**

- `saas-backend.config.domain_events_transport`（禁止 `memory` 枚举）
- `saas-backend.health.redis.status`

---

### Increment 3：Kafka 降级为可选（Enhancement）

**价值：** docker-compose / runAll 可不启动 Kafka；文档标明 kafka 仅作兼容回退。

**范围：**

- `dockerInfra` 文档；runAll 组 optional kafka
- 保留 `transport=kafka` 代码路径供回滚

**验证：** 配置切换回 kafka 的 smoke（manual 或 integration tag）

**Future：** 完全移除 Kafka 依赖与镜像

---

## 增量排序

1. **Increment 1** — 配置改 redis + memory=false，端到端验收
2. **Increment 2** — 删 memory 路径（完成用户「仅 redis/kafka」约束）
3. **Increment 3** — Kafka 可选化 / 运维文档

## 与 value-stream.yaml 的 reconciliation

| 现有 step | 动作 |
|-----------|------|
| `domain-events-consumer-split` / `increment2-transport-redis-memory` | 保留，`domain_events_transport` 默认改为 redis |
| `increment2-in-memory-sync` | **deprecated** — memory 模式移除 |
| 新增 `message-queue-redis-default` 流 | 或追加到 `domain-events-consumer-split` 下新 step |

## 自检

- [x] Increment 1 端到端可验收（配置 → Redis → 注册链）
- [x] Increment 2 交付「仅 redis/kafka」用户决策
- [x] 依赖既有 Redis 实现，无新 broker 开发
- [ ] YAML 写入 — **待用户确认路径**
