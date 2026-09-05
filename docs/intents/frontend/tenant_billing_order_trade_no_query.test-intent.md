# 测试意图：租户订单列表按交易单号 / 商户单号查询

## 测试目标

覆盖 VS-TO-1～4：交易单号、商户单号、空查询、越权。

## 测试分层

- 单元：`BillingOrderNumberSearch.test.js`、`BillingOrders.tradeNoSearch.test.js`
- 后端：`orders_list_trade_no_test.go` 租户 GET 命中与跨租户 0 条

## 用例矩阵

| ID | 步骤 | 期望 |
|----|------|------|
| VS-TO-1 | 填交易单号点查询 | URL 含 `order_number=` 微信支付单号；后端命中 `wechat_transaction_id` |
| VS-TO-2 | 填商户单号点查询 | URL 含商户单号；后端命中 `out_trade_no` |
| VS-TO-3 | 不填或清空 | 无 `order_number` |
| VS-TO-4 | 他租户同号 | `total=0` |

## 数据与环境

vitest jsdom；Go MySQL test DB。

## 通过标准

上述测试全绿。无 MQ 断言（纯查询例外）。
