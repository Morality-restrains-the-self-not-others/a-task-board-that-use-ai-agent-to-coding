# 推荐人分账看板按渠道聚合（隐藏受推荐人订单号）

日期：2026-08-24  
状态：accepted（/goal 跳过 Step 1 用户闸门）

## Context

`/profile/referral/`「订单分账」表把 `billing_profit_sharing.order_number`（如 `ORD-20260820-{tenant}-{orderId}`）直接展示给推荐人。订单号可反推受推荐人消费，属于多余暴露。推荐人真正需要的是：**某个渠道在什么时间段产生了多少订单金额、多少仍冻结、多少可分账**。

## Decision

1. `GET /api/billing/profit-sharing/referrer-orders/` 改为返回 `channels[]`，按 `billing_referral_edge.channel_code` 聚合；`period_from`/`period_to` 为该渠道订单 `paid_at` 最小/最大。
2. 响应**禁止**包含 `order_id`、`order_number`、`referred_user_id`、openid、微信分账单号。
3. 金额口径：`order_amount_yuan_cents` = Σ 订单总额；`frozen_amount_yuan_cents` = Σ 冻结态佣金；`shareable_amount_yuan_cents` = Σ 可分账佣金。`shareable` = 可分账佣金 > 0。
4. 新写路径 `POST /api/billing/profit-sharing/referrer-orders/share-channel/`，body `{ channel_code }`：对当前用户该渠道下所有 `shareable` 行循环 `executeProfitSharing`（微信仍按单出站）。逐行状态机幂等。
5. 保留既有 `POST .../{id}/share/`（不在列表暴露 id；管理/回归仍可用）。
6. 前端表格列：渠道号、订单时段、订单金额、冻结金额、可分账金额、操作。确认框不得出现订单号。
7. 无架构边界变更（仍 taskBill + taskFE，无新表/无新 MQ）→ **不升 ArchiMate、无新 ADR**。`No-ADR: covered by existing referrer-manual-profit-sharing design; privacy-only contract change`

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 仅前端打码订单号 | API 仍泄漏，DevTools 可见 |
| 按日/月再切片 | 用户要的是渠道 + 起止时间，一行一渠道即可 |
| 删除手动分账 | 15–30 天窗口仍需推荐人触发 |

## Consequences

- 旧 SPA 若仍读 `orders`，表格变空（失败封闭，优于继续漏单号）；须同步构建 taskFE。
- `user_id=0` 的历史单归入空渠道码（前端「未标注渠道」）。
