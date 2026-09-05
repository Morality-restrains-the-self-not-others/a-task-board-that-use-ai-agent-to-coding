# NFR 澄清: 消息队列 Kafka → Redis + 去除 memory 模式

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-31-message-queue-kafka-to-redis-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-31-message-queue-kafka-to-redis-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | **L3** | Redis 模式下 Django 重启 ≤60s 内，各域 consumer 持续消费；Redis 宕机时 health 失败、不静默丢事件 |
| 可维护性 | **L3** | 传输仅 `kafka` \| `redis`；配置切换后重启进程即可，无 memory 三岔路 |
| 数据一致性 | L2 | Redis Streams XACK + at-least-once；业务幂等（user_id / company_id） |
| 容错机制 | L2 | Internal API 5xx 指数退避 ≤5 次；Redis 断连 consumer 不 XACK |
| 性能 | L1 | 异步路径；consumer → Internal API P95 ≤ 5s |
| 可伸缩性 | L1 | 单实例 per 域；Redis Streams consumer group 承担积压 |
| 安全性 | L2 | Internal secret；Redis 仅内网/dev bind |
| 可观测性 | L2 | health 含 redis 探针；transport 字段暴露当前 broker |
| 合规与隐私 | L0 | 不新增数据驻留要求 |
| 迁移/回滚 | L2 | 切换窗口内允许 at-most-once 丢失（dev）；生产需 drain 或双写窗口 |

## 相对既有 NFR 的变更

| 原 `domain-events-consumer-split` NFR | 本变更 |
|----------------------------------------|--------|
| Increment 2 支持 `memory` + redis 对称 | **移除 memory**；pytest/dev 统一 redis 或显式 fixture |
| redis 模式可用性 L1（dev/CI 为主） | **Increment 1 起 redis 为默认**，可用性 **L3** 与 kafka 对齐 |
| `django.messageQueue.memory` 联动 | **废弃**；health 不再 skip redis |

---

## 逐增量 NFR 分析

### Increment 1: 配置切换 Redis 默认

#### 可用性 — L3

- **量化**：`domainEvents.transport=redis` 且 `memory=false` 时，`/api/health/` 中 redis check **不 skipped**；Redis 不可达时 health **503**。
- **量化**：Django 滚动重启 60s 内，`USER_CREATED` 积压由 `task-events-accounts` 消费完毕。
- **降级**：Redis 不可用时，领域事件发布失败应 **显式报错/重试**，禁止 fallback 到 memory。
- **场景**：QS-01、QS-02

#### 可维护性 — L3

- `port_config.json` 三处一致：`domainEvents.transport`、`email_queue.type`、`django.messageQueue.memory=false`。
- 文档 `port_config.json.md` 与配置同提交。
- **场景**：QS-03

#### 数据一致性 — L2

- Redis Streams：`XADD` + consumer group + `XACK`；语义对齐 Kafka offset。
- **禁止** kafka 与 redis 同时消费同一 stream/topic（配置 + 单 transport 字段）。
- 邮件队列：`email_queue.type=redis` 使用 channel pub/sub，与领域事件 stream **键空间隔离**（不同 key/channel）。

#### 容错机制 — L2

- Redis 连接失败：publisher 抛错或重试 3 次（指数退避 1s/2s/4s）；consumer 不 ACK 直至 Internal API 成功。
- **场景**：QS-04

#### 性能 — L1

- 配置切换本身无性能目标；Redis publish P95 ≤ 50ms（dev 单节点）。

#### 可观测性 — L2

- health JSON：`checks.redis.status`、`checks.kafka.skipped=true`（redis 模式）。
- 日志：启动时打印 `Using RedisStreamEventPublisher` / `transport=redis`。

---

### Increment 2: 去除 memory 代码路径

#### 可维护性 — L3

