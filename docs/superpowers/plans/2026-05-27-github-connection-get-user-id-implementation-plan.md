# Implementation Plan: GitHub Connection GET user_id 签名修复

> Design: `docs/superpowers/specs/2026-05-27-github-connection-get-user-id-design.md`

## Tasks

- [x] **Step 1:** 编写失败测试 `test_get_github_app_connection_user_scoped_route`
  - GET `/api/user/{pk}/accounts/github/app/connection/` 应返回 200（mock gitOauth）
- [x] **Step 2:** 修复 `GithubAppConnectionView.get` 签名，增加 `user_id: str | None = None`
- [x] **Step 3:** 运行 pytest 验证
- [x] **Step 4:** 浏览器刷新 git-site-oauth 页，确认 connection 200

## Commands

```bash
cd task2app/Saas_project && pytest tests/test_github_app_connection_get_user_scoped.py -v
```
