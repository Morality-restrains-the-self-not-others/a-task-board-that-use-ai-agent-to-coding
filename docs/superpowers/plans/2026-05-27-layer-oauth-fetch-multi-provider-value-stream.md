# Value Stream: 容器层 OAuth 拉取 AccessToken 多 Provider

> Derived from design: `docs/superpowers/specs/2026-05-27-layer-oauth-fetch-multi-provider-design.md`

## Value Summary

容器 UI「拉取 AccessToken」对 `port_config.json` gitOauth 配置的全部 Git 站点（含 `localhost:8012` GitLab）可用，与 bootstrap 克隆凭证路径一致。

## End-to-End Flow

[用户点击拉取 AccessToken] → [onlineServiceJS 枚举层内 origin → repo_match_key] → [Django 匹配任务仓库 + TaskRepoIdentity] → [gitOauth access-for-user 按 provider] → [写入 .task2app_access_token] → [用户可 push/clone 复用 token]

## Value Increments

### Increment 1: Django repo_match_key 换 token 薄切片（Thin Slice）
**Value to user:** GitLab localhost 仓库可经 API 换发 access_token。  
**Scope:** `resolve_git_auth_by_repo_match_keys_for_container_task` + API 扩展 + 单测。  
**Depends on:** 无。

### Increment 2: onlineServiceJS fetch/refresh 多 Provider（Core Value）
**Value to user:** 容器 UI 按钮不再报「未发现 github.com」。  
**Scope:** `layerGitOauthFetchTokenFiles.mjs`、`layerGitOauthRefreshPush.mjs`、`repoMatchKey.mjs`。  
**Depends on:** Increment 1。

### Increment 3: 错误文案与 GitLab slug 解析（Enhancement）
**Value to user:** 失败提示可行动；`gitlab_repo_slug_from_url` 支持 configured host。  
**Depends on:** Increment 2。

### Increment 4: 回归保护（Essential Support）
**Scope:** GitHub 现有测试 + value-stream.yaml 条目。  
**Depends on:** Increment 2。
