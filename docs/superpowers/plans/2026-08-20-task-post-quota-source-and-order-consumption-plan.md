# 实施计划 — 任务帖配额来源与订单消耗归属

## 切片 1 — Schema + 配额拆分 API

- [ ] `dataMigrate/taskBill/051_task_post_grant_source_and_order.sql`
- [ ] `getTaskPostQuotaSplit` + `handleResourceQuotas` 字段
- [ ] 测试 T1/T2

验证：`go test ./src -count=1 -run 'TestResourceQuotasTaskPostGiftedPurchased'`

## 切片 2 — 支付建批次 + 消耗归属

- [ ] markOrderPaid 插入 purchase grant
- [ ] adminGrant 回写 grant.order_id
- [ ] consumeTaskPostQuotaTx 返回批次并写 source_grant_id/related_order_id
- [ ] 续存同路径
- [ ] 测试 T3–T5、T7、T9

验证：`go test ./src -count=1 -run 'TestTaskPostPurchaseLot|TestConsumeTaskPostQuotaPrefersGift|TestConsumeTaskPostQuotaPurchaseAfterGift'`

## 切片 3 — 订单消耗 DTO + 回填 + 退款

- [ ] orderJSON resource_consumption
- [ ] ensureTaskPostPurchaseLots
- [ ] 退款 remaining=0
- [ ] 测试 T6、T8

验证：`go test ./src -count=1 -run 'TestOrderJSONTaskPostConsumption|TestEnsureTaskPostPurchaseLots|TestRefundZerosPurchaseLot'`

## 切片 4 — 前端

- [ ] BillingDashboard 赠送/购买
- [ ] BillingOrders / OrderDetail 资源消耗
- [ ] 测试 F1–F3

验证：vitest 对应文件

## 切片 5 — OpenAPI / 意图图 / 价值流 YAML

- [ ] openapi.yaml 配额与订单响应说明
- [ ] conf/value-stream.yaml
- [ ] 架构 v91 四件套

## 事件任务

扩展 BILLING_TRANSACTION_CREATED payload；无新消费者。

## 可观测性

`task_post_lot_consumed`、`task_post_purchase_lot_issued`、`task_post_purchase_lots_backfilled`（含 tenant_id、order_id、grant_id，无密钥）。
