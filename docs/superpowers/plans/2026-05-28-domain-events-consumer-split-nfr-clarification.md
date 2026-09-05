# NFR 澄清: 领域事件消费者 — 按域多进程 + 可切换传输

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-28-taskkafka-consumer-split-design.md`（v2）
> - 价值流文档: `docs/superpowers/plans/2026-05-28-domain-events-consumer-split-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | **L3** | Django 重启 ≤60s 内，accounts 域 consumer 持续消费且无重复建公司 |
| 可维护性 | **L3** | 按域独立进程发布；`domainEvents.transport` 可切换 kafka/redis/memory |
| 容错机制 | L2 | Internal API 5xx 指数退避重试 ≤5 次；at-least-once + 幂等 |
| 数据一致性 | L2 | 单事件处理幂等；跨事件链（USER_CREATED→COMPANY_CREATED）最终一致 |
| 性能 | L1 | 非用户同步路径；端到端事件处理 P95 ≤ 30s（Increment 1） |
| 可伸缩性 | L1 | 单实例 per 域；峰值由 Kafka/Redis 承担，不横向扩 consumer V1 |
| 安全性 | L2 | Internal secret；事件 payload 不落敏感字段日志 |
| 可观测性 | L2 | 每域 `/api/health/` + event_type 计数/失败率日志 |
| 合规与隐私 | L0 | 不新增数据驻留或跨境要求 |
| 性能（cloud 域 Increment 4） | L2 | cloud handler 长任务：max.poll.interval 适配，不阻塞其他域 |

## 逐增量 NFR 分析

### Increment 1: accounts 薄切片 + 进程隔离

#### 可用性 — L3（核心）

- **量化**：`saas-backend` 进程重启（滚动或 `run.sh restart` 仅 Django）期间，`task-events-accounts` 保持 Running；重启完成后 60s 内积压 `USER_CREATED` 被处理完毕。
- **降级**：Django Internal API 不可达时 consumer **不提交 offset**（或进入重试），禁止静默丢事件。
- **质量场景**：QS-01、QS-02

#### 可维护性 — L3

- **量化**：runAll 可单独 `restart` `task-events-accounts`；与 `manage.py start_kafka_consumer` **互斥**（配置门禁）。
- **质量场景**：QS-03

#### 容错机制 — L2

- Internal HTTP 超时 30s；5xx/网络错误退避重试（初始 1s，最大 30s，最多 5 次）。
- `USER_CREATED` 建公司：Django 端以 `user_id` 幂等（已存在则 skip）。
- **质量场景**：QS-04

#### 数据一致性 — L2

- 传输语义：**at-least-once**；业务 **effectively-once**（幂等键 `user_id` / `company_id`）。
- 跨事件 `COMPANY_CREATED` 链：允许秒级最终一致，不要求与 HTTP 注册响应同事务。
- **领域影响**：实体用 `create_if_absent` / 状态转移，非 blind insert。

#### 性能 — L1

- 不优化用户-facing API；consumer → Internal API P95 ≤ 5s 即可（Increment 1）。

#### 安全性 — L2

- `X-TaskEvents-Internal-Secret` 必填；仅 bind `127.0.0.1`（dev）或内网。

#### 可观测性 — L2

- health 返回 `transport`、`subscribed_events`、`last_processed_at`（可选）。

---

### Increment 2: transport redis/memory

#### 可维护性 — L3

- `port_config.domainEvents.transport` 切换后，**无需改代码**即可在 kafka/redis 间切换（重启对应进程）。
- `memory` 时不启动任何 `task-events-*`（与 `django.messageQueue.memory` 一致）。

#### 数据一致性 — L2

- Redis Streams 使用 consumer group + XACK；与 Kafka offset 语义对齐。
- **禁止** 同时 kafka 与 redis 消费同一逻辑事件（配置校验）。

#### 可用性 — L1

- redis 模式不要求 99.9%（开发/CI 为主）；kafka 模式沿用 L3。

---

### Increment 3–5: projects / realtime / cloud / billing

#### 可用性 — L3（延续）

- 各域进程独立重启；cloud 域重启不影响 accounts 注册链。

#### 性能 — L2（仅 cloud / realtime）

- `SSE_MESSAGE`：Redis publish P95 ≤ 200ms（consumer 侧）。
- cloud 启停：允许分钟级处理；**隔离**靠独立进程 + `max.poll.interval.ms` ≥ 300000。

#### 可伸缩性 — L1 → 触发 L2

- V1 每域单进程；当日事件量 >10k/min 单 topic 时评估按 partition 扩实例（升级触发）。

---

## 质量场景

### QS-01: Django 重启期间注册事件仍被处理

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L3 |
| 刺激源 | 运维 / 开发 `run.sh restart`（仅 Django） |
| 刺激 | 重启窗口内 Kafka 仍有 `USER_CREATED` 消息；重启后新注册再发一条 |
| 制品 | task-events-accounts + Django Internal API |
| 环境 | `transport=kafka`，docker-infra 正常 |
| 响应 | 两条事件均导致至多一家公司 per user_id；consumer 进程未退出 |
| 响应度量 | 重启前后 consumer PID 不变或 runAll 自动拉起；60s 内 `accounts_company` 行数符合预期；pytest/E2E 脚本断言 |

