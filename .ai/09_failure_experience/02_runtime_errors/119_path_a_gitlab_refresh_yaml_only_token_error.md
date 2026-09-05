# [运行时] Path A GitLab 重试授权后仍「授权异常」：refresh 只查 YAML

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-27
- 编号：119
- 维护者：Trae AI 团队

## 现象

- 项目详情「云端开发」`http://115.29.110.74/example-user/somanyad.git` 徽章「授权异常」，点「重试」能打开 GitLab 授权（`gitlab-start-from-gateway` 200），跳回本页后仍是授权异常。
- Loki `trace_id` 如 `d491e421-ffc9-4fe6-a5b8-51b9d61771e7`（与项目 GET 同链）：
  `{job=~".+"} |= "access-for-user"` → `未找到 provider_key=gitlab:tenant-877397588196749312 对应的配置`

## 根因

1. Path A 租户 GitLab 的 `provider_key` 是 `gitlab:tenant-{company_id}`，配置在 `git_oauth_tenant_gitlab_oauth_connections`，不在 YAML `git-oauth-providers`。
2. `refreshAccessToken` 只扫 `GetProviderConfigs("gitlab")`（YAML）。OAuth 回调已把 refresh 写入凭据表，但 `access-for-user` 换票时找不到 client_id/secret/website → 502 → 前端 `token_error`。
3. 项目详情回调处理误用 `oauthResult.outcome === 'success'`（实际字段是 `severity`），成功回流后既不 `fetchProjectDetail` 也不强制重探活。

同类：授权按钮/catalog 见 [118](./118_path_a_ip_git_borrows_gitlab_default.md)；禁止借 `gitlab:default` token 见 [108](./108_ztree_push_gitlab_borrow_other_ce_token.md)；克隆凭证 YAML miss 见 [120](./120_fork_bound_clone_path_a_yaml_miss.md)。

## 解决方案

1. `refreshAccessToken`：YAML 未命中时走 `resolveProviderByServiceProvider`（`ResolveByServiceProviderWithDB`）。
2. `providerConfigForRepoURL`：先 `ResolveProviderByGitsiteWithDB`，再 YAML alias。
3. 项目详情：`severity === 'success'` 时 `refreshGitReposOAuthStatus(..., { force: true })` 并 emit `oauth-callback-success`。

## 验证

```bash
cd taskGitOauth && go test ./src/ -count=1 -run 'RefreshAccessTokenUsesTenantPathA|RefreshAccessTokenUnknownTenant|ProviderConfigForRepoURLUsesTenantPathA'
cd taskFE/app && npx vitest run src/composables/useProjectDetailGitRepos.batch.test.js src/components/ProjectDetailGitReposSection.oauth-row.test.js
```

## 关联

- Loki：`access-for-user refresh 失败 uid=877397583960502272`
- `taskGitOauth/src/internal_handlers.go` `refreshAccessToken`
- `taskFE/app/src/components/ProjectDetailGitReposSection.vue`
