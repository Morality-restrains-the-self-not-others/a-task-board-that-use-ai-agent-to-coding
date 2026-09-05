# 计划 — 订单号租户基因

- **Date:** 2026-08-19
- **DDD:** `docs/superpowers/specs/2026-08-19-order-id-tenant-shard-ddd.md`

- [x] T1 红：`generateOrderNumber(tenant, id)` 四段格式；非正 id/tenant 报错
- [x] T2 绿：实现生成 + `ParseResourceOrderNumber`
- [x] T3 `createOrder` 两租户号含各自 tenant 与 id
- [x] T4 `loadOrder(tid, oid)` 错租户无行；IDOR 仍 404
- [x] T5 更新 `generateOrderNumberFn` 注入签名与 duplicate 测
- [x] T6 租户面 handler / `markOrderPaid` / refund 走复合查询；`tenantID<=0` 回调用 `loadOrderByID` 再取行上 tenant
- [x] T7 无新事件（意图表已例外）
- [x] T8 gofmt / 针对性 `go test`
