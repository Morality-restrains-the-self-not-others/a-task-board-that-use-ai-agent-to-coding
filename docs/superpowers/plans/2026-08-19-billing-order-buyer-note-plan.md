# 实施计划 — 下单施工留言

- **Date:** 2026-08-19
- **DDD:** `docs/superpowers/specs/2026-08-19-billing-order-buyer-note-ddd.md`

## 事件契约

无新事件（见 DDD 例外）。意图对照：`docs/intents/billing-order-buyer-note.intent.md`。

## 任务

- [x] **T1** 红：`taskBill/src/orders_buyer_note_test.go` — 磁盘可写留言；仅任务帖非空留言失败；超长失败；空留言成功
- [x] **T2** 绿：DDL `049_order_buyer_note.sql` + `createOrderWithNote` + `orderJSON.buyer_note` + handler 解析
- [x] **T3** FE：OrderCreate 磁盘时 textarea；POST `buyer_note`；OrderDetail / 超管展开展示
- [x] **T4** OpenAPI POST body 登记 `buyer_note`；意图/价值流测试点
- [x] **T5** `gofmt` / 相关 `go test` / vitest；登记精准重启 task-bill + taskFE
