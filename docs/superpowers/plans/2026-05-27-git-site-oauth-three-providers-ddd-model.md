# DDD 模型: Git 网站授权多 service_provider

## 限界上下文

**账户 / Git OAuth**（`accounts`）

## 值对象

- `OAuthProviderKey`：`provider:service_provider`
- `GitOauthProviderCatalogEntry`：目录项（label、website、provider_key）

## 领域服务

- `GitOauthProviderCatalogService`：从 `GIT_OAUTH_PROVIDER_CONFIGS` 展开为展示用目录（无 I/O）

## 应用层 / 基础设施

- `GitOauthProvidersCatalogView`：HTTP 适配器，调用目录展开逻辑
- 既有 `resolve_provider_key` / `issue_git_oauth_start_token` 消费 `service_provider`

## 不变式

- 目录项数量 = 配置中 `provider:service_provider` 唯一键数量
- 设置页 `connection` / `start` 必须携带与选中按钮一致的 `service_provider`
