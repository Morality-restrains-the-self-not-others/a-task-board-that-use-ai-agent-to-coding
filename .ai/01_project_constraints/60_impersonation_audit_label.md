# 模拟登录必须在日志与审计中标识（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-23
- 维护者：Trae AI 团队
- 适用范围：整个 monorepo 所有 HTTP/RPC 访问日志、结构化业务日志、审计表、领域事件
- 架构决策：**ADR-0038**（accepted）
- 约束索引：第 55 条

## 核心原则

当请求处于**系统管理员（或持有 `user:impersonate` 的平台员工）以目标用户身份登录**的会话中时，**所有**日志与审计记录必须同时标明：

| 字段 | 含义 |
|------|------|
| `impersonating` | `true` |
| `impersonator_user_id` | 操作者（管理员）用户 ID（字符串） |
| `impersonated_user_id` | 被模拟用户 ID（字符串） |
| `impersonation_session_id` | 模拟会话 ID（字符串） |

禁止只打 `user_id=被模拟用户` 而看不出真人操作者。禁止在日志中输出模拟 token、密码、完整理由正文（理由落库收信箱/会话表）。

## 注入来源（SSOT）

1. **网关**：forward-auth 在模拟 token 下必须注入 `X-Impersonator-Id`、`X-Impersonation-Session-Id`；`X-User-Id` 仍为目标用户。
2. **tracelog HTTP 中间件**：从上述头写入 request context；每条 `http_request` 及同 ctx 的 slog 必须带上表内字段。
3. **业务审计**：`auth_impersonation_session` 与用户收信箱信件必须含操作者、目标、会话、理由正文。Kafka `UserImpersonationStarted` / `UserInboxMessageCreated` 含操作者、目标、会话及 `reason_len`（禁止事件体带完整理由，避免总线明文扩散）。

## 触发

- 新增/修改访问日志、slog/JSON 日志、审计表、forward-auth 头
- 改模拟登录开始/结束路径
- 评审「以该用户身份登录」相关排障日志

## 验收

```bash
# 模拟会话下 http_request 日志含 impersonator_user_id
rg -n 'impersonator_user_id' shareLib/tracelog taskAuth --glob '*impersonation*'
# 网关透传会话头
rg -n 'X-Impersonation-Session-Id' taskGateway
```
