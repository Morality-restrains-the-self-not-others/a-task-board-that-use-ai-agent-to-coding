# DDD — 按交易单号查询订单

- **日期**: 2026-08-21
- **NFR**: `docs/superpowers/plans/2026-08-21-system-admin-order-trade-no-query-nfr-clarification.md`

## 限界上下文

`Billing` / `ResourceOrder`（taskBill）。无新上下文。

## 模型

- **聚合**: `ResourceOrder`（已有）。本增量只读。
- **值对象**: `TradeNoQuery` — 规范化后的查询串（trim、≤128）。
- **查询端口**: `listOrdersByTradeNo(tenantID, query, status, limit, offset)` → `[]ResourceOrder, total`。
- **匹配规则**: `order_number = q OR payment_ref = q OR (q 为合法 id 则 id = q)`。三段号不拆 `id`。

## 架构

端口-适配器：HTTP handler 解析 query → 查询函数 → JSON 列表（既有 `orders/total/limit/offset`）。

## 事件

无。纯查询书面例外，见意图文档。

## 幂等消费

不涉及 Kafka。
