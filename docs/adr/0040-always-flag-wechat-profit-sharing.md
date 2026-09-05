# ADR-0040: 租户微信订单一律打分账标识

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** /goal 租户下单时对所有订单打分账标识，无论是否有推荐人
- **Supersedes:** [ADR-0033](0033-paytime-referral-profit-sharing.md) 中「仅支付时推荐资格为真才设置 Native `SettleInfo.ProfitSharing`」条款；资格现查仍约束分账台账与打款。

---

## Context

微信支付 Native 下单的 `settle_info.profit_sharing` 只能在预下单时设置。官方说明：传入 `true` 后支付成功资金进入不可用余额，可通过请求分账或 [解冻剩余资金](https://pay.weixin.qq.com/doc/v3/merchant/4012526374.md) 处理，或支付成功 30 天后自动解冻；传入 `false`/不传则资金直接可用，**事后无法再分账**。文档：<https://pay.weixin.qq.com/doc/v3/merchant/4012791877.md>。

ADR-0033 为避免无资格订单冻结资金，仅在「有推荐边且支付时刻资格 active」时打标。结果是：无推荐人、资格查询失败、支付时尚未获资的订单永远无法分账。产品现要求：**租户下单时对所有订单打上分账标识，无论是否有推荐人。**

## Decision

We will:

1. 租户资源订单微信 Native 预下单（`wechatPrepay` live）**一律**设置 `native.PrepayRequest.SettleInfo.ProfitSharing = true`（仓内 `sdk/wechatpay-go` `services/payments/native`，禁止手搓 JSON）。不依赖推荐边、不依赖 taskReferral 资格查询。
2. `markOrderForProfitSharing` **仍**按支付时刻资格落 `billing_profit_sharing`：无推荐人 / 资格 inactive / 查询失败 → **不落佣金行**（`referrer_user_id` 非空约束；无接收方不能请求分账）。
3. 微信支付成功且本单无佣金行时，调用已有 `unfreezeProfitSharing`（SDK `OrdersApiService.UnfreezeOrder`），商户分账单号稳定为 `UF{order_id}`，避免全部货款冻结至 30 天。mock 模式跳过。有佣金行则仍走 8 天 delay + `CreateOrder(UnfreezeUnsplit=true)`。
4. 既有已支付、当时未打标的微信单仍不可补标（微信约束）。PayPal 无对等字段。

```
wechatPrepay (live)
  → 一律 SettleInfo.ProfitSharing=true
支付成功
  → 有资格推荐人：insert billing_profit_sharing（pending）
  → 否则：UnfreezeOrder(UF{order_id})
```

## Alternatives Considered

### Alternative 1: 维持 ADR-0033（有资格才打标）

- **Pros:** 无推荐人订单资金立即可用。
- **Cons:** 与「无论是否有推荐人都打标」产品要求冲突；资格查询 fail-closed 会漏标。
- **Why rejected:** 本需求明确要求全部订单打标。

### Alternative 2: 一律打标但不解冻无佣金单

- **Pros:** 实现最少；30 天后微信自动解冻。
- **Cons:** 无推荐人的全部微信货款最长冻结 30 天，现金流不可接受。
- **Why rejected:** 官方提供解冻剩余资金接口专用于「不需要分账的已打标订单」。

### Alternative 3: 无推荐人也插入 `billing_profit_sharing`

- **Pros:** 本地可枚举全部已打标订单。
- **Cons:** `referrer_user_id` NOT NULL；管理端会把空接收方当佣金记录；JOIN 推荐人图会污染。
- **Why rejected:** 微信侧标识与本地佣金台账分离；打标不等于有分账对象。

## Consequences

### Positive

- 所有新微信订单具备事后分账能力（平台/后续政策）。
- 预下单不再依赖 taskReferral 可达性，消除 fail-closed 漏标。
- 无佣金对象的订单通过解冻尽快释放商户可用余额。

### Negative / Trade-offs

- live 预下单每笔都冻结资金，直到分账或解冻成功。
- 解冻依赖支付回调已写入真实 `transaction_id`；失败则资金仍冻至微信自动解冻。
- ADR-0033 的预下单资格门禁作废，测例与意图需改写。

### Mitigations

- 解冻使用稳定 `out_order_no=UF{order_id}`（微信「同一分账单号多次请求等同一次」）。
- 结构化日志记录 `profit_sharing=true` / `unfreeze` 结果（tenant_id、order_id、out_trade_no；禁止密钥/openid）。
- 解冻失败 warn + 不回滚已支付订单。

## References

- Native 下单分账标识：https://pay.weixin.qq.com/doc/v3/merchant/4012791877.md
- 解冻剩余资金：https://pay.weixin.qq.com/doc/v3/merchant/4012526374.md
- [ADR-0032](0032-wechatpay-go-sdk.md) 仓内 wechatpay-go
- [ADR-0033](0033-paytime-referral-profit-sharing.md) 支付时刻资格（台账/打款仍适用）
- [ADR-0030](0030-admin-only-order-profit-sharing.md) 分账明细仅管理员可见