- 删除 `use_memory_message_queue()` 默认路径后，**单一真相**：`domainEvents.transport`。
- `test_in_memory_services.py` 迁移为 opt-in fixture，不阻塞 CI。

#### 可用性 — L3（dev 行为变更）

- **权衡**：Django dev 热重载 **不再** 同步执行领域 event handler；开发者须保持 taskEvents 进程运行。
- **缓解**：runAll 一键启动 task-events 组；文档说明 dev 工作流。

#### 数据一致性 — L2

- pytest：使用 `transport=redis` + fakeredis / docker redis，或 `@override_settings` 注入 `InMemoryEventPublisher` **仅测试模块内**，非全局默认。

---

### Increment 3: Kafka 可选化（Future）

#### 可维护性 — L2

- 保留 `transport=kafka` 代码路径 1 个发布周期，供回滚。
- docker-compose 中 kafka 服务标记 optional。

#### 可用性 — L1

- 无 Kafka 容器时，仅 redis 模式可 full stack 启动。

---

## 质量场景

### QS-01: Django 重启后 Redis 积压消费

| 要素 | 内容 |
|------|------|
| 刺激 | `transport=redis`，注册 10 用户后 kill Django 再启动 |
| 环境 | task-events-accounts Running，Redis 正常 |
| 响应 | 60s 内 10 条 USER_CREATED 处理完，公司数 = 10 |
| 响应度量 | `UserViewSet_email_register_test` + consumer 日志无 duplicate error |

### QS-02: Redis 宕机 health 失败

| 要素 | 内容 |
|------|------|
| 刺激 | stop redis 容器，GET `/api/health/` |
| 响应 | 503 或 `checks.redis.status=fail`；**不** report overall ok |
| 响应度量 | `test_health_endpoint.py` |

### QS-03: 配置切换无需改代码

| 要素 | 内容 |
|------|------|
| 刺激 | `domainEvents.transport` kafka → redis，重启 Django + taskEvents |
| 响应 | 新注册仍触发建公司；无 Kafka 连接日志 |
| 响应度量 | `test_task_events_redis_transport.py` |

### QS-04: Internal API 失败不丢事件

| 要素 | 内容 |
|------|------|
| 刺激 | mock Internal API 返回 503 三次后 200 |
| 响应 | 第四次成功；Redis pending 不丢 |
| 响应度量 | 现有 taskEvents retry 测试或 manual |

### QS-05: 去除 memory 后 pytest CI

| 要素 | 内容 |
|------|------|
| 刺激 | CI 无 `USE_IN_MEMORY_SERVICES` |
| 响应 | 价值流 active steps 全绿；Redis 由 CI service 提供 |
| 响应度量 | valueStream `message-queue-kafka-to-redis` 流 |

---

## 领域模型影响（供 `/5-ddd`）

| NFR 决策 | DDD 影响 |
|----------|----------|
| 去除 memory 同步 | EventPublisher 为基础设施端口；测试用 TestDouble 注入，非领域层分支 |
| Redis Streams 单 stream key | `DomainEvent` 值对象不变；BrokerCursor 存 redis stream id |
| 邮件 queue 独立 type | 邮件限界上下文保持独立 `email_queue`，不与 domainEvents 聚合 |
| health redis 必检 | 无新聚合；运维读模型 via health endpoint |

---

## 明确不做

- Redis Cluster / Sentinel 高可用（本增量 L1 可伸缩，单节点 redis）
- 双写 kafka+redis 迁移窗口自动化（手动 drain / 接受 dev 丢失）
- 移除 Kafka 代码（Increment 3 为 optional，非本 Sprint 必须）
- Saas_Ai_Provider 镜像 hash 消费者改 Redis（独立 backlog）

---

## 自检

- [x] 每个 Increment 有 NFR 等级
- [x] 质量场景可映射到现有/计划测试
- [x] memory 移除的 dev 权衡已记录
- [x] 与 `domain-events-consumer-split` NFR 差异已 reconciliation
