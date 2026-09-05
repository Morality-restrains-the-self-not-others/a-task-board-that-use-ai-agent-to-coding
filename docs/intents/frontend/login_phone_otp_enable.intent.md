# 功能意图：登录页电话号码验证码登录可用

## 用户故事

作为访客或「添加账号」用户，我在  
`/auth/login/?add_account=1&next=/tenant/{id}/billing/recharge/`  
希望能用**手机号 + 短信验证码**登录，并在成功后回到充值页且保留本机其它账号槽。

## 验收标准

1. 系统策略 `enable_phone_login=true` 时，登录页展示「手机号/验证码」入口；为 false 时不展示且后端拒绝手机登录。
2. 部署后全局策略通过 data migration 打开为 true（管理员仍可在 SystemAdmin 关闭）。
3. 查询参数 `next` 为同站相对 path（如 `/tenant/.../billing/recharge/`）时，登录成功后跳转到该 path。
4. OIDC authorize 类 `next` 行为保持不变且优先于业务 path。
5. `add_account=1` 时登录成功 upsert `savedAccounts`，不清除其它账号槽。
6. 不新增公网 HTTP 接口；复用既有 OTP / policy API。

## 范围

- Django `SystemFeaturePolicy` data migration
- `Login.vue` 回跳与多账号槽
- 意图 / value-stream 文档
- 不含 OTP 迁 Go、不含强制开启充值前短信验证

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 用户手机验证码登录成功 | UserLoggedInViaPhoneOtp（存量路径） | forward-login / LoginSerializer | 审计等存量 | 证据豁免：本期仅开启策略与前端入口，沿用存量 forward-login，不改 MQ 契约 |
| 策略迁移打开手机登录 | — | — | — | 部署配置迁移 |
| 登录回跳 / 账号槽 upsert | — | — | — | 纯前端 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：策略开启 + next/add_account 闭环 |
| 2026-07-15 | 澄清：Navbar 从租户页「添加账号」须遵循多账号 AC7，不得把 `/tenant/...` 写入 next；仅非隔离业务 deep-link 可保留充值 path |
