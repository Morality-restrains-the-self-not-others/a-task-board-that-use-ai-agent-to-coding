# Implementation Plan: GitLab OAuth profile_failed 修复

## Task 1: 配置对齐 ✅

- [x] `port_config.json` localhost GitLab scope → `read_repository api read_user`
- [x] `gitService/scripts/sync_local_oauth_app_scopes.sh`
- [x] `gitService/run.sh` 启动后调用 sync

## Task 2: gitOauth fail-fast ✅

- [x] `provider_registry.py` 校验 GitLab scope 须含 read_user/api/read_api
- [x] `api/tests.py` 覆盖 `read_repository write_repository profile` 拒绝场景

## Task 3: 可观测性 ✅

- [x] `gitlab_tokens.py` profile 失败记录 status + body 预览

## Task 4: E2E 回归 ✅

- [x] Playwright 完整 OAuth 链路测试

## Verify

```bash
./gitService/scripts/sync_local_oauth_app_scopes.sh
cd task2app/playwright/front_project && npx playwright test tests/ProjectDetail.gitlab-oauth-profile-failed-debug.playwright.test.js
cd gitOauth && DJANGO_SETTINGS_MODULE=config.settings python -m pytest api/tests.py::ProviderConfigCompatibilityTests -q
```
