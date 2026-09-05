# 功能意图：申请推荐资格前先关注服务号

## 意图

未获得推荐资格的用户在填写名称/简介前必须先关注服务号。闸门展示 **动态 scene 二维码**（`POST /api/auth/wechat/mp/follow-qr/` 返回的 `qr_src`），不再用静态 `jjf_qrcode.png` 作为关注凭证。用户扫码并由系统按临时 ID 对账后，再展示原申请表。若该微信已绑定其他账号，点「我已关注」展示服务端提醒。

## 角色

- 已登录用户

## 行为

1. `service_account_bound=false`：进入闸门时一次 POST follow-qr（同步门闩 + Idempotency-Key）；展示动态码与「请先关注服务号」；隐藏申请表。微信创码失败则错误节点带 `data-traceId`，禁止回退静态图。
2. 「我已关注」同步门闩，GET follow-status；`bound` 则展示表单；`ticket_status=conflict` 用服务端 `message`；`expired` 提示刷新取码；`has_unionid=false` 且非 conflict 时提示先绑定微信登录。
3. 「绑定微信登录」写入 path 限定 cookie 后整页跳转 `/api/auth/wechat/bind/?app=web&next=/profile/referral/`。
4. `service_account_bound=true`：直接展示 `ReferralQualificationApplyForm`。
5. 资格审批中/已获资格：不展示关注闸门。
6. 请求失败错误节点带 `data-traceId`。

## 非目标

- 后台轮询、自动跳转微信、代用户关注、把他人账号 id 展示给当前用户。

## 业务意图 → 事件对照

无前端直发事件。

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-26 | 初稿 |
| 2026-08-26 | 无 unionId 账号须先绑定微信登录才能锁定关注 |
| 2026-08-26 | 绑定成功后经 next 回到 `/profile/referral/` |
| 2026-08-26 | 静态码改为动态 scene 码；conflict 文案来自 follow-status |
