# [运行时] Path A GitLab IP 仓被当成 gitlab:default →「未检测到可用授权」

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-27
- 编号：118
- 维护者：Trae AI 团队

## 现象

- 项目详情「云端开发」仓库卡片：`http://115.29.110.74/example-user/somanyad.git`
- 红字：`无法获取 GitLab 分支：未检测到可用授权。请先在个人资料完成 GitLab 绑定，或先登录本地 GitLab（localhost）后重试。`
- 徽章「未授权」，**没有**「OAuth 授权」按钮。
- `data-traceId` 截断如 `996ec9a4-b03e…`；当时 Loki `{job=~".+"}` 无流（见 OPT-20260827-002）。

## 根因

1. 仓库是租户 Path A GitLab（`git_oauth_tenant_gitlab_oauth_connections.base_url` 为裸 IP），`provider_key` 应为 `gitlab:tenant-{company_id}`。
2. `GET /api/git-oauth/providers/` 只吐 YAML；未按 `company_id` / `X-Tenant-Id` 合并 Path A。前端 catalog 也未带租户。
3. heuristic 只认 `github.com` / `localhost` / 主机名含 `gitlab`。裸 IP **匹配不到**，按钮不出现。
4. `ProviderResolver.matchProvider` 对未命中 YAML 且 URL 以 `.git` 结尾的仓曾回退 `gitlab:default`。用户绑的是 Path A 而非默认 CE → 空 token → `gitlabNeedsAuthMessage`。同类问题见 [108](./108_ztree_push_gitlab_borrow_other_ce_token.md)（禁止借其它 CE token）。

## 解决方案

1. Catalog：`handleGitOauthProviders` 按租户合并 `gitlab:tenant-{id}`（website=`base_url`，不泄漏 `client_secret`）。
2. 前端 `loadGitOAuthProviderCatalog(fetch, companyId)` 走 `buildProviderCatalogRequest`。
3. 未匹配 YAML 的 `*.git` / `gitlab*` **不得**借用 `gitlab:default`；`matchRepoProvider` 用内部 tenant-connection 按 host 对齐 Path A。

## 验证

```bash
cd taskGitOauth && go test ./src/ ./infrastructure/ -count=1 -run 'CatalogIncludesTenantPathA|ByHostPathA'
cd taskProjectService && go test ./src/ -count=1 -run 'PathA|UnknownDotGitDoesNotBorrowDefault'
cd taskFE/app && npx vitest run src/utils/repoOAuthAuthorizeUtils.test.js
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/108_ztree_push_gitlab_borrow_other_ce_token.md`
- `docs/intents/backend/tenant_gitlab_oauth_connection.test-intent.md` 第 6 条
