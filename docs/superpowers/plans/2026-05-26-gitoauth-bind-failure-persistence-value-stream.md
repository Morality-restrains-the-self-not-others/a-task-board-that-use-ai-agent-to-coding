# Value Stream: gitOauth 绑定失败留痕与状态化凭据

> Derived from approved change intent: “OAuth 回调先落库，后续失败写失败原因，不删除记录”  
> Related implementation scope: `gitOauth/api/*.py`, `task2app/Saas_project/projects/views/project_views.py`

## Value Summary
当 GitHub/GitLab OAuth 回调在后续绑定主站失败时，系统仍保留凭据记录并标注失败原因，让研发与运维能够可追踪诊断，用户侧看到可解释状态而不是“记录消失”。

## End-to-End Flow
用户触发 OAuth 回调  
-> gitOauth 换票成功  
-> 凭据表先写入 `pending`  
-> 调用主站 bind  
-> 成功则更新为 `active` / 失败则更新为 `failed + bind_error`  
-> task2app 查询摘要时可见失败原因，换票仅使用 `active` 记录  
-> 用户与研发获得可解释结果并可继续补救绑定。

## Affected Existing Streams
- `task-detail-oauth-binding-guidance`（直接影响：失败提示与引导语义）
- `oauth-token-fetch-timeout-governance`（直接影响：gitoauth 凭据读取策略）
- `cloud-credential-summary-string-contract`（直接影响：summary 契约字段扩展）
- `task-detail-oauth-repo-url-row-action`（关联影响：仓库级绑定状态可见性）

## New Stream Needed?
需要新增一个独立流，用于沉淀“凭据状态机 + 失败留痕”这条跨 provider 的能力，避免散落在现有流中难以追踪：

- `gitoauth-binding-state-persistence`（domain: 云平台与资源）

## Value Increments

### Increment 1: 回调先落库薄切片（Thin Slice）
**Value to user:** 即使 bind 失败，数据库也能看到尝试记录，不再“消失”。  
**Scope:** OAuth callback 把凭据先写入，状态设为 `pending`。  
**Depends on:** nothing

### Increment 2: 绑定结果状态化（Core Value）
**Value to user:** 成功/失败状态可区分，失败原因可追踪。  
**Scope:** bind 成功置 `active`，失败置 `failed` 并写 `bind_error`；不再删除行。  
**Depends on:** Increment 1

### Increment 3: 消费侧契约收敛（Essential Support）
**Value to user:** 系统不会使用失败凭据继续换票，避免误判与脏状态。  
**Scope:** `access-for-user` 仅选择 `bind_status=active`；summary 增加 `bind_status/bind_error`。  
**Depends on:** Increment 2

### Increment 4: 回归与可观测加固（Enhancement）
**Value to user:** 回归时及时发现错误，不再重复出现“记录丢失”。  
**Scope:** GitHub/GitLab callback 失败留痕测试、契约测试、迁移发布验证。  
**Depends on:** Increment 3

## Fields Impact
- `git-oauth.api_githubappusercredential.bind_status`（新增）
- `git-oauth.api_githubappusercredential.bind_error`（新增）
- `git-oauth.api_githubappusercredential.refresh_token_cipher`（保持，语义从“存在即可用”变为“需 active 才可用”）
- `git-oauth.api_githubappusercredential.provider`（按 provider_key 区分）
- `git-oauth.api_githubappaccesstokenuseaudit.action`（回调发放审计保持）

## Test Impact
- 更新：`gitOauth/api/tests.py`（summary 契约新增字段）
- 新增/扩展：OAuth callback 失败后留痕测试（GitHub/GitLab）
- 关联验证：`task2app/Saas_project/tests/test_project_branches_gitlab_auth_guard.py`

## Status Changes
- 现有 `task-detail-oauth-binding-guidance` 的后续 planned 步骤可在该能力稳定后逐步激活。
- 新增流 `gitoauth-binding-state-persistence` 建议首步为 `active`（已落地代码与测试）。

## Cross-Stream Dependencies
- `gitoauth-binding-state-persistence` -> `task-detail-oauth-binding-guidance`  
  （前者提供可消费的失败状态，后者负责用户可见引导）
- `gitoauth-binding-state-persistence` -> `oauth-token-fetch-timeout-governance`  
  （凭据可用性判定从“有 refresh”升级为“active 且有 refresh”）

