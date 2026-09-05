# Value Stream: GitLab OAuth Application 启动期自愈创建

> Derived from design: `docs/superpowers/specs/2026-06-22-gitlab-oauth-app-bootstrap-fix-design.md`

## Value Summary

GitLab 容器重建（数据卷清理/新环境部署）后，OAuth Application 自动创建并与 YAML 配置对齐，消除手动介入——用户 OAuth 授权链路在首次部署即可用。

## Related Value Streams

- **`gitlab-oauth-scope-failfast-governance`**: extension — 新增 `gitlab-oauth-app-bootstrap` 步骤，增强现有 `gitlab-scope-startup-failfast` 步骤的 GitLab 侧 Application 存在性保证。原有 stream 覆盖 Django 侧 scope 校验；本修复覆盖 GitLab 容器侧 Application 自愈创建。

## End-to-End Flow

[GitLab 容器启动] → [sync_local_oauth_app_scopes.sh 执行] → [find_or_initialize_by uid] → [Application 已存在? 更新 scopes : 创建 Application] → [gitOauth OAuth start 成功]

## Value Stage Classification

- **Core value** — GitLab 容器重建后 OAuth Application 自动创建，无需手动在 Admin UI 操作
- **Essential support** — 同步脚本幂等化（find-or-create），每次启动均保证 Application 存在且 scopes 对齐
- **Enhancement** — 日志区分 "Created" vs "Updated" 输出
- **Future** — 无需

## Value Increments

### Increment 1: OAuth App Bootstrap (Thin Slice — the whole fix)

**Value to user:** GitLab 容器首次启动或数据卷清理后，OAuth 授权链路零手动介入即可用。

**Scope:** 修改 `gitService/scripts/sync_local_oauth_app_scopes.sh` Ruby runner：
- `Doorkeeper::Application.find_by(uid:)` → `find_or_initialize_by(uid:)`
- 新记录：设置 uid, secret, name, redirect_uri, scopes, confidential
- 已存在：保持现有更新逻辑

**Depends on:** nothing (独立 bug 修复)

## Impacted Existing Streams

| Stream | Impact |
|--------|--------|
| `gitlab-oauth-scope-failfast-governance` | 直接增强 — 新增 bootstrap 步骤 |
| `gitoauth-binding-state-persistence` | 间接增强 — Application 存在是 OAuth 回调落库前提 |
| `project-detail-repo-oauth-row-action` | 间接增强 — 仓库行 OAuth 入口依赖 GitLab provider 可用 |

## Proposed YAML Entries

Add one step to existing `gitlab-oauth-scope-failfast-governance` stream:

- Step name: `gitlab-oauth-app-bootstrap`
- Status: `active`
- Test file: `gitService/scripts/test_sync_oauth_app.sh` (需新增集成测试)
- Fields:
  - `git-service.runtime.oauth_app_bootstrapped` — Application 自愈创建结果
  - `git-oauth.api_githubappusercredential.scope` — 同步后的 scope 对齐
  - `git-oauth.api_githubappusercredential.provider` — provider 维度

## Candidate Test Files

- `gitService/scripts/test_sync_oauth_app.sh` — 集成测试：验证 find-or-create 逻辑
- `tests/test_git_oauth_scope_validation.py` — 已有，无需修改
