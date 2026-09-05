# DDD — 交易记录流水账读模型

Bound context: **Billing**（taskBill owner）。本增量不拆新限界上下文。

## 已有聚合（不改一致性边界）

- `BillingAccount`：余额、`task_post_quota`
- `BillingTransaction`：amount、balance_before/after、usage_amount、billing_unit
- `BillingResourceGrant`：quantity/remaining（FEFO 消耗）

## 新增读模型（非聚合）

**LedgerSnapshot**（Value Object）

- `BalanceBefore` / `BalanceAfter`（分）
- `DisplayLines`（现金句 + 可选资源剩余句）

**ResourceChange**（Value Object）

- `ResourceType`、`Quantity`、`RemainingAfter?`、`Display`

推导服务：`TransactionLedgerReadService`（实现落在 `taskBill/src` 的 attach* 函数，不新建独立包以免空抽象）。

## 端口

无新端口。继续用既有 `*sql.DB` 查询（存量服务模式）。不引入 Kafka 新契约。

## 事件

无。纯查询例外已写入意图文档。

## NFR 落地

- 租户隔离：所有查询含 `account_id` / `tenant_id`
- 任务帖剩余：未过滤回放，避免筛选破坏快照
