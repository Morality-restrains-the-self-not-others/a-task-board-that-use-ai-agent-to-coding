# 功能意图：超管按推荐人查询微信分账台账并核微信单笔

## 意图

系统管理员在推荐绩效抽屉按路径「推荐人 → 被推荐人 → 已打分账标的订单」列出订单，并对当前页调用微信支付 SDK `QueryOrder` 查询分账结果。微信无按 openid 列清单 API；本地「打标」等于存在 `billing_profit_sharing` 行。

## 角色

- 系统管理员 / 平台员工：跨租户读
- 租户成员：403

## 行为

1. `GET /api/system-admin/profit-sharing/?referrer_user_id={uid}&status=all`：网关已验证 + `IsPlatformStaff`；JOIN `billing_referral_edge` → 被推荐人订单 → `billing_profit_sharing`；分页同现网 `limit`/`offset`。
2. 不传 `referrer_user_id` 时全局队列行为不变。
3. 每项含 `referred_user_id`（付款人）、`fail_reason`、`fail_trace_id`（可空）。可附 `receiver_registration_status`。平台员工可见微信单号、`app_id`、`openid`（与全局队列同一取值：接收方成对，对齐 `wechat_identity` 支付/mp 行）。不含 `referrer_openid` 键。
4. `POST /api/system-admin/profit-sharing/refresh-wechat/`：body `{referrer_user_id, ids}`；ids 必须落在该 JOIN 结果内；逐笔 `QueryOrder`；不回写本地 status；不调用 CreateOrder。
5. 未登录 401；非平台员工 403；非法 status / 冒用他人 id / ids 超限 400。
6. 缺 transaction_id 或微信错误：该行返回 `wechat_error`，其余行继续。

## 非目标

- 不按付款人（被推荐人）过滤（本轮）
- 不下载日账单
- 不把 `referrer_openid` 键或推荐人自助接口的 openid 返回浏览器

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 按推荐人查被推荐人打标订单的分账结果 | 无对应事件 | — | — | 只读图查询 |
| 当前页 QueryOrder 核状态 | 无对应事件 | — | — | 出站只读，不改本库 |

## 变更记录

- 2026-08-23：推荐绩效抽屉按接收方查分账 + 可选微信核单
- 2026-08-26：列表项增加 `fail_trace_id`
- 2026-08-26：平台员工列表增加 `app_id` / `openid`（对账「appid 与 openid 不匹配」）
- 2026-08-26：成对取接收方登记，避免支付 AppID + web 登录 openid 混用
- 2026-08-26：取值改为接收方成对，禁止 `receiver.appid` + 台账快照 openid
