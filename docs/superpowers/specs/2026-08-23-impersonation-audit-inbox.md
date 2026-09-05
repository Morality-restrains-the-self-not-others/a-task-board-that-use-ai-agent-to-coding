# 设计：模拟登录审计标识 + 用户收信箱

- **日期**: 2026-08-23
- **状态**: accepted（goal-mode 自动采用）
- **ADR**: ADR-0038（扩展 ADR-0037）
- **架构**: v103 application-integration / enterprise-landscape

## 问题

系统管理员以用户身份登录后，日志可能只看到被模拟用户；被模拟用户也无法得知代登及理由。

## 方案（已采用）

1. **元规则 55**：所有日志/审计标识 impersonator + impersonated + session。
2. **网关头**：`X-Impersonator-Id`、`X-Impersonation-Session-Id`、`X-Impersonating: 1`。
3. **tracelog**：头 → ctx → `http_request` / slog。
4. **必填理由**：8～500 字，写入会话表。
5. **收信箱**：taskAuth 表 `auth_user_inbox_message`；成功后一封 `impersonation_notice`。
6. **前端**：就地 overlay 弹窗填理由（禁止 Teleport）；账号中心「收信箱」页。

## API

| 方法 | 路径 | 权限 |
|------|------|------|
| POST | `/api/system-admin/users/{id}/impersonate/` | `user:impersonate`；body `{reason}` |
| GET | `/api/auth/inbox/` | 登录用户，仅本人信件 |
| PATCH | `/api/auth/inbox/{id}/read/` | 登录用户，仅本人信件 |

## 事件

| 意图 | 事件 | topic | 消费者 |
|------|------|-------|--------|
| 开始模拟 | UserImpersonationStarted | user-impersonation-started | 人工审计 |
| 结束模拟 | UserImpersonationStopped | user-impersonation-stopped | 人工审计 |
| 收信箱新信 | UserInboxMessageCreated | user-inbox-message-created | 本期无自动消费者 |

## 冷热与分片

- inbox 时间累积型，年增量预估 < 10 万：单表 + `created_at` 索引；主键 Snowflake。
- 分片键审视：路径带 `userId`（收件人），适合按用户分片，本期单表。
