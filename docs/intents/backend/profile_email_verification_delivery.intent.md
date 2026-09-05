# Intent: 个人资料邮箱绑定验证码须真实投递

## 背景与目标

厂商门户前置绑定邮箱（`/profile/?sso_error=email_required#rg=profile.email_binding`）。用户看到「验证码已发送，请查收邮箱」，但收件箱为空。HTTP 200 仅表示 Kafka `EMAIL_SENT` 入队；`task-events-email-sent-1-send-email` 用空 `host_password`（未叠 conf-local）对 QQ SMTP AUTH 得到 535，重试至 DLT。目标：验证码路径同步 SMTP，成功才 200；邮件消费者叠 conf-local 密码。

## 范围与边界

- 范围内：taskAuth `publishEmailSent`（`verification_code` 同步 SMTP）、SMTP fallback 正文含 OTP、taskEvents `LoadSettings` 经 `ReadAppFragment` 叠 conf-local、消费者 `pre_delivered` 跳过二次 SMTP、taskFE 失败路径不得显示「已发送」。
- 范围外：不轮换 QQ 授权码（运维/conf-local）；不改 Kafka 主题名；不把邀请/激活邮件改为同步 SMTP。

## 约束与风险

- ADR-0054 / 元规则 58：机密只在 conf-local。
- OTP TTL 5 分钟，禁止「入队即成功」。
- 日志不得打印 SMTP 密码；可打 `password_configured=true/false`。
- 失败须 502 + 前端 `data-traceId`。

## 验收标准

1. `LoadSettings` 在 tracked `host_password: ""` 时采用 `conf-local/.../email.yaml` 的密码。
2. `verification_code` SMTP 失败时 `POST /api/accounts/users/send_verification_code/` 返回 502，前端不出现「验证码已发送，请查收邮箱」。
3. SMTP 成功后 Kafka 带 `pre_delivered=true`，消费者不再 SMTP。
4. fallback 正文含 6 位验证码。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 发送邮箱绑定验证码 | EMAIL_SENT | Kafka topic `email-sent`；OTP 同步 SMTP 后带 `pre_delivered` | taskAuth `publishEmailSent` | `email_sent/1_send_email` 跳过二次 SMTP | OTP 必须先 SMTP 才能诚实 200 |

## 变更记录

- 2026-09-01：Loki 证据 trace `959d757f-8034-428f-b31f-091b9ee1a2e9` / `7fdf1b90-aee4-4780-97b8-5488582f9e96`；QQ SMTP 535；实施 overlay + OTP 同步 SMTP。
