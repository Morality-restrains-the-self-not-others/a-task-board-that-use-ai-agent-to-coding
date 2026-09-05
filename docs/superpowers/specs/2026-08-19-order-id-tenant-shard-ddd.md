# DDD — 订单号租户基因

- **Date:** 2026-08-19
- **NFR:** `docs/superpowers/plans/2026-08-19-order-id-tenant-shard-nfr-clarification.md`

taskBill 为 `main` 包，不拆新 hex 目录。概念落在现有 `ResourceOrder`。

## Bounded context

Billing（taskBill）拥有 `billing_resource_order`。

## 模型

| 类型 | 名称 | 职责 |
|------|------|------|
| Aggregate | ResourceOrder | 根 `id`（订单 Snowflake）；归属 `tenant_id` |
| VO | OrderNumber | `ORD-{utcDate}-{tenantId}-{orderId}` |
| Domain service | ParseResourceOrderNumber | 四段/三段分流；三段无 tenant |
| Port | 加载 | `loadOrder(tenantID, orderID)`；`loadOrderByID` 仅管理/回调 |

## 不变量

1. 新单 `order_number` 第 4 段 == `id`，第 3 段 == `tenant_id`。
2. 租户面读必须同时谓词 tenant + id。
3. 64-bit `id` 不含租户位。

## 业务意图 → 事件

书面例外：无新 MQ 事件（只改展示号派生与只读加载）。下单仍用既有 `resource_order_created` 日志。
