# 超管邮箱邀请过期扫描

- **状态:** active
- **日期:** 2026-08-29
- **服务 Owner:** taskAuth（`auth_email_registration_invite`）
- **协作:** taskEvents timer `email_invite_expiry_scan/1_expire`

## 业务意图

到期仍为 `pending` 的超管邮箱邀请被标为 `expired`。触发只允许外界一次性调用（taskEvents timer HTTP），禁止 taskAuth HTTP 进程内 sleep/ticker 环。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ 类型/契约 | 发布点 | 消费者 | 例外理由 |
|---------|--------|-------------|--------|--------|---------|
| 扫描并过期到期邀请 | EMAIL_INVITE_EXPIRY_SCAN | 无 Kafka（timer 叫醒） | taskEvents timer | 无 | 周期扫表不是领域事实；一次性 POST 幂等 UPDATE |

## 验收要点

- `POST /api/internal/taskauth/email-invites/expire-due/` 将到期 pending 标 expired
- `startEmailInviteCleanupLoop` 不在 taskAuth 启动路径
- timer 健康端口 = `conf/events/domain-events/email_invite_expiry_scan/config.yaml` 的 18071