### QS-02: Django Internal API 短暂不可用

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 + 容错 |
| 等级 | L3 |
| 刺激源 | Django 启动中 / 崩溃 |
| 刺激 | consumer 收到 `USER_CREATED`，Internal API 返回 502 连续 3 次后恢复 200 |
| 制品 | task-events-accounts |
| 环境 | kafka 正常 |
| 响应 | 重试后成功处理；offset 仅在 200 后提交；无重复公司（幂等） |
| 响应度量 | 日志含 retry 计数；DB 中 company 仅 1 条；集成测试模拟 502 |

### QS-03: 仅重启 accounts 域消费者

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L3 |
| 刺激源 | 运维 |
| 刺激 | `runAll` 或 `taskEvents/run.sh restart accounts` |
| 制品 | task-events-accounts |
| 环境 | 正常 |
| 响应 | accounts 进程重启；`task-events-cloud`（若已部署）与 Django 不受影响 |
| 响应度量 | health 端点恢复 < 10s；其他进程 health 无失败 |

### QS-04: 重复投递 USER_CREATED

| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | Kafka 重平衡 / 至少一次语义 |
| 刺激 | 同一 `user_id` 的事件投递 2 次 |
| 制品 | Django Internal API `accounts` handler |
| 环境 | 正常 |
| 响应 | 第二次为 no-op；仍返回 200 |
| 响应度量 | `Company.objects.filter(creator_id=user_id).count() == 1` |

### QS-05: transport 切换 redis（Increment 2）

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L3 |
| 刺激源 | 开发配置 |
| 刺激 | `domainEvents.transport` 从 `kafka` 改为 `redis`，重启 accounts consumer |
| 制品 | Go RedisStreamBroker + Django Redis publisher |
| 环境 | 仅 Redis，无 Kafka 容器 |
| 响应 | 注册后事件被 accounts 消费并建公司 |
| 响应度量 | `test_port_config_merge` + 新 redis E2E 通过 |

### QS-06: memory 模式不启 consumer 进程

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | pytest / 本地 dev |
| 刺激 | `django.messageQueue.memory=true` |
| 制品 | runAll + taskEvents |
| 环境 | 测试 |
| 响应 | 无 `task-events-*` 进程；`InMemoryEventPublisher` 同步 handler |
| 响应度量 | `test_in_memory_services` 通过；runAll status 无 task-events |

---

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 可用性 L3 + 按域进程 | 限界上下文边界与 **消费者进程** 对齐；跨上下文仅用领域事件 | 每上下文独立 **EventHandler** 端口（应用层），非大聚合 |
| 一致性 L2 最终一致 | `USER_CREATED` 与 `COMPANY_CREATED` 分属 accounts 上下文，链式事件 | 小聚合 + `CompanyCreated` 领域事件发布 |
| 幂等 L2 | 建公司/工作区用业务键去重 | 实体方法 `ensure_default_company(user_id)` |
| 传输抽象 | 领域层 **不** 依赖 Kafka/Redis | `IEventPublisher` / `DomainEventBus` 端口在基础设施层实现 |
| cloud 长任务 L2 | cloud 上下文 handler 异步、可重入 | `CloudServerOperation` 状态机 + 幂等 `operation_id` |
| Internal API 防腐层 | 消费侧不直接 ORM | Go → HTTP → Django **Application Service** |

---

## 权衡与边界

### 取舍

- **进程隔离优先于消费延迟**：接受 Internal HTTP 一跳（L1 性能），换取 Django 独立部署（L3 可用性）。
- **最终一致优于注册 HTTP 与建公司同事务**：注册 API 可先 201，公司通过事件补齐（与现 Kafka 一致）。
- **Redis Streams 优先于 Pub/Sub**：换取持久化与 consumer group，配置略复杂。
- **按域多进程优先于单 Go 二进制**：运维进程数增加，故障域缩小。

### 明确不做什么（V1）

- 不做跨域 **Saga 编排器**（仍靠事件链 + 幂等）。
- 不做 exactly-once 端到端（保持 at-least-once + 幂等）。
- 不做 consumer 水平自动扩缩（L1 可伸缩）。
- 不在 V1 实现 DLQ topic（失败仅日志 + 重试上限，DLQ 为 Increment 5+ 可选）。
- 不合并 `Saas_email` 与 `domainEvents` 配置段。

### 升级触发条件

| 条件 | 升级项 |
|------|--------|
| 单 topic 消费 lag > 5min 持续 1h | 可伸缩性 L1→L2，cloud 域多实例 |
| 客户要求审计每条事件处理 | 可观测性 L2→L3，全量 AuditEvent + 保留 90 天 |
| Internal API P95 > 2s 且阻塞消费 | 性能 L1→L2，cloud handler 下沉或批量 API |
| 生产要求 zero message loss on broker crash | 一致性 L2→L3，Outbox + 事务发布 |

---

## 跳过声明

| 类别 | 理由 |
|------|------|
| 合规与隐私 | L0。不新增个人数据存储位置；事件 payload 沿用现有字段。 |
| 极致性能 | 非用户同步路径；Increment 1 标 L1。 |
| 多区域部署 | 不在范围；Kafka/Redis 单集群。 |

---

## 自检

- [x] 相关类别均有 L0–L4
- [x] L2+ 有量化目标
- [x] L2+ 有 QS（QS-01–06）
- [x] 响应度量可验证
- [x] 领域模型影响已标注
- [x] 权衡与边界已写
- [x] 跳过类别有理由
