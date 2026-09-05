# DDD 建模：移除手机号验证码登录

- 日期：2026-08-24
- 关联设计：2026-08-24-remove-phone-code-login-design.md

## 领域事件（书面例外）

**本任务为纯功能删除，不产生新的业务意图，无领域事件投递需求。**

既有事件链不变：
- `USER_CREATED`：phone_register 注册成功 → 既有 publish（auth_phone_register.go），不修改
- OTP 自动注册（`handlePhoneOTPLogin` 内 `createUserWithPhoneLogin` + USER_CREATED）随函数删除而消失 —— 这是**意图收缩**而非新事件：验证码登录不再创建用户

## 领域概念映射

| 领域概念 | 现状 | 目标态 |
|---|---|---|
| 登录方式（LoginMethod） | email/username/phone 密码 + phone OTP | email/username/phone 密码（OTP 移除） |
| 验证码（VerificationCode） | 登录/注册/绑手机/重置 四场景 | 注册/绑手机/重置 三场景 |
| 自动开通（OTP Auto-Register） | 验证码登录时隐式建号 | **移除**（注册必须显式 phone_register） |
| 密码重置（PasswordReset） | 邮箱链接 + 手机/邮箱验证码 | 不变 |

## 聚合边界
- `LoginMethod` 聚合：phone 密码登录路径的查询/校验函数（findLoginMethodByCanonicalPhone 等）全部保留
- `VerificationCode` 聚合：`verifyPhoneVerificationCode` 保留（注册/重置消费方不变）
- 删除对象：`handlePhoneOTPLogin`（handler 层，非领域对象）

## 结论
无新聚合、无新事件、无文档对照表变更（docs/intents/ 无新增意图）。
