# DDD — 资源订单号全局唯一

- **Date:** 2026-08-19
- **NFR:** `docs/superpowers/plans/2026-08-19-billing-order-number-uniqueness-nfr-clarification.md`

## 限界上下文

`taskBill` / Resource Billing。不新增上下文。

## 聚合

`ResourceOrder`（既有）：`id`（Snowflake 身份）、`tenantId`、`orderNumber`（展示）、`status`、行项。

## 值对象

`OrderNumber`：`ORD-{yyyyMMdd}-{tenantId}-{orderId}`（ADR-0018）。不变量：

- `tenantId > 0` 且 `orderId > 0`
- 全局唯一（由 orderId 保证）
- 展示号内的 tenant 段是路由提示，不是分片键；查询聚合仍用列 `tenantId` + `id`

## 领域服务

`generateOrderNumber(tenantId, orderId) → OrderNumber`：纯函数，无仓储。禁止读「当日最大序号」。

## 端口

无新端口。插入仍走既有订单仓储（`INSERT billing_resource_order`）。

## 领域事件

无新事件。创建成功路径已有结构化日志 `resource_order_created`（含 `order_id` / `order_number` / `tenant_id`）。

**无新 MQ 事件例外：** 本增量不改变下单业务意图，只改变展示号派生规则；不引入新的成功/失败业务状态。

## 幂等键与重复边界

与 NFR 一致：一次 POST = 一笔订单；UNIQUE(id) + UNIQUE(order_number)；禁止 tenant_id 作去重键。
