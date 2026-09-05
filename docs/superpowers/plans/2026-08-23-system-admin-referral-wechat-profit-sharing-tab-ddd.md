# DDD — 推荐人图上的分账核对（只读）

- **日期**: 2026-08-23
- **NFR**: `docs/superpowers/plans/2026-08-23-system-admin-referral-wechat-profit-sharing-tab-nfr-clarification.md`

## 限界上下文

- **Billing (taskBill)**：拥有 `billing_profit_sharing`、`billing_resource_order`、`billing_referral_edge`、微信 SDK 端口。
- **Identity (taskAuth)**：openid 仅内部；本增量不新增公开查询。

## 模型

- **既有实体** `ProfitSharingRecord`（台账行：out_profit_sharing_no、order_id、referrer、status）。
- **值对象** `WechatProfitSharingQuery`：out_order_no + transaction_id → state（只读）。
- **领域服务（查询）** `ListFlaggedOrdersForReferrer(referrerID)`：边 → 买家订单 → 台账 INNER JOIN。
- **不新增聚合、不新增领域事件**。

## 端口

```
ProfitSharingQueryPort.QueryOrder(ctx, outOrderNo, transactionID) -> (state, err)
```

实现：既有 `queryProfitSharingOrder` / `profitSharingQueryOrderCall`。

## 业务意图 → 事件

纯查询例外，见意图文档。无 MQ 任务。
