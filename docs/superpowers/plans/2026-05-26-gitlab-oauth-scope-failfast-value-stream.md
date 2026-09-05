# Value Stream: GitLab OAuth Scope 合法化与启动期防呆

> Derived from design: `docs/superpowers/specs/2026-05-26-gitoauth-gitlab-provider-config-compat-design.md`

## Value Summary
项目详情页与任务详情页中的 GitLab OAuth 授权链路稳定可用，且当配置误填 GitHub scope 时在服务启动阶段即可阻断并给出可定位错误。

## End-to-End Flow
[用户点击 OAuth 授权] -> [task2app 解析 repo_url 与 provider 路由] -> [gitOauth 读取 provider 配置并构造 authorize URL] -> [GitLab 接受合法 scope 并展示授权页] -> [用户继续授权]

## Value Stage Classification
- Core value
  - GitLab provider scope 配置合法化（`read_repository api read_user`）
  - OAuth start 链路继续命中本地/公网 GitLab provider
- Essential support
  - task2app 启动期 scope fail-fast 校验
  - gitOauth 启动期 scope fail-fast 校验
  - 错误信息携带 provider/service_provider 与非法 token
- Enhancement
  - 回归测试覆盖（start URL、配置校验）
- Future
  - 统一多 provider scope 词法白名单与 schema 规范化

## Value Increments

### Increment 1: GitLab Scope Thin Slice (E2E)
**Value to user:** 点击 OAuth 授权时能进入 GitLab `/oauth/authorize`，不再因 invalid scope 被拒。  
**Scope:** 修正 GitLab provider 的 `target.scope`，保持 GitHub 配置不变。  
**Depends on:** nothing

### Increment 2: Startup Fail-Fast Guard (Core)
**Value to user:** 配置误填不会在运行期才暴露，服务启动即报错，减少线上排障成本。  
**Scope:** 在 task2app 与 gitOauth 配置归一化入口增加 `provider=gitlab` 的 scope 合法性校验。  
**Depends on:** Increment 1

### Increment 3: Regression Guard & Verification
**Value to user:** 后续改动不易回归，能持续保证 GitLab OAuth 可用性。  
**Scope:** 增加/更新最小回归测试，覆盖合法 scope 放行、GitHub 风格 scope 拒绝。  
**Depends on:** Increment 2

## Impacted Existing Streams
- `project-detail-repo-oauth-row-action`（直接）
- `gitoauth-binding-state-persistence`（直接）
- `task-detail-oauth-repo-url-row-action`（间接）
- `task-detail-oauth-binding-guidance`（间接）

## Proposed YAML Entries (for valueStream config)
- Stream name: `gitlab-oauth-scope-failfast-governance`
- Domain: `项目与工作空间`
- Step slicing:
  - `gitlab-scope-thin-slice` (active)
  - `gitlab-scope-startup-failfast` (active)
  - `gitlab-scope-regression-guard` (active)

## Candidate Test Files
- `tests/test_github_app_start_redirect_uri.py`
- `tests/test_git_oauth_scope_validation.py`

## Candidate Fields
- `saas-backend.projects_projectrepo.repo_url`
- `saas-backend.accounts_user.id`
- `git-oauth.api_githubappusercredential.provider`
- `git-oauth.api_githubappusercredential.bind_status`
- `git-oauth.api_githubappusercredential.bind_error`
- `git-oauth.api_githubappusercredential.scope`

