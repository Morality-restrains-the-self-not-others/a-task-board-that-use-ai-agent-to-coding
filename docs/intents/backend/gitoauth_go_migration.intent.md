# 功能意图：gitOauth 迁 Go（taskGitOauth）

## 意图

将 Git 站点 OAuth App 凭据服务从 Django `gitOauth` 迁至 Go `taskGitOauth`，保持 HTTP 契约与数据所有权，并在能力对齐后清理 Python 实现。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| OAuth callback 绑定主站成功 | GitOauthCredentialBindActivated | GIT_OAUTH_CREDENTIAL_BIND_ACTIVATED | taskGitOauth callback | 结构化日志/审计表 | 证据豁免：首期不强制 Kafka（与存量一致） |
| OAuth callback 绑定失败 | GitOauthCredentialBindFailed | GIT_OAUTH_CREDENTIAL_BIND_FAILED | taskGitOauth callback | 同上 | 证据豁免：首期不强制 Kafka（与存量一致） |
| access-for-user 换发成功 | — | — | taskGitOauth | api_gitoauthappaccesstokenuseaudit | 审计表副作用，无领域事件名 |
| health / summary-for-user / available-ids | — | — | — | — | 纯查询，无领域事件 |

## 纯查询例外

- health、summary-for-user、available-ids：无事件

## 验收

- 端口 8002；路径与 JSON 字段兼容
- 存量 Fernet 密文可换票
- Python `gitOauth` 进程不再由 runAll 启动
