# Value Stream: GitLab OAuth profile_failed 修复

> 设计：`docs/superpowers/specs/2026-05-27-gitlab-oauth-profile-failed-fix-design.md`

## Value Summary

项目详情页 GitLab OAuth 授权完成后能成功读取用户资料并回跳，不再出现 `profile_failed` Toast。

## End-to-End Flow

```text
[项目详情点击 OAuth 授权]
  → [task2app start → gitOauth authorize URL]
  → [GitLab 授权页（scope 合法）]
  → [gitOauth callback 换票 + GET /api/v4/user]
  → 【交付点】回跳项目详情，无 profile_failed
```

## Value Increments

### Increment 1: scope 配置与 GitLab 应用对齐（薄切片）

- `port_config.json` GitLab scope 改为 `read_repository api read_user`
- `gitService/scripts/sync_local_oauth_app_scopes.sh` + `run.sh` 自动同步 Doorkeeper scopes
- Playwright E2E 回归

### Increment 2: fail-fast 校验（支撑）

- `gitOauth/config/provider_registry.py` 启动时拒绝缺少 read_user/api 的 GitLab scope

### Increment 3: 可观测性（增强）

- `gitlab_tokens.py` profile 失败日志含 HTTP status/body 片段

## 验收

- Playwright `ProjectDetail.gitlab-oauth-profile-failed-debug.playwright.test.js` 通过
- 新 token scopes 含 `read_user` 与 `api`
