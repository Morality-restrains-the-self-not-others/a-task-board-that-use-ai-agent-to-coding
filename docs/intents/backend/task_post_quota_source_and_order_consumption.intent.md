# 功能意图：任务帖配额来源拆分与订单消耗归属

## 背景与目标

租户在账单页看到的任务帖配额是单一合计，无法区分赠送与购买；消耗配额时无法在订单上看到「用的是哪一笔订单的资源」。目标：账户剩余按来源拆分；消耗写入批次并归属订单。

## 范围与边界

- 范围内：taskBill 配额 GET、订单 GET、购买发放批次、创建/续存消耗 FEFO、退款清批次 remaining、taskFE 账单卡与订单展开/详情。
- 范围内另含：订单 GET `resource_consumption` 须包含本单已支付的 GitLab 磁盘/流量（granted=行项数量；消耗按区域配额 LIFO 分摊）。
- 范围外：GitLab 磁盘/流量赠送/购买账户级拆分展示；改价；新 HTTP 路径。

## 约束与风险

- 单库单表仍属 taskBill；`tenant_id` 为查询分片键。
- GET 上幂等回填购买批次（禁止启动 DDL）。
- 赠送优先于购买，其次 FEFO。

## 验收标准

1. 配额 GET 含 `task_post_quota` / `task_post_quota_gifted` / `task_post_quota_purchased`，三者关系成立。
2. 支付任务帖后产生 `source_kind=purchase` 且带 `order_id` 的 grant。
3. 消耗后流水带 `source_grant_id` 与 `related_order_id`；订单 GET `resource_consumption.task_post` 数量与事件一致。
4. 前端账单卡可见「赠送/购买」；订单展开可见消耗区块。

## 实施计划

见 `docs/superpowers/plans/2026-08-20-task-post-quota-source-and-order-consumption-plan.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | MQ类型/契约 | 例外理由 |
|---------|----------------|--------|--------------|------------|---------|
| 查询配额拆分 | — | — | — | — | 纯查询 |
| 查询订单消耗 | — | — | — | — | 纯查询 |
| 消耗任务帖配额（创建/续存） | BillingTransactionCreated | taskBill outbox | 既有 billing 消费者 | BILLING_TRANSACTION_CREATED | 扩展 payload |
