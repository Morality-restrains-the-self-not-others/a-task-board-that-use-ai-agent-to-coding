# 功能意图：OTP 迁 taskAuth 首切 + Navbar add_account AC7

## 用户故事

1. 作为多账号用户，在租户页点「添加账号」时，登录成功后不应回到原租户 URL（AC7）。
2. 作为平台，手机验证码的存储与校验归属 taskAuth（Go），符合单表所有权与 Go-first。

## 验收标准

1. `resolveAddAccountNext('/tenant/...')` → `/`；Navbar 省略业务 next。
2. `add_account=1` 且 next 仍含 `/tenant/` 时，Login 用新 userId 经 `resolveSwitchHref` 纠偏。
3. `accounts_sms_verification_code` owner 为 task-auth；auth.db 有迁移 `006_sms_verification_code.sql`。
4. `POST /api/accounts/users/send_verification_code/`（手机）在 taskAuth 写库；mock SMS 可发码。
5. `POST /api/internal/verification-code/verify/` 校验成功；Django `verify_phone_code_row` 调该接口。
6. 云 SMS（aliyun/tencent）经 Django `dispatch-sms` 临时桥接（不落库）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 发手机验证码 | VerificationCodeSent | taskAuth sendPhoneVerificationCode | — | 证据豁免：首切仅结构化日志 |
| 验手机验证码 | VerificationCodeVerified | taskAuth verifyPhoneVerificationCode | — | 证据豁免：首切仅结构化日志 |
| add_account AC7 | — | — | — | 纯前端 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 |
