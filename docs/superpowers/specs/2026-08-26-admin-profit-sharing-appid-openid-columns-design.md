# 设计：管理端分账表展示 AppID / OpenID

- **日期**: 2026-08-26
- **入口**: `/system-admin/users/` 推荐绩效抽屉「微信分账」Tab（主需求）；同 API 的「待分账」队列一并展示
- **架构变更**: 无（不新增服务/聚合/消息；不更新 `docs/architecture/`）
- **python_api_approval**: 未触发（扩展既有 Go `taskBill` 列表 JSON）
- **总体设计审批**: approved（goal-mode 自动采用）

## 背景

运营在用户页微信分账表对照「appid 与 openid 不匹配」失败时，看不到实际使用的 AppID / OpenID，只能猜。此前 OPT-20260822-024 禁止管理端 JSON 下发 openid，以便推荐人自助页与展示名解析不泄漏 PII。超管对账场景需要反过来：**平台员工列表必须可见**。

## 方案

1. `GET /api/system-admin/profit-sharing/` 每项增加 `app_id`、`openid`（始终出键；空串表示未绑定/未快照）。
2. 取值（必须成对，禁止混源）：
   - 有接收方登记时：`app_id`/`openid` 均取 `billing_profit_sharing_receiver`（taskAuth `wechat_identity` 支付/mp 行的计费投影）。
   - 无接收方 openid 时：`openid` 才回退台账 `referrer_openid`；`app_id` 仍只来自接收方（空则 `""`）。
   - **禁止** `receiver.appid` + 台账快照 openid 拼一对（现网「appid与openid不匹配」即此：支付 AppID `wx31273ca77c89dffe` 配了网站应用 openid）。
3. LEFT JOIN 接收方表（PK=`referrer_user_id`，不改变行数）。
4. 出站分账 `resolveProfitSharingReceiverOpenid` 同样优先已登记接收方，覆盖过期快照后再查 taskAuth。
5. 前端列头 **AppID** / **OpenID**，空值「—」。落点：`ReferralWechatProfitSharingTab`（用户点名表）与 `SystemAdminProfitSharingPanel`（同一 API）。
6. **不改**：推荐人自助 `GET /api/billing/profit-sharing/referrer-orders/`、订单详情 `profit_sharing[]`、展示名 enrich。

## 非目标

- 不新增 API、不改分账执行、不把字段下发给租户/推荐人本人
- 不在日志打印 openid 明文

## 业务意图 → 事件对照

纯查询字段扩展，无新事件。
