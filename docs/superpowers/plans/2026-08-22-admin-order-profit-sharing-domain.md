# DDD：管理员订单分账只读

- **日期**: 2026-08-22
- **NFR**: `docs/superpowers/plans/2026-08-22-admin-order-profit-sharing-nfr-clarification.md`

## Bounded Context

Billing（taskBill）。taskFE 为只读适配器。不新增 context。

## 读模型

- **AdminOrderDetail**：`ResourceOrder` + items + consumption + `[]ProfitSharingSnapshot`
- **ProfitSharingSnapshot**：`ReceiverUserID`、`AmountYuanCents`、`Status`、时间与失败原因。不含 OpenID。

## 聚合

写聚合仍是既有 `ProfitSharingRecord`（支付完成时创建）。本增量不新增聚合、不发新领域事件。

## 端口

- `ListProfitSharingByOrderID(orderID) []ProfitSharingSnapshot` — 基础设施查询 `billing_profit_sharing`
- 管理员 HTTP 适配器组装 `AdminOrderDetail`；租户 HTTP 适配器继续 `orderJSON` 且不得调用该端口的 JSON 附加

## 事件

无新事件。支付完成已标记分账。

## 分片

管理员点查 `order_id`；租户读带 `tenant_id`。快照按 `order_id` 索引。
