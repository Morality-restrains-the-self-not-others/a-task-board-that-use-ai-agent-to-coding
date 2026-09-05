# DDD：租户订单交易单号 / 商户单号查询

- **日期**: 2026-08-29
- **限界上下文**: Billing（taskBill）+ 租户控制台（taskFE）

## 聚合

既有 `ResourceOrder`（`billing_resource_order`）。本增量不新增实体。

凭证字段（已有）：

- `wechat_transaction_id` — 微信支付交易单号（用户口中的「交易单号」）
- `out_trade_no` — 商户订单号
- `order_number` / `id` / `payment_ref` — 既有统一查询 OR 集

## 仓储

既有 `listOrdersByTradeNo(tenantID, q, status, limit, offset)`：`tenantID > 0` 时强制本租户。不新增端口。

## 领域事件

无。纯查询书面例外。

## 应用服务

`handleListOrders` 已读 `order_number`。前端补齐调用。
