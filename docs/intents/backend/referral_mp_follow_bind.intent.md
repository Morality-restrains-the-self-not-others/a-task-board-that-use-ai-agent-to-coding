# 功能意图：服务号关注事件用临时 ID 对账并绑定 mp openId

## 意图

推荐页签发 Snowflake 临时 ID 动态码。微信关注（subscribe）或已关注扫码（SCAN）回调到达 taskAuth 后，先按 EventKey 匹配 `auth_wechat_mp_follow_ticket`，再取 unionId。unionId（或 mp openid）已属其他账号则不抢绑，把票记为 conflict；当前用户点「我已关注」看到提醒。未命中票的 subscribe 仍走 v112 pending。关注不建号。

## 角色

- 微信服务器（回调，public + 验签）
- 已登录用户（签发动态码、查询绑定状态；仅本人）

## 行为

1. POST `/api/auth/wechat/mp/follow-qr/`（token + Idempotency-Key）：复用未过期 pending 票，否则 Snowflake `temp_id` 调 `cgi-bin/qrcode/create`（QR_STR_SCENE）。
2. GET 回调验签成功回显 echostr。
3. POST 回调：明文 XML 或 Encrypt（`encodingAESKey`）；subscribe 或 SCAN；解析 scene（去掉 `qrscene_`）；命中票则按票的 `user_id` 绑定或 conflict。
4. unionId 空或等于票用户 → upsert `app_key=mp` + 票 `status=bound` + `WECHAT_MP_SUBSCRIBED outcome=bound`。
5. unionId/mp openid 已属他人 → 不改 `wechat_identity`；票 `status=conflict`；`WECHAT_IDENTITY_CONFLICT` + `WECHAT_MP_SUBSCRIBED outcome=conflict`。
6. 无 scene 的 subscribe：v112 pending 旁路。
7. GET follow-status：`bound` + `ticket_status` / `conflict_code` / `message`；禁止返回他人 user_id/unionId/openid。
8. 回调 POST 丢失时：follow-status 用未过期 pending 票的 `temp_id` 对账 `cgi-bin/user/info` 的 `qr_scene_str`（不是整表 unionId 匹配）；命中则走同一 bind/conflict 规则。

## 非目标

- 关注建号、前端轮询、取关删除分账接收方、静态 `jjf_qrcode.png` 作为闸门凭证。

## 业务意图 → 事件对照

| 意图 | 事件 | 键 |
|------|------|-----|
| 关注/扫码锁定成功 | WECHAT_MP_SUBSCRIBED outcome=bound | mp openid |
| 票冲突 | WECHAT_IDENTITY_CONFLICT | owner_user_id + openid |
| 票冲突产品结果 | WECHAT_MP_SUBSCRIBED outcome=conflict | temp_id |
| 无 scene 未命中 | WECHAT_MP_SUBSCRIBED outcome=pending | mp openid |
| 签发二维码 | 无 MQ | 例外：只写本聚合票据 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-26 | 初稿（静态码 + unionId 锁定） |
| 2026-08-26 | 改为动态 scene 票；SCAN；unionId 冲突按 temp_id 提醒 |
| 2026-08-26 | follow-status 用 `qr_scene_str` 补偿丢失的 subscribe/SCAN POST |
