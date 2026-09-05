# 设计文档：GitLab OAuth profile_failed 修复

**日期：** 2026-05-27  
**状态：** 已实施  
**问题：** 项目详情页 OAuth 授权后 Toast「授权失败：无法读取 GitLab 用户资料」（`gitlab=profile_failed`）

---

## 根因（Playwright 核验）

1. gitOauth 授权 URL 请求 scope：`read_repository api read_user`（来自 `port_config.json`）
2. 本地 GitLab Doorkeeper 应用注册 scope 原为：`read_user read_repository write_repository profile`（**缺少 `api`**）
3. 授权页曾报：`The requested scope is invalid, unknown, or malformed`
4. 历史已签发 token 实际 scope 为 `read_repository write_repository profile`（**无 `read_user`/`api`**）
5. 换票成功后 `GET /api/v4/user` 返回 403 → `fetch_gitlab_user_profile` 抛错 → `profile_failed`

## 方案

1. **GitLab 应用 scope 对齐**：`gitService/scripts/sync_local_oauth_app_scopes.sh` 在 GitLab 启动后把 Doorkeeper Application 的 scopes 更新为包含 `api read_user`
2. **run.sh 自动同步**：`gitService/run.sh start` 结束时调用上述脚本
3. **回归测试**：Playwright E2E `ProjectDetail.gitlab-oauth-profile-failed-debug.playwright.test.js` 覆盖完整授权链路

## 非目标

- 不改 gitOauth 换票/profile 业务逻辑（根因在 GitLab 应用配置）
- 不修改 port_config 中的 scope 字符串（已与 gitOauth 一致）

## 验收

- Playwright：项目详情 OAuth → GitLab 登录 → Authorize → 回跳项目详情，无 `profile_failed` Toast
- 新 token scopes 含 `read_user` 与 `api`
