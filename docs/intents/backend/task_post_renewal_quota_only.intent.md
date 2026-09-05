# 意图：任务帖续存只扣配额、不另扣费

## 背景与目标

购买任务帖按 0.55 元/帖获得「创建帖次数」。续存应再消耗 1 次配额并延长 12 个月，**不得**再扣钱包余额，也不得按 `server_start_renewal` 单价记一笔付费消费。

## 范围与边界

- 范围内：`consumeTaskPostRenewal`；公开定价 `task_renewal_points`；管理端「续存同价」文案；`updateResourcePricing` 不再把创建帖价写入续存单位。
- 范围外：不改购买任务帖定价；不改存续期算法；不改配额不足时的 402。

## 约束与风险

- 续存与创建帖一样：只减 `task_post_quota`（FEFO），流水 `amount=0`，单位 `task_post_quota`（价 0）。
- 钱包 `balance` 不变。
- 审计流水保留（与创建帖消耗配额同模式），不视为扣费。

## 验收标准

1. 配额 3、余额 9999 时续存成功 → 配额 2、余额仍 9999、`cost_cents=0`。
2. 流水挂在 `task_post_quota`，`amount=0`，不使用标价的 `server_start_renewal`。
3. 配额为 0 时续存失败（402），余额不变。
4. 公开定价 `task_renewal_points=0`（无额外续存费）。
5. 管理端说明为「只消耗配额、不另扣费」。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 续存任务帖（扣 1 配额） | TASK_POST_RENEWED | task-post-renewed | taskTaskService `handleRenewTask` | 站内通知 | — |
| 配额消耗审计（amount=0） | BILLING_TRANSACTION_CREATED | 既有 billing outbox | `consumeTaskPostRenewal` | 账本投影 | 与创建帖配额消耗同模式，非扣余额 |

## 实施计划

1. 续存改用 `task_post_quota` 单位；去掉 `getUnitPriceCents` / `server_start_renewal` 标价。
2. 公开 API 续存价固定 0；管理端文案更正。
3. `041` 将存量 `server_start_renewal.price` 置 0。

## 变更记录

- 2026-08-16：新建。续存只扣任务帖配额，不进行其他扣费。
