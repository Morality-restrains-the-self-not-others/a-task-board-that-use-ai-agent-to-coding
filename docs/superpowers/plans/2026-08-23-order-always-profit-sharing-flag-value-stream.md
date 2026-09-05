# 价值流：租户下单一律打微信分账标识

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-order-always-profit-sharing-flag-design.md`

## Related Value Streams

修改既有「微信支付分账标识」流（ADR-0033 / VS-PS-4/5/6），不是新绿场流。

| 既有 | 本增量变化 |
|------|------------|
| VS-PS-4 有资格 → 打标 | 仍打标（恒 true） |
| VS-PS-5 无推荐人/无资格 → 不打标 | **改为仍打标**；无资格不落佣金行；无佣金行则解冻 |
| VS-PS-6 绑边无资格、支付时已获资仍打标并可落行 | 保持 |

## Increment

**单一增量**：live Native 预下单一律 `profit_sharing=true` + 支付成功分支（落台账 vs 解冻）。

步骤：

1. 租户对资源订单选择微信支付
2. taskBill `wechatPrepay` 向微信 Native 下单并带分账标识
3. 用户扫码支付成功
4a. 有资格推荐人 → 落 `billing_profit_sharing`，后续 timer 请求分账
4b. 否则 → `UnfreezeOrder(UF{order_id})`

测试点：T1–T10（见 `billing_order_always_profit_sharing_flag.test-intent.md`）。

## 字段

- 出站：`settle_info.profit_sharing`
- 出站：`profitsharing/orders/unfreeze` 的 `transaction_id`、`out_order_no`
- 读：`billing_referral_edge`、taskReferral qualification/active
- 写：`billing_profit_sharing`（有资格时）
