# 功能意图：租户级自建 GitLab OAuth 连接

## 意图

租户管理员在设置中登记本公司自建 GitLab 的 OAuth Application（每租户最多一条）；成员在「Git 网站授权」中对该实例完成个人 OAuth 绑定，以便任务侧访问该 GitLab 上的私有仓库。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 管理员保存连接 | TenantGitLabOAuthConnectionUpserted | TENANT_GITLAB_OAUTH_CONNECTION_UPSERTED | taskGitOauth PUT | 审计/缓存失效 | 证据豁免：首期仅 HTTP 写库/审计，不强制 Kafka |
| 管理员删除连接 | TenantGitLabOAuthConnectionDeleted | TENANT_GITLAB_OAUTH_CONNECTION_DELETED | taskGitOauth DELETE | 级联解绑用户 credential | 证据豁免：首期仅 HTTP 写库/审计，不强制 Kafka |
| 成员 OAuth 绑定成功 | GitOauthCredentialBindActivated | （既有策略） | callback | 审计 | 证据豁免：与 gitoauth_go_migration 同策略（首期日志/审计） |
| GET 连接 / providers | — | — | — | — | 纯查询 |

## 验收

- 设置页可配置；redirect_uri 可复制，且为租户隔离路径 `/api/accounts/tenant-{company_id}/oauth/callback/`（不同租户 URI 不同）
- 非管理员 PUT/DELETE → 403
- 成员 providers 可见本租户条目
- DELETE 后绑定清除
