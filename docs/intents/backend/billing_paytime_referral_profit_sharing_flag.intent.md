# 功能意图：租户微信支付按支付时刻推荐资格设置分账标识

## 意图

租户为资源订单发起微信支付时，**分账标识**已改为订单一律打标（见 `billing_order_always_profit_sharing_flag.intent.md` / ADR-0040）。本文件只保留：**支付时刻推荐资格**仍决定是否落 `billing_profit_sharing` 佣金行。资格不以绑边快照为准。

## 角色

- 租户付款人：下单支付，不感知分账标识
- 推荐人：须在**该笔支付发生时**具备活跃资格（`referral_code` approved 且未过期/未取消）
- 平台：预下单打标；支付成功后落 `billing_profit_sharing`

## 行为

1. Native 预下单打标条件见 ADR-0040 / `billing_order_always_profit_sharing_flag.intent.md`（一律 `ProfitSharing=true`）。
2. 支付成功 `markOrderForProfitSharing`：支付时资格；边上 `commission_eligible=0` 但现查活跃 → **仍落分账行**；现查不活跃或无边 → 不落行。
3. 资格内部接口：`user_id` 空 → 400；无内部密钥且环境已配置密钥 → 403；无申请/pending/rejected/expired/revoked → `active=false`；approved 未过期 → `active=true`。
4. 禁止手写微信支付 HTTP 设置分账字段；必须走 `services/payments/native`。

## 非目标

- 不为已支付且当时未打标的微信单补 `profit_sharing`（微信不支持）
- 不改 PayPal 路径（无对等分账标识）
- 不改点数计提对 `commission_eligible` 快照的依赖（ADR-0025）
- 不把分账执行 HTTP 从 `wechatV3Post` 迁到 `services/profitsharing`（OPT 跟踪）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|------------|--------|--------------|---------|
| 预下单设置分账标识 | — | 出站微信 APIv3 | `wechatPrepay` | 微信侧冻结可分账资金 | 无新领域事件：请求字段；支付成功仍走既有 `PAYMENT_SUCCEEDED` |
| 支付成功标记分账记录 | — | 既有支付成功路径 | `markOrderForProfitSharing` | `billing_profit_sharing` | 写入副作用已由支付成功事件覆盖，本增量不新发 MQ |
| 现查推荐资格 | — | HTTP 内部 | taskBill → taskReferral | 只读 `referral_code` | 同步查询，无状态变更 |

## 变更记录

- 2026-08-23：Native 分账标识改为订单一律打标，见 `billing_order_always_profit_sharing_flag.intent.md`（ADR-0040）。本文件仅保留「支付时刻资格约束佣金台账」语义。
- 2026-08-22：新增支付时刻资格 + Native `SettleInfo`；修正绑边无资格导致后续订单无法分账（ADR-0033）
