# ADR-0033: 微信分账资格以支付时刻为准

- **Status:** superseded
- **Date:** 2026-08-22
- **Author:** cursor
- **Deciders:** /goal 租户下单按支付时推荐资格设置分账标识
- **Superseded by:** [ADR-0040](0040-always-flag-wechat-profit-sharing.md)（Native 分账标识改为订单一律打标；本 ADR 的支付时刻资格仍约束 `billing_profit_sharing` 落库与打款）

---

## Context

微信支付 APIv3 Native 预下单必须在请求里设置 `settle_info.profit_sharing=true`，事后无法给未打标交易做分账。既有实现：

1. `wechatPrepay` 走仓内 `sdk/wechatpay-go` 的 `native.Prepay`，但**未**设置 `SettleInfo`。
2. `findReferrerForOrder` / 点数计提以 `billing_referral_edge.commission_eligible` 为门禁；该字段是 **绑边时刻快照**（[ADR-0025](0025-referral-multi-channel-codes.md)），`ON DUPLICATE KEY` 不覆盖。
3. 推荐人先分享、后获审批时，边上永远是 `commission_eligible=0`，被推荐租户后续付款既打不上分账标，也落不了 `billing_profit_sharing`。

产品要求：资格检查以**支付时**推荐人当前状态为准。绑定时无资格、支付前已获资格的，该租户**新支付订单**仍应可分账。已支付且当时未打标的微信单无法补标（微信侧约束）。

资格真相在 taskReferral 的 `referral_code`（approved 且未过期/未取消）；边表归属 taskBill。跨服务禁止直连对方库（元规则 19）。

## Decision

We will:

1. 租户微信 Native 预下单时，按**付款用户**查 `billing_referral_edge`（有边即存在推荐人）。再经内部 HTTP 向 taskReferral 现查该推荐人是否**当前**有活跃资格。仅当现查为真时，用 `sdk/wechatpay-go` `native.PrepayRequest.SettleInfo.ProfitSharing = true`（禁止手搓 `/v3/pay/transactions/native`）。
2. 支付成功后的 `markOrderForProfitSharing` 使用同一套支付时资格判定，**不再**要求边上 `commission_eligible=1`。
3. taskReferral 提供 `GET /api/internal/referral/qualification/active/?user_id=`，内部密钥门禁，返回 `{user_id, active}`，实现复用 `referrerHasActiveQualification`。
4. 查询失败 **fail-closed**（不打标、不落分账行），避免未冻结资金却本地记分账。绑边快照仍用于渠道统计与**点数计提**（ADR-0025 其余条款不变）。

```
付款人 user_id
  → billing_referral_edge（有推荐人？）
  → GET taskReferral qualification/active（支付时刻）
  → true：Native Prepay SettleInfo.profit_sharing=true
  → 支付成功：markOrderForProfitSharing 同样现查
```

## Alternatives Considered

### Alternative 1: 凡有推荐边一律打标，执行分账时再查资格

- **Pros:** 后来获资也能覆盖「支付时尚未获资」的订单（微信资金已按分账单冻结）。
- **Cons:** 无资格推荐人的订单也会冻结待分账资金，需额外解冻；与「有资格才设置分账标识」不符。
- **Why rejected:** 产品明确要求有资格才打标。

### Alternative 2: 审批通过时把该推荐人全部边 `commission_eligible` 翻成 1

- **Pros:** 少一次跨服务调用。
- **Cons:** 资格过期/取消与边表易漂移；支付路径仍可能读到过期快照；与「支付时刻」语义弱于现查。
- **Why rejected:** 资格 SSOT 在 `referral_code`，支付时现查更准。取消资格仍走既有 disable-eligibility 停点数计提。

### Alternative 3: taskBill 直连 task_referral 读 `referral_code`

- **Pros:** 预下单少一跳 HTTP。
- **Cons:** 违反单表所有权。
- **Why rejected:** 元规则 19。

## Consequences

### Positive

- 先推荐后获资：被推荐租户之后的微信支付可打标并可分账。
- 支付时已取消/过期资格：不打标，避免错误冻结。
- 分账标识走官方 Native SDK，与 ADR-0032 一致。

### Negative / Trade-offs

- 预下单依赖 taskReferral 可达；失败则该笔无法分账（微信不可补标）。
- 支付当时无资格、支付完成后才获资的**已付款**微信单无法补分账。
- ~~点数计提仍看绑边快照~~已同步：`accrueReferralFromConsumption` 改为支付/消费时刻现查资格（OPT-20260822-059），与微信分账同口径。

### Mitigations

- 预下单失败与资格查询失败打结构化日志（含 tenant_id / referrer 指纹，禁止密钥）。
- 未补标历史单记 OPT，不假装能对微信补分账。
- 点数与微信分账已对齐（OPT-20260822-059），不在本 ADR 范围。

## References

- [ADR-0025](0025-referral-multi-channel-codes.md) 绑边渠道快照
- [ADR-0032](0032-wechatpay-go-sdk.md) 仓内 wechatpay-go
- 意图：`docs/intents/backend/billing_paytime_referral_profit_sharing_flag.intent.md`
