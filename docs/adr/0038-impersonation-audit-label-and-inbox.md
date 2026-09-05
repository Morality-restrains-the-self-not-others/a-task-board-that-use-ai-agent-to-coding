# ADR-0038: 模拟登录日志标识、必填理由与用户收信箱

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** goal-mode auto-adopt

---

## Context

ADR-0037 已实现独立模拟会话与 `X-Impersonator-Id`，但：

1. HTTP 访问日志与 slog 仍可能只出现被模拟用户，排障无法区分真人操作者。
2. 开始模拟不要求理由，被模拟用户事后无法得知为何被代登。
3. 平台没有用户收信箱，无法把模拟通知送达本人。

## Decision

We will:

1. 制定元规则第 55 条：模拟会话下所有日志/审计必须含 `impersonating`、`impersonator_user_id`、`impersonated_user_id`、`impersonation_session_id`。
2. forward-auth 额外注入 `X-Impersonation-Session-Id`（及 `X-Impersonating: 1`）；APISIX `upstream_headers` 放行。
3. `shareLib/tracelog` 从上述头写入 request context，并输出到每条 `http_request` 与同 ctx slog。
4. `POST .../impersonate/` 请求体必须含 `reason`（trim 后 8～500 个 Unicode 字符），写入 `auth_impersonation_session.reason`。
5. 在 **taskAuth** 新增 `auth_user_inbox_message`（表前缀 `auth_`）；模拟成功后给被模拟用户插入一封 `impersonation_notice`，幂等键为 `impersonation_session_id`。
6. 登录用户仅能读取自己的收信箱（防 IDOR）；收信箱不是租户 region。
7. 发布 `UserInboxMessageCreated`（topic `user-inbox-message-created` + `-dlt`）；本期无自动消费者。
8. 前端在点击「以该用户身份登录」时先弹窗填写理由，再发请求。

## Alternatives Considered

### Alternative 1: 新建独立 inbox 微服务

- **Pros:** 可被多服务复用
- **Cons:** 违反本期范围与 Go 服务优先的「扩展现有」；信件与身份同库更易事务
- **Why rejected:** 年增量小，taskAuth 已是身份权威

### Alternative 2: 仅邮件通知

- **Pros:** 无需新表
- **Cons:** 用户可能未绑邮箱；模拟排障常发生在登录后
- **Why rejected:** 收信箱是站内可审计记录

### Alternative 3: 日志里打完整理由

- **Pros:** 排障方便
- **Cons:** 理由可能含工单外 PII
- **Why rejected:** 理由落库收信箱/会话表；日志只打 ID

## Consequences

### Positive

- 审计可区分操作者与被模拟用户
- 被模拟用户能看到代登理由
- 元规则约束后续服务不得遗漏标识

### Negative / Trade-offs

- 开始模拟多一次写收信箱；须与会话同事务或幂等
- 存量模拟会话无 reason（列默认空）

### Mitigations

- `UNIQUE(impersonation_session_id)` 防重复写信
- 热表即可；归档记 OPT

## References

- [ADR-0037](0037-admin-user-impersonation.md)
- `.ai/01_project_constraints/60_impersonation_audit_label.md`
