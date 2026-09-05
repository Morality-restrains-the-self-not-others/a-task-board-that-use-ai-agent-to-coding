# DDD 领域模型：taskGitOauth

**日期：** 2026-07-15  
**限界上下文：** Git Site OAuth App Credentials（与 taskAuth 身份上下文分离）

## 聚合

### OauthCredentialBinding（聚合根）

| 字段 | 说明 |
|------|------|
| provider_key | `github` 或 `gitlab:{sp}` |
| task2app_user_id | 平台用户 |
| git_user_id / git_login | 远端身份 |
| refresh_token_cipher | Fernet 密文 |
| bind_status | pending \| active \| failed |
| bind_error | ≤512，敏感红化 |

**不变式：** 仅 pending→active / pending→failed；access 换发要求 active + 非空 cipher。

## 值对象

- `CredentialBindStatus`、`BindError`、`OauthProviderKey`、`OauthAuthorizeTarget`、`RepoOrigin`

## 领域服务

- `OauthAuthorizeRouteDomainService` — 按 origin / service_provider 解析授权目标
- `OauthCredentialBindingLifecycleService` — 绑定生命周期

## 仓储

- `OauthCredentialBindingRepository` → SQLite `api_gitoauthappusercredential`
- 审计为附属写模型（非聚合）：access / task credential audit 表

## 领域事件

| 事件 | 触发 | 投递 |
|------|------|------|
| `GIT_OAUTH_CREDENTIAL_BIND_ACTIVATED` | bind 成功 | 日志 + 表状态；首期不强制 Kafka |
| `GIT_OAUTH_CREDENTIAL_BIND_FAILED` | bind 失败 | 同上 |
| `OauthAuthorizeRouteResolved/Rejected` | 路由解析 | 日志 |

## 与应用服务边界

- Browser OAuth（start/callback）= 应用服务编排 Provider HTTP + 聚合 + 主站 bind
- access-for-user = 应用服务 + 进程内缓存（非领域持久化）
