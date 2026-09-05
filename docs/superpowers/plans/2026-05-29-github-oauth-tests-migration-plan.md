# 实施计划: GitHub OAuth 测例迁移至 gitOauth

> 设计：`docs/superpowers/specs/2026-05-29-github-oauth-tests-migration-to-gitoauth-design.md`

**Goal:** GitHub OAuth 内部 API 契约测例归属 gitOauth；task2app 删除错位 mock 测例；价值流指向真源。

## Tasks

- [x] **Task 1:** gitOauth — `test_summary_exposes_failed_bind_status_and_error`
- [x] **Task 2:** gitOauth — `test_summary_filters_by_provider_key`
- [x] **Task 3:** gitOauth — `test_user_ids_filters_by_provider_key`
- [x] **Task 4:** 删除 `task2app/.../test_fetch_gitoauth_credential_summary_for_user.py`
- [x] **Task 5:** 删除 `task2app/.../test_fetch_gitoauth_credential_user_ids.py`
- [x] **Task 6:** 更新 `value-stream.yaml` 两处 `test_file`

## 验证

```bash
cd gitOauth && python -m pytest api/tests.py -q
cd task2app/Saas_project && pytest tests/test_github_app_connection_get_user_scoped.py tests/test_gitoauth_per_provider_service_base.py -q
```
