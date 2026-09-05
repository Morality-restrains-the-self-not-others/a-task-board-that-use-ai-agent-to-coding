# ADR-0037: 平台用户模拟登录（Impersonation）

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** goal-mode auto-adopt

---

## Context

平台运维需要以目标用户的角色查看其工作台与权限边界，而不能索要密码或长期持有该用户的登录 token。RBAC 已预留平台权限码 `user:impersonate`（super_admin / employee），但没有任何会话实现。

模拟登录是安全敏感路径：错误实现会导致权限提升、与用户共享持久 token、无法审计、无法退出。

## Decision

We will 在 **taskAuth** 实现独立模拟会话：

1. 开始模拟须持有平台权限 `user:impersonate`。
2. 签发**独立** `auth_impersonation_session.token_key`，不复用、不读取目标用户 `auth_customtoken`。
3. 管理员原 token 仅放入 HttpOnly `impersonatorRestore` cookie，退出时还原。
4. 禁止自模拟、嵌套模拟；模拟 `platform:manage` / 超管目标时操作者必须也有 `platform:manage`。
5. 会话 TTL 1 小时；开始/结束发布 `UserImpersonationStarted` / `UserImpersonationStopped`。
6. forward-auth 在解析到模拟 token 时注入 `X-User-Id=target` 与 `X-Impersonator-Id=actor`。

## Alternatives Considered

### Alternative 1: 直接切换到目标用户的 auth_customtoken

- **Pros:** 实现短
- **Cons:** 管理员拿到用户持久登录令牌；无法独立过期；污染 last_ip
- **Why rejected:** 凭证泄露面不可接受

### Alternative 2: 仅前端改 localStorage 假装登录

- **Pros:** 无后端
- **Cons:** 网关仍按管理员鉴权，UI 与 API 不一致；可被伪造
- **Why rejected:** 不能实现「以用户角色」访问

### Alternative 3: Django 实现

- **Pros:** 历史 Django 有类似 admin
- **Cons:** 违反 Go 优先；会话真源已在 taskAuth
- **Why rejected:** 元规则 20

## Consequences

### Positive

- 运维可复现用户视角；权限码终于有执行点
- 审计可区分 actor / target

### Negative / Trade-offs

- forward-auth 多一次表查询（可用 token 前缀或同查询短路）
- 前端账号槽需备份/恢复，避免把模拟态当成长期账号

### Follow-up

- 模拟会话热表归档（>90 天）记 OPT
- 本期不为事件配置自动消费者（人工审计 + DLT）
- **审计标识、必填理由、用户收信箱见 ADR-0038**
