# 功能意图：推荐人查看冻结/可分账单并手动分账

## 意图

推荐人在推荐页看到自己作为接收方的分账订单：完成交易后 15 天内、且分账记录仍为 pending 时为冻结；满 15 天且未满 30 天可点分账；满 30 天提示已过期无法分账。分账记录已是 `failed` 时，即使仍在 15 天窗口内也展示为失败（不得显示冻结），15–30 天内仍可重试。系统 25 天兜底仍在用户未点时补申请。

## 角色

- 推荐人（已登录，`referrer_user_id` = 当前用户）：列表 + 手动分账
- 系统小时扫描：仅 25–30 天兜底，不再在 15 天到期时自动分账
- 被推荐人 / 其它租户成员：不可见、不可点

## 行为

1. `GET /api/billing/profit-sharing/referrer-orders/` 按渠道聚合返回 `channels`（渠道号、`period_from`/`period_to`、订单金额、冻结佣金、可分账佣金）；只含当前用户为推荐人的数据；**不含** `order_id`/`order_number`/openid。
2. `POST /api/billing/profit-sharing/referrer-orders/share-channel/`：对该渠道下所有 `shareable` 行出站分账；无本人行 404；无可分账 409。逐行 `out_profit_sharing_no` 幂等。
3. 存量 `POST /api/billing/profit-sharing/referrer-orders/{id}/share/`：仅 `shareable`；冻结/过期 409；已成功 200 空操作；列表不再返回 id。
4. 前端分账按钮同步门闩 + 同一次意图回传同一 `Idempotency-Key`。
5. 写路径鉴权：网关已验证 + `X-User-Id` 必须等于行上 `referrer_user_id`（否则 404）。

## 非目标

- 不把分账展示给买家订单页
- 不取消 25 天兜底扫描
- 不新增领域事件总线（出站微信 CreateOrder 例外）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|------------|--------|--------------|---------|
| 推荐人手动发起分账 | — | 出站微信 APIv3 CreateOrder | `handleReferrerShareProfitSharing` → `executeProfitSharing` | 微信冻结资金划转 | 无新领域事件：与扫描分账同一聚合写路径，成功只更新 `billing_profit_sharing.status` |

## 变更记录

- 2026-08-23：推荐页订单态 + 手动分账 API；due pending 停止自动执行
- 2026-08-24：列表改为渠道聚合，禁止回传受推荐人订单号；新增 share-channel
- 2026-08-24：`failed` 不得映射为冻结；冻结仅 `pending` + 未满 15 天。渠道冻结金额不含失败佣金。
