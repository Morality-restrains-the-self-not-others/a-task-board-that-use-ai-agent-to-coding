# 角色权限分析：otp-go-full-native-sms

**日期**: 2026-07-15  
**设计**: `2026-07-15-otp-go-full-native-sms-design.md`

| 改动点 | 角色 | 权限边界 |
|--------|------|----------|
| 公网 `send_verification_code` / `send_password_reset_code` | 匿名 | 仅发短信到请求号码；重置码须号码已注册 |
| 公网 `POST /api/auth/` phone+code | 匿名 | 验码成功才建号；条款经 enrich-login |
| 内部 `verification-code/verify` | 服务间 secret | 不变 |
| 充值 `recharge_*_sms` | 已登录租户成员 | 不变；OTP 底层 Go |
| Django `dispatch-sms` / `forward-login` | 内部 | Go 停用；保留端点防旧调用 |

无权限扩大；自动注册与首切 forward-login 行为一致。
