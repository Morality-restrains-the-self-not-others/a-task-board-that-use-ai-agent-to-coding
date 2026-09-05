# 邮件邀请退订

- **状态:** active
- **日期:** 2026-08-29
- **服务 Owner:** taskAuth（`auth_email_unsubscription`）
- **协作:** taskTenantService（邀请跳过 SMTP）、taskEvents（投递纵深 + 模板）、taskFE（确认页 + 复制链接提示）
- **网关:** taskGateway → taskAuth public/internal

## 业务意图

1. 用户在邀请邮件中点击「退订邮件邀请」，该邮箱进入退订列表。
2. 任意用户再对该邮箱发邮件邀请时，系统不发信，提示邀请人手动复制邀请链接。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ 类型/契约 | 发布点 | 消费者 | 例外理由 |
|---------|--------|-------------|--------|--------|---------|
| 点击退订成功 | EMAIL_UNSUBSCRIBED | Kafka `email-unsubscribed` | taskAuth public unsubscribe | taskEvents 结构化日志 | — |
| 对已退订邮箱创建邀请 | INVITATION_CREATED（`email_skipped=true`） | 既有 `invitation-created` | taskTenantService | 跳过 SMTP | 邀请记录仍创建 |
| 超管对已退订邮箱邀请 | 不发 EMAIL_SENT | — | taskAuth | — | 同步跳过，无邮件事件 |

## 非目标

- 不拦截验证码、密码重置、激活邮件
- 本期不做重新订阅、不做营销订阅中心

## 验收要点

- 邀请邮件含退订按钮；one-click POST 有效
- 退订后再次邮件邀请：`email_skipped` + 可复制 URL，无 SMTP
- 链接邀请 / 电话邀请不受影响
