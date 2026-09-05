# 价值流：推荐资格服务号动态 scene 码

- **Date:** 2026-08-26
- **Increment:** 单增量交付（动态码 + SCAN 对账 + 冲突提醒 + UI）

## 触发到价值

```
已登录用户打开 /profile/referral/ 且未绑服务号
  → POST follow-qr 得到带 temp_id 的动态码
  → 微信扫码：未关注 subscribe+qrscene_；已关注 SCAN
  → 平台按 temp_id 命中票，取 unionId
  → 空闲则绑定 mp；已属他人则 conflict，点「我已关注」看到提醒
  → 绑定成功后填写名称/简介申请资格
```

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| TP1 | 未绑定 | 有动态 QR，无申请表 |
| TP2 | 点「我已关注」仍 pending | 提示未确认，表单隐藏 |
| TP3 | 已绑定 | 展示申请表 |
| TP4 | SCAN 命中票且 unionId 空闲 | 票用户 mp 别名 |
| TP5 | scene 命中且 unionId 属他人 | 不抢绑；conflict 文案 |
| TP6 | 无 scene subscribe 未知 unionid | pending；不建 user |
| TP7 | apply 未绑定 | 400 `service_account_not_followed` |
| TP8 | 回调签名错误 | 403 |
| TP9 | 回调丢失但粉丝 qr_scene_str=temp_id | 点「我已关注」bound |

## 范围内 / 外

- 内：taskAuth 创码/回调/票表、taskFE 动态闸门、APISIX follow-qr。
- 外：过期票 timer 清理、取关撤销分账。

## fields（三元组）

- `task-auth.auth_wechat_mp_follow_ticket.id`
- `task-auth.auth_wechat_mp_follow_ticket.user_id`
- `task-auth.auth_wechat_mp_follow_ticket.status`
- `task-auth.wechat_identity.openid`（mp 别名，冲突时不改）
