# 事件消费者幂等消费（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-18
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 49 条
- Cursor：`.cursor/rules/event-consumer-idempotency.mdc`（alwaysApply）
- ADR：`docs/adr/0015-event-consumer-idempotency.md`
- 门禁：`db/scripts/ci/check_event_consumer_idempotency.py`
- 自测：`db/scripts/ci/test_check_event_consumer_idempotency.py`

## 背景（为何是元规则）

Kafka 是 **at-least-once**：同一条领域事件可能因生产者重试、消费者超时后重投、rebalance、进程在 Ack 前崩溃而被处理两次。若 handler 直接「再扣一次 / 再停一次 / 再发一封」，副作用会翻倍。

本仓库已踩过键过粗的坑：用 `company_id` / `user_id` 当消费幂等键，会把**后续独立意图**静默吞掉（失败经验 16、64）。NFR 阶段（元规则 48）只保证设计文档写了键；本条强制**实现**走共享幂等消费，且键与业务重复边界同粒度。

## 核心原则

**领域事件消费采用 at-least-once + 幂等 handler = effectively-once。禁止假设 Kafka 只投递一次。禁止在业务 HTTP 服务里自建无幂等的 consume loop。**

## 强制要求

### 1. 统一消费入口

- Kafka 消费**只允许**落在 `taskEvents` intent worker
- 必须经 `eventbin.RunIntent` / `eventbin.Run` → `consumer.RunWithDelivery` → `domain.IdempotentDispatchService`
- **禁止**在 `taskBill` / `taskCloudService` 等业务进程用 `kafka.NewReader` / `kafka.Reader` 自建消费循环
- `eventbin.RunIntent` 的 keyFn **不得为 nil**（显式传入 `consumer.IdempotencyKeyFromEnvelope` 或专用函数）

### 2. 幂等键与业务重复边界同粒度

- 键必须标识「同一笔用户意图 / 同一笔业务」，而不是「同一个租户/用户从此只能成功一次」
- **禁止**默认用 `company_id` / `tenant_id` / `user_id` 作事件消费幂等键
- 优先：`event_id` / `stop_request_id` / `transaction_id` / 业务单号；其次 `task_id`（或 `task_id` + 状态指纹）
- 通用 `IdempotencyKeyFromEnvelope` 字段序必须让 `event_id`、`task_id` 排在 `user_id` / `company_id` 之前
- 载荷固定带租户/用户字段的事件，必须写**专用** key 函数（参考 `CLOUD_SERVER_STOPPED`、`TASK_CREATED`）

### 3. 重放语义

| 情况 | 必须行为 |
|------|----------|
| 相同幂等键再次到达 | `DispatchSuccess` + Ack；**不得**再执行副作用 |
| 幂等命中 | **必须**打 `warn` 日志 `idempotency skip`（`event_type` + 键指纹）；禁止静默跳过 |
| handler 失败（可重试） | **不得** `Mark`；以便重试真正执行 |
| 幂等键含 token/密码 | 日志只写 SHA-256 指纹，禁止完整键 |

### 4. 存储

- 默认：进程内 `idempotency.MemoryStore`（L2 可接受；进程重启后可能再执行一次，handler 仍须自然幂等）
- 资金 / 配额 / 云资源 / 支付（NFR ≥ L3）：**owner 服务**须有 DB 唯一约束或幂等表作为最终去重；MemoryStore 只是快路径，不是唯一手段
- 禁止把「消费者自己会去重」当成跳过 NFR 审视的理由（元规则 48 仍要写键）

### 5. 测试（新增/修改消费者时）

- **重放**：同一幂等键两次投递 → handler 副作用只发生一次
- **不塌缩**：不同业务 ID（不同 `task_id` / `event_id`）不得因同租户/同用户被当成一次
- 改 key 推导时同步改 `taskEvents/consumer/key_test.go`

Timer worker 不是 Kafka 消费者：其调用的 owner 一次性 API 仍须 HTTP 幂等（元规则 48），不走本条的 `RunIntent` 门禁。

## 实现参考

| 组件 | 路径 |
|------|------|
| 共享 runner | `taskEvents/consumer/runner.go` |
| 幂等调度 | `taskEvents/domain/dispatch_service.go` |
| 默认/专用键 | `taskEvents/consumer/key.go` |
| 进程内 store | `taskEvents/idempotency/memory.go` |
| intent 入口 | `taskEvents/eventbin/run.go`、`taskEvents/cmd/<event>/<intent>/main.go` |

## 与既有规则的关系

| 规则 | 管什么 | 与本条关系 |
|------|--------|------------|
| NFR 幂等性审视（元规则 48） | 设计阶段写清键与边界 | 本条是**消费实现**硬约束；48 的键必须在此落地 |
| 死信 Topic | 永久失败 / 重试耗尽 | 本条管成功路径重复投递；失败仍走 DLT |
| 业务服务禁止进程内轮询 | 禁止业务进程自唤醒 | 本条禁止业务进程自建 Kafka consume |
| 意图 → MQ 事件 | 成功路径必须发事件 | 本条约束事件被消费时不得重复副作用 |

## 验收

```bash
python3 db/scripts/ci/test_check_event_consumer_idempotency.py
python3 db/scripts/ci/check_event_consumer_idempotency.py
cd taskEvents && go test ./consumer/ ./domain/ -count=1
```

## 变更日志

- 2026-08-18：版本 1.0.0 - 初版；与 ADR-0015 同步
