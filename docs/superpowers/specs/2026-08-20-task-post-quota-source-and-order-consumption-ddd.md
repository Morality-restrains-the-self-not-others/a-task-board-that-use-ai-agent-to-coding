# DDD — 任务帖配额来源与订单消耗归属

- **Date:** 2026-08-20

## 限界上下文

**计费（taskBill）**：账户配额、资源批次、资源订单、交易流水。taskFE 为展示适配器。

## 聚合

| 聚合根 | 不变式 |
|--------|--------|
| BillingAccount | `task_post_quota >= 0`；合计 = 未过期批次 remaining 之和（回填完成后）；允许回填前 purchased 用差值 |
| ResourceGrant（批次，从属于账户） | remaining ∈ [0, quantity]；purchase 必须有 order_id |
| ResourceOrder | paid 后任务帖行项对应至多一条 purchase 批次 |
| BillingTransaction | 配额消耗可带 source_grant_id；related_order_id 与批次 order_id 一致 |

## 值对象

- `SourceKind` = gift | purchase
- `QuotaSplit` = {total, gifted, purchased}

## 领域事件

既有 `BILLING_TRANSACTION_CREATED`。消耗成功后 payload 增加 `source_kind`、`order_id`、`source_grant_id`。不新增 topic。

## 端口

- 查询：QuotaQuery、OrderQuery（扩展 DTO）
- 命令：ConsumeTaskPostQuota、MarkOrderPaid（内部增发批次）

无新服务。
