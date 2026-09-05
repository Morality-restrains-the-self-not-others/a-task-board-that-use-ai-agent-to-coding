# 实施计划 — 资源订单号全局唯一

- **Date:** 2026-08-19
- **DDD:** `docs/superpowers/specs/2026-08-19-billing-order-number-uniqueness-ddd.md`

> 展示号格式已被 [ADR-0018](../../adr/0018-order-id-tenant-shard-routing.md) 修订为 `ORD-YYYYMMDD-{tenantId}-{id}`。本计划任务按当时三节号完成；四段号见 `2026-08-19-order-id-tenant-shard-plan.md`。

## 事件契约

无新事件（见 DDD 例外）。意图对照表：`docs/intents/backend/billing_resource_order_number.intent.md`。

## 任务

- [x] **T1** 红：`taskBill/src/orders_number_test.go` — 格式 `ORD-YYYYMMDD-{id}`、orderID<=0 失败、两租户创建号码不同且后缀=id
- [x] **T2** 绿：`generateOrderNumber(orderID int64)` 去掉 SELECT MAX；`createOrder` / `insertOrderWithRetry` 传入 orderID
- [x] **T3** 更新 `orders_duplicate_test.go` 注入函数签名；并发/1062 回归仍绿
- [x] **T4** FE：`BillingOrders.vue`、`SystemAdminOrderRecords.vue` 订单号 `break-all`（真 `<a href>` 不变）
- [x] **T5** 跑 `go test` 相关包；`gofmt`；意图文档

不改 DDL（VARCHAR(64) 足够）。不改支付 out_trade_no。
