# Value Stream: gitOauth GitLab Provider 配置兼容修复

> Derived from design: `docs/superpowers/specs/2026-05-26-gitoauth-gitlab-provider-config-compat-design.md`

## Value Summary

当用户在项目详情页点击 GitLab OAuth 授权时，系统能够稳定命中正确 provider 配置并跳转到 GitLab 授权页，不再出现 `bad_state` 回落。

## End-to-End Flow

用户在项目详情页点击 `OAuth 授权`
-> task2app 生成 start token 并返回 `authorize_url`
-> gitOauth 解析 token 并按 `service_provider/allowedHost` 命中 provider 配置
-> gitOauth 组装 `client_id + redirect_uri + scope` 并 302 到 GitLab `/oauth/authorize`
-> 用户在 GitLab 完成授权并回调主站
-> 前端回到目标项目页并可继续后续操作。

## Affected Existing Streams

- `project-detail-repo-oauth-row-action`
  - `project-detail-repo-oauth-provider-routing`（active）
- `gitoauth-binding-state-persistence`
  - `task-detail-gitlab-auth-guard-integration`（active，间接受益）
- `cloud-integration`
  - `cloud-credential-summary-string-contract`（active，配置命中稳定性相关）

## New Stream Needed?

不需要新增业务流。该需求属于现有 `project-detail-repo-oauth-row-action` 的稳定性修复与兼容增强，可在该流下补充步骤。

## Value Increments

### Increment 1: GitLab 路由修复薄切片（Thin Slice）
**Value to user:** 点击 OAuth 后可进入 GitLab 授权页，不再立刻回落 `bad_state`。  
**Scope:** 仅修复 gitOauth provider 配置归一化与命中逻辑，确保 GitLab local provider 可读。  
**Depends on:** nothing

### Increment 2: 双结构兼容加固（Core Value）
**Value to user:** 不同环境下（list/dict 结构）都能稳定发起授权，降低环境差异导致故障。  
**Scope:** 统一支持 `gitOauth` 配置 list/dict 两种结构，保证 provider/provider_key/service_provider 一致。  
**Depends on:** Increment 1

### Increment 3: 可观测与回归守卫（Essential Support）
**Value to user:** 再次故障时可快速定位（而不是黑盒 `bad_state`）。  
**Scope:** 授权 start 失败分类日志 + 单测 + Playwright 回归。  
**Depends on:** Increment 2

### Increment 4: 健康信号增强（Enhancement）
**Value to user:** 运维可提前发现“服务在线但配置空”的隐患。  
**Scope:** 健康接口补充 provider 配置摘要（如 gitlab 配置计数）。  
**Depends on:** Increment 3

## Fields Impact

本次以配置解析与路由逻辑为主，默认不引入新的持久化字段。  
相关引用字段（已有）：

- `saas-backend.projects_projectrepo.repo_url`
- `saas-backend.accounts_user.id`
- `git-oauth.api_githubappusercredential.provider`

## Test Impact

- 新增/更新 `gitOauth` 侧 provider 归一化单测（覆盖 list + dict）
- 新增/更新 `gitOauth` GitLab start 路由单测（`service_provider=local-gitlab`）
- 新增/更新 Playwright 用例：项目详情 OAuth 二跳应进入 `localhost:8012/oauth/authorize`

## Status Changes

- `project-detail-repo-oauth-row-action` 可追加一个 planned/active 步骤用于配置兼容修复与回归守卫。
- 其他流状态不变。

## Cross-Stream Dependencies

- `project-detail-repo-oauth-row-action` -> `gitoauth-binding-state-persistence`
  （前者保证授权入口稳定，后者消费授权结果状态）
- `project-detail-repo-oauth-row-action` -> `cloud-integration.cloud-credential-summary-string-contract`
  （授权入口稳定后，凭据摘要链路可持续生效）

