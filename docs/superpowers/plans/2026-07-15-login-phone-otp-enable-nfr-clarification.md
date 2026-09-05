# NFR：登录页电话验证码登录可用

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-15-login-phone-otp-enable-design.md`  
**默认级别**: L2；认证相关抬升至 L3

| 质量属性 | 级别 | 场景 | 验收 |
|----------|------|------|------|
| 安全性 | L3 | 开放重定向、OTP 暴力、token 泄露 | `PostLoginReturnUrl` 白名单；既有发码限流/OTP 校验；日志脱敏 |
| 可用性 | L2 | 策略开后入口可见；SMS 失败可感知 | UI 展示 + 错误弹窗 |
| 可运维性 | L2 | 管理员可关手机登录 | SystemAdmin 开关 |
| 性能 | L2 | 登录页多一次 public policy GET（存量） | 无新增重请求 |
| 兼容性 | L2 | OIDC `next` 行为不变 | OIDC 优先于业务 next |

## 领域模型影响

无新聚合；仅复用 SystemFeaturePolicy、VerificationCode、AccountSlot。
