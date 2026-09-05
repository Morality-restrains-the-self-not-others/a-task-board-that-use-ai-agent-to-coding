# 计划：租户订单列表按交易单号 / 商户单号查询

- **日期**: 2026-08-29

## Task 1 — 后端契约回归（TDD）

- [ ] `taskBill/src/orders_list_trade_no_test.go`：租户 GET `order_number=` 命中 `wechat_transaction_id`；命中 `out_trade_no`；他租户同号 `total=0`。
- 命令：`cd taskBill && go test ./src -count=1 -run 'TestHandleListOrders'`

## Task 2 — 前端查询组件（TDD）

- [ ] 新增 `BillingOrderNumberSearch.vue`（交易单号 + 商户单号 + 查询/清空）。
- [ ] `BillingOrders.vue` 接入；`fetchOrders` 带 `order_number`。
- [ ] `BillingOrders.tradeNoSearch.test.js`：交易单号 / 商户单号 / 空查询。
- 命令：`cd taskFE/app && npx vitest run src/views/BillingOrders.tradeNoSearch.test.js`

## Task 3 — 契约与意图

- [ ] `taskBill/src/openapi.yaml` 更新 `order_number` 说明。
- [ ] `docs/intents/frontend/tenant_billing_order_trade_no_query.{intent,test-intent}.md`
- [ ] `docs/flows/value-stream-test-integration.wsd` 测试点。
- [ ] `conf/value-stream.yaml` 步骤。

## Task 4 — 事件

- [ ] 确认无 MQ（设计书面例外）。无需 publish 任务。
