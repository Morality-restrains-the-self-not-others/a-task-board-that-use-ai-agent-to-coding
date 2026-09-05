# 角色权限分析：gitOauth → taskGitOauth

**日期：** 2026-07-15  
**基于：** `2026-07-15-gitoauth-python-to-go-migration-design.md`

## 角色

| 角色 | 说明 |
|------|------|
| 终端用户（已登录） | 经 Gateway JWT；`X-User-Id` 注入后发起 start-from-gateway |
| 终端用户（桥接 JWT） | 持 task2app 签发的短时 start token（query `token`） |
| 内部服务 | taskCredential / taskProject / taskCloud / Django accounts — 调 internal API |
| 匿名（OAuth Provider） | callback 带 code/state，无用户会话头（依赖 signed cookie） |

## 端点权限矩阵

| 端点 | 认证 | 授权要点 |
|------|------|----------|
| `GET /api/health/` | 无 | 公开 |
| `GET …/oauth/start/` | Bridge JWT | `sub` 即用户；typ/aud/iss 校验 |
| `GET …/oauth/start-from-gateway/` | `X-User-Id`（Gateway） | 仅可代表该 uid 发起 |
| `GET …/oauth/callback/` | Session state | state 绑定 uid；禁跨 SP |
| `GET …/<sp>/oauth/callback/` | 同上 / 配置路由 | 歧义 SP → 409 |
| `POST …/internal/*/oauth/*` | 服务间（内网） | 首期与 Python 一致：无额外 mTLS；依赖网络隔离；body `user_id` 为目标用户 |
| Swagger/schema | 内网/文档 | 与现网一致 |

## 变更影响

- **无新增权限模型**；行为对齐 Python。
- 清理 Python 后，权限边界不变，仅实现语言变更。
- **敏感数据：** refresh 密文、access 明文仅 ephemeral 返回；审计禁明文 token。

## 审计要求

- access-for-user / token-use-report / task-credential-audit 继续写审计表
- 日志禁止打印 token/secret；失败日志截断 500 字符
