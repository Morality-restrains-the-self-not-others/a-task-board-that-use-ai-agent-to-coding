# 功能意图：租户微信支付对所有订单打分账标识

## 意图

租户为资源订单发起微信支付时，Native 预下单经仓内 `sdk/wechatpay-go` **一律**设置 `settle_info.profit_sharing=true`，不论付款人是否有推荐人、推荐人资格是否 active。支付成功后：有支付时刻活跃资格的推荐人则落 `billing_profit_sharing`；否则调用解冻剩余资金，避免货款长期冻结。

取代 `billing_paytime_referral_profit_sharing_flag` 中「无推荐人/无资格则不打标」条款。资格现查仍约束佣金台账。

## 角色

- 租户付款人：下单支付，不感知分账标识
- 推荐人：仅当支付时刻有活跃资格时成为该单分账接收对象
- 平台：预下单打标；无接收方则解冻剩余资金

## 行为

1. `POST /api/tenant/{tid}/billing/orders/{orderId}/pay/` 且 `payment_method=wechat`：`wechatPrepay` live 在 `native.NativeApiService.Prepay` 前设置 `PrepayRequest.SettleInfo.ProfitSharing=true`。无边、资格 false、资格服务失败均**仍打标**。
2. 支付成功 `markOrderForProfitSharing`：支付时刻资格 active 才落佣金行；否则不落行。
3. 微信支付成功且无佣金行：`UnfreezeOrder`，`out_order_no=UF{order_id}`，同一单号重放等价。mock 跳过。
4. 禁止手写微信支付 HTTP 设置分账字段。

## 非目标

- 不为已支付且当时未打标的微信单补标
- 不改 PayPal
- 不把无推荐人订单写入佣金台账
- 不向租户 API 暴露分账字段（ADR-0030）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|------------|--------|--------------|---------|
| 预下单一律设置分账标识 | — | 出站微信 APIv3 | `wechatPrepay` | 微信冻结可分账资金 | 无新领域事件 |
| 无接收方解冻剩余 | — | 出站微信 APIv3 | `markOrderPaid` 异步 | 微信解冻 | 支付成功事件已覆盖 |
| 支付成功标记佣金分账 | — | 既有支付成功路径 | `markOrderForProfitSharing` | `billing_profit_sharing` | 同上 |

## 变更记录

- 2026-08-23：一律打标 + 无接收方解冻（ADR-0040）
- 2026-08-22：支付时刻资格 + Native `SettleInfo`（ADR-0033，打标条件已废）
