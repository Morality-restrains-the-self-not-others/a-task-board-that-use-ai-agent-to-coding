# 测试意图：登录页电话号码验证码登录可用

## 对应功能意图

`docs/intents/frontend/login_phone_otp_enable.intent.md`

## 测试点

| # | 场景 | 类型 | 期望 |
|---|------|------|------|
| T1 | migration 后 global policy | Django | `enable_phone_login is True` |
| T2 | `next=/tenant/x/billing/recharge/` | Vitest | 解析为该 path |
| T3 | `next` 为 `//evil` 或绝对 URL | Vitest | 拒绝，回退其它策略 |
| T4 | OIDC authorize `next` | Vitest | 仍优先 OIDC 解析 |
| T5 | 登录成功 | Vitest（mock） | 调用 `persistLoginAccountSlot` |
| T6 | 策略 false | 存量 | 隐藏手机入口 / 后端拒绝 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 |
