# [运行时] 派生任务已绑定 Git OAuth，引导克隆仍报授权未齐（YAML miss → missing）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-27
- 编号：120
- 维护者：Trae AI 团队

## 现象

- 任务详情「云端开发」派生时仓库行显示 **已绑定 Git OAuth**（`fork-auto-run-oauth-bound`）。
- 容器引导克隆失败：`引导克隆失败：仓库 Git 授权未齐（http://115.29.110.74/example-user/somanyad.git）`。
- 后端 `error_code=REPO_CLONE_CREDENTIALS_INCOMPLETE`，`missing_repo_credentials` 含该 Path A IP URL。
- 任务例：`task_880739244337819648`（同类 `task_880716211791360000`，OPT-20260827-025）。

## 根因

1. Fork UI 走 `GET /api/git-oauth/user-app-connection/?repo_url=` → `ResolveProviderByGitsiteWithDB` 命中租户 Path A（`gitlab:tenant-{company_id}`）→ `connected:true`。
2. 容器 `repo-clone-credentials` 在 `taskCredentialService.buildCredentials` 用 **YAML-only** `ProviderResolver.ResolveProvider`。`115.29.110.74` 不在 `git-oauth-providers` → 空键。
3. 空键被当成「无 identity / 未绑定」写入 `MissingIdentityRepos`（HTTP 409），**从未**调用 gitOauth `access-for-user`。用户其实已绑定。
4. 与 [118](./118_path_a_ip_git_borrows_gitlab_default.md)（catalog/按钮）、[119](./119_path_a_gitlab_refresh_yaml_only_token_error.md)（refresh 只查 YAML）同一 Path A 漏网：克隆侧禁止借 `gitlab:default` 之后，漏了 gitsite SSOT。

## 解决方案

1. YAML miss 且仓有 host、非 `github.com` 时，换票键为 `gitsite:{host}`，不要标 missing。
2. `GitoauthHTTPClient.FetchAccessToken` 对该键 POST `/api/internal/gitsite/{host}/oauth/access-for-user/`（gitOauth 已按 Path A DB 解析），body 只带 `user_id`。
3. `repo_clone_credentials.provider` 仍为 `gitlab`，`git_http_username=oauth2`。layer-oauth 同一套 fallback。

## 验证

```bash
cd taskCredentialService && go test ./application/ ./infrastructure/ -count=1 -run 'TestBuildCredentialsPathA|TestGitsiteFallback|TestResolveLayerOauthTokens_PathA|TestFetchAccessTokenGitsite'
cd taskGitOauth && go test ./src/ -count=1 -run 'TestGitsiteAccessForUserPathAIPHostReachesHandler'
```

## 关联

- `taskCredentialService/application/clone_provider.go`
- `taskCredentialService/infrastructure/gitoauth_client.go`
- `docs/intents/frontend/create_task_oauth_bind.intent.md` T8b
- OPT-20260827-025
