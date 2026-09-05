# DDD 领域模型: 领域事件消费者 — 按域多进程 + 可切换传输

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-taskkafka-consumer-split-design.md`（v2）
> - 价值流: `docs/superpowers/plans/2026-05-28-domain-events-consumer-split-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-05-28-domain-events-consumer-split-nfr-clarification.md`
>
> **建模范围**: Increment 1（Broker 抽象 + accounts 域）+ 跨上下文事件契约；Increment 3–4 仅列上下文边界。

**NFR 约束摘要**: 可用性 L3 → 消费进程与 Django 解耦；一致性 L2 → at-least-once + 幂等；可维护性 L3 → 传输可切换、按域进程。

---

## 限界上下文 (Bounded Context)

| 上下文 | 职责 | 宿主 |
|--------|------|------|
| **event-delivery** | 订阅传输层、解析信封、路由到域命令、Ack/重试 | `taskEvents`（Go） |
| **accounts-onboarding** | 用户注册后确保默认公司与成员关系 | Django `accounts`（既有） |
| **organization-setup** | 公司创建后的交付物/进度/默认工作区 | Django `accounts` + `projects`（既有，Increment 2+） |
| **cloud-runtime** | 云主机启停与授权 | Django `cloud`（Increment 4） |
| **project-collaboration** | 工作区、任务、AI 回复持久化 | Django `projects`（Increment 3） |
| **realtime-delivery** | SSE 推送到 Redis | `taskEvents-realtime` / `taskSSE`（Increment 3） |
| **billing-audit** | 计费事件审计日志 | Django + `taskBill` emit（Increment 5） |

**上下文映射**:

```
accounts-onboarding --[USER_CREATED]--> event-delivery --[HTTP]--> accounts-onboarding
accounts-onboarding --[COMPANY_CREATED]--> event-delivery --> organization-setup
saas-backend (API) --[publish]--> Kafka/Redis --> event-delivery
```

---

## Increment 1: event-delivery 上下文

### 实体

- **ConsumerSession**（聚合根）：一次运行中的域消费者会话；持有 `domain_name`、`subscribed_events`、`transport`。
- **ProcessedEventRecord**（实体，聚合内）：已处理事件的幂等记录（`idempotency_key` + `processed_at`）。

### 值对象

- **EventEnvelope**: `event_type`, `data` (JSON), `key`, `occurred_at`
- **TransportKind**: `memory` | `kafka` | `redis`
- **ConsumerGroupID**: 如 `task-events-accounts`
- **IdempotencyKey**: 由 `event_type` + 业务键组成，如 `USER_CREATED:{user_id}`
- **BrokerCursor**: 传输层游标（kafka offset / redis stream id），opaque 给基础设施

### 聚合: ConsumerSessionAggregate

- **聚合根**: `ConsumerSession`
- **一致性边界**: 订阅列表 + 幂等表（内存或短期 TTL 缓存）+ 当前处理中的单条消息
- **不变量**:
  - 仅处理 `subscribed_events` 内的 `event_type`
  - 同一 `IdempotencyKey` 不得重复调用 DomainCommand
  - 仅当 DomainCommand 成功后才 Ack broker

### 领域服务

- **DomainEventRouter**: `event_type` → `DomainCommand` 描述（path、domain）
- **IdempotentDispatchService**: 查重 → 调用 `DomainCommandPort` → 记录 ProcessedEventRecord

### 端口（仓储 / 防腐层，ABC 或 Go interface）

| 端口 | 职责 |
|------|------|
| **EventBrokerPort** | `Subscribe`, `Ack`, `Close` — 无 Kafka/Redis 类型泄露到领域服务入参 |
| **DomainCommandPort** | `Dispatch(ctx, DomainCommand)` → 成功/可重试失败/永久失败 |
| **IdempotencyStorePort** | `Seen(key) bool`, `Mark(key)` |

### 领域事件（event-delivery 内部，过去式）

- `DomainEventReceived` — 已从 broker 拉取
- `DomainEventDispatched` — 已调用 Django Internal API 成功
- `DomainEventProcessingFailed` — 永久失败或重试耗尽

### 与传输实现的关系

```
domain/          ← 纯 Go，无 franz-go / redis  import
infrastructure/  ← KafkaBroker, RedisStreamBroker 实现 EventBrokerPort
application/     ← ConsumerApp 组装 session + 循环
```

---

## Increment 1: accounts-onboarding 上下文（Django，契约层）

> 不新建 ORM 实体；在应用层暴露 **策略** 与 **幂等** 契约，供 Internal API 实现。

### 领域服务（概念）

- **EnsureDefaultCompanyForUser**: 输入 `user_id`, `username`；若已有 creator 公司则 no-op；否则创建 Company + CompanyMember(admin)

### 领域事件（对外发布，已存在）

| 事件 | 触发 | 消费者 |
|------|------|--------|
| `UserCreated` | 注册成功 | event-delivery → accounts |
| `CompanyCreated` | 公司创建后 | event-delivery → organization-setup（Increment 2） |

### 幂等规则（NFR QS-04）

- `EnsureDefaultCompanyForUser`: 幂等键 `user_id`
- Internal API 返回 200 即表示「已处理或已存在」

### 仓储

- 沿用 Django `UserRepository` / `CompanyRepository`（基础设施层），领域层不新增表。

---

## Increment 2–5（边界占位，不全量建模）

| 上下文 | 聚合根（候选） | 关键领域事件 |
|--------|----------------|--------------|
| organization-setup | `Company` | `CompanyCreated`, `WorkspaceCreated` |
| cloud-runtime | `CloudServerOperation` | `CloudServerStarted`, `CloudServerStopped` |
| project-collaboration | `Task` | `TaskCompleted`, `ProjectUpdated` |
| realtime-delivery | —（无状态路由） | `SseMessagePublished` |
| billing-audit | — | `BillingTransactionRecorded` |

---

## 代码骨架落点

### Go — `taskEvents/domain/`（event-delivery）

| 文件 | 内容 |
|------|------|
| `event_envelope.go` | EventEnvelope, IdempotencyKey |
| `transport_kind.go` | TransportKind |
| `consumer_session.go` | ConsumerSession 聚合根 |
| `event_broker_port.go` | EventBrokerPort |
| `domain_command_port.go` | DomainCommand, DomainCommandPort |
| `idempotency_store_port.go` | IdempotencyStorePort |
| `event_router_service.go` | DomainEventRouter |
| `dispatch_service.go` | IdempotentDispatchService |
| `domain_events.go` | DomainEventReceived / Dispatched / Failed |

### Django — Increment 1 仅文档契约

- 实现落点（Build 阶段）：`accounts/application/onboarding/ensure_default_company.py` + `accounts/urls_task_events_internal.py`
- **不在本步创建 Django domain 新文件**，避免与既有 `accounts` 模型重复。

---

## 自检 (Hard Gate)

- [x] event-delivery 模型位于 `taskEvents/domain/`
- [x] 领域层无 Kafka/Redis/HTTP import（Go 端口仅 interface）
- [x] 幂等与 Ack 顺序符合 NFR L2/L3
- [x] 按域进程与 ConsumerSession.subscribed_events 对齐
- [x] accounts 幂等键 `user_id` 已声明
- [x] 跨上下文通信用领域事件名，非 topic 名硬编码在 Django 领域层

---

领域模型骨架已生成到 `taskEvents/domain/`。可进入 `/6-plans-实施计划`。
