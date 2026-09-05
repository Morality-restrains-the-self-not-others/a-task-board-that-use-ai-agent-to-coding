# 意图：OTP/SMS 全切 Go（原生 SDK）

**stem**: `otp_go_full_native_sms`  
**日期**: 2026-07-15

## 验收标准

1. 配置 `SMS_PROVIDER=mock` 时发码成功且不调用 Django dispatch-sms。
2. `SMS_PROVIDER=aliyun` 且缺密钥时发码失败（不静默成功）。
3. 密码重置手机码发/验不经 Django send/verify-password-reset-code（手机路径）。
4. phone+code 登录 HTTP 路径不调用 forward-login。
5. 充值发码仍可用（底层 Go 原生 SMS）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 发手机验证码 | VerificationCodeSent | VERIFICATION_CODE_SENT | taskAuth sendPhoneVerificationCode | 审计 | 证据豁免：结构化日志 |
| 验码成功 | VerificationCodeVerified | VERIFICATION_CODE_VERIFIED | verifyPhoneVerificationCode | 审计 | 证据豁免：结构化日志 |
| OTP 自动注册 | UserCreated | USER_CREATED | enrich-login 自愈 | 下游开户 | — |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：全切 Go 原生 SMS 验收与事件对照 |
