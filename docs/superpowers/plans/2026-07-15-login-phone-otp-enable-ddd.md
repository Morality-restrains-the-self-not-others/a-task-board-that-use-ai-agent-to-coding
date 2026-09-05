# DDD 摘要：登录页电话验证码登录可用

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-15-login-phone-otp-enable-design.md`

## Bounded Context

用户与认证（Auth）：taskAuth（入口/Token）+ Django accounts（OTP/SMS）+ Vue 会话槽。

## 既有模型（本期不新建聚合）

| 类型 | 名称 | 说明 |
|------|------|------|
| Entity | SystemFeaturePolicy | `enable_phone_login` |
| Entity | User / LoginMethod | OTP 登录后身份 |
| VO | PostLoginReturnUrl | 同站回跳 path |
| VO / Collection | AccountSlot / savedAccounts | 多账号槽 |
| Domain Service | VerificationCodeService | 发码/验码（存量） |

## 领域事件

本期不新增事件类型；OTP 登录成功沿用存量路径。纯前端回跳/槽 upsert：**无对应 MQ 事件**（见设计文档例外表）。

## 架构变更

无（不写 architecture target）。
