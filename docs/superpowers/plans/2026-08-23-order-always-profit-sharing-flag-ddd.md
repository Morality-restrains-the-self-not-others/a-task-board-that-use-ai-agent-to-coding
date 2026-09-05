# 领域模型摘记：微信订单一律分账标识

- **日期**: 2026-08-23
- **BC**: Billing（taskBill `main` 包，不新建 `domain/` 目录——与现有支付/分账同构）

## Policy

| 名称 | 规则 |
|------|------|
| WechatNativeProfitSharingFlag | live Native 预下单恒为 true |
| ReferralCommissionLedger | 仅支付时刻资格 active 的推荐人可成为 `billing_profit_sharing` 接收方 |
| UnfreezeUnsplitRemainder | 已打标且无佣金行 → Unfreeze `UF{order_id}` |

## Events

无新增领域事件。支付成功沿用既有路径。

## Ports

- 已有：`native.NativeApiService.Prepay`、`profitsharing.OrdersApiService.UnfreezeOrder`（SDK 适配器，可注入 `*Call` 变量）
