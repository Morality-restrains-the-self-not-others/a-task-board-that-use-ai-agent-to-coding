# 功能意图：推荐页冻结订单与可分账订单

## 意图

登录用户在 `/profile/referral/` 按**渠道**看分账：渠道号、该渠道订单起止时间、订单金额、冻结金额、可分账金额。完成后 15 天内且分账仍为 pending 的佣金计入冻结；已 `failed` 的佣金不计入冻结（与管理端「失败」一致）；满 15 天未满 30 天可点「分账」（按渠道批量，含失败重试）；满 30 天不再可分。不展示受推荐人订单号。

## 行为

- GET `/api/billing/profit-sharing/referrer-orders/`（Anti-Replay-OK: 只读）→ `channels`
- POST `.../referrer-orders/share-channel/`：`createClickGuard` + `Idempotency-Key`；`aria-busy`
- 错误展示 `data-traceId`
- 不展示 openid、订单号

## 业务意图 → 事件对照

前端不发布领域事件；写操作经 taskBill 出站微信 CreateOrder。
