# 功能意图：申请推荐资格须已绑定服务号 openId

## 意图

`POST /api/accounts/users/referral-codes/apply/` 仅在该用户 `wechat_identity` 存在 `app_key=mp` 且 openid 非空时接受。status 返回 `service_account_bound`。

## 角色

- 已登录申请用户
- taskReferral

## 行为

1. 未绑定：apply 400 `service_account_not_followed`。
2. 已绑定：原 intro/legal_name/consent 校验不变。
3. status JSON 含布尔 `service_account_bound`。

## 非目标

- 在 taskReferral 内处理微信回调。

## 业务意图 → 事件对照

无新事件（申请提交仍走既有路径）。

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-26 | 初稿 |
