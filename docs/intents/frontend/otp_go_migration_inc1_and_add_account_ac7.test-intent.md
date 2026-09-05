# 测试意图：OTP 迁 taskAuth 首切 + Navbar add_account AC7

## 对应功能意图

`otp_go_migration_inc1_and_add_account_ac7.intent.md`

## 测试点

| # | 场景 | 类型 | 期望 |
|---|------|------|------|
| T1 | resolveAddAccountNext 租户页 | Vitest | `/` |
| T2 | send+verify mock | Go | 发码后验码成功，二次失败 |
| T3 | internal verify 鉴权 | Go | 缺 secret → 403 |
| T4 | Django dispatch-sms | 手工/集成 | 仅发送不落库 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 |
