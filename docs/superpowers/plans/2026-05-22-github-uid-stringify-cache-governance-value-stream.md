# Value Stream: Github UID 字符串化与缓存治理设计

> Derived from plan: `/Users/task2app/.cursor/plans/githubuid字符串化设计_d6ecfaa9.plan.md`
>  
> YAML constraints reference: `valueStream/README.md`

## Value Summary
对任务绑定仓库的用户而言，系统统一仅接受字符串 `github_user_id`，并保证 access token 缓存过期时间必填，从而避免跨服务 UID 类型漂移与脏缓存导致的授权失败。

## End-to-End Flow
用户触发 GitHub 绑定/取 token/发起 PR  
→ task2app 归一化 `github_user_id` 并请求 git-oauth  
→ git-oauth 返回字符串 UID 的凭据摘要与 token  
→ task2app 以字符串 UID 写入绑定与缓存（`access_token_expires_at` 非空）  
→ 云侧/出站链路使用统一契约继续执行  
→ 用户在仓库绑定、凭据摘要、PR 推送链路获得稳定结果。

## Affected Streams
- `oauth-token-fetch-timeout-governance`
- `cloud-integration`
- `internal-api-timeout-governance`
- `task2app-outbound-governance`

## Value Increments

### Increment 1: UID 字符串契约 + 缓存非空薄切片（Thin Slice）
**Value to user:** 连接 GitHub 后，首次拉取 token 就走统一字符串 UID，且缓存不会产生 `expires_at=null` 的无效记录。  
**Depends on:** nothing  
**Impacted stream:** `oauth-token-fetch-timeout-governance`

Steps:
1. `oauth-github-uid-cache-thin-slice`
   - `test_file`: `tests/test_layer_github_oauth_tokens.py`
   - fields:
     - `saas-backend.accounts_usergithubappaccesstokencache.github_user_id`
     - `saas-backend.accounts_usergithubappaccesstokencache.access_token_expires_at`
     - `saas-backend.projects_taskgithubrepooauthbinding.github_user_id`
     - `git-oauth.api_githubappusercredential.github_user_id`

### Increment 2: 绑定与摘要链路字符串一致性
**Value to user:** 多账号场景下，仓库绑定和连接摘要返回的 `github_user_id` 口径一致（全字符串），避免选错账号。  
**Depends on:** Increment 1  
**Impacted stream:** `cloud-integration`

Steps:
1. `cloud-github-binding-string-contract`
   - `test_file`: `tests/test_github_task_repo_oauth_binding.py`
   - fields:
     - `saas-backend.projects_taskgithubrepooauthbinding.github_user_id`
     - `saas-backend.projects_taskgithubrepooauthbinding.repo_slug`
2. `cloud-credential-summary-string-contract`
   - `test_file`: `tests/test_fetch_gitoauth_credential_summary_for_user.py`
   - fields:
     - `git-oauth.api_githubappusercredential.github_user_id`
     - `git-oauth.api_githubappusercredential.github_login`

### Increment 3: 内网 RPC 超时治理下的 UID 协议收敛
**Value to user:** 内网调用即使超时/失败，错误契约和重试路径仍保持字符串 UID，不出现类型分叉。  
**Depends on:** Increment 2  
**Impacted stream:** `internal-api-timeout-governance`

Steps:
1. `internal-rpc-uid-string-contract-guard`
   - `test_file`: `tests/cloud/services/test_layer_github_oauth_tokens_observability.py`
   - fields:
     - `saas-backend.projects_taskgithubrepooauthbinding.github_user_id`
     - `saas-backend.projects_taskgithubrepooauthbinding.user_id`
     - `git-oauth.api_githubappusercredential.github_user_id`

### Increment 4: 出站异步链路 UID 字符串透传
**Value to user:** 任务推送后续 PR 异步链路使用同一 UID 契约，避免发起后在异步阶段失败。  
**Depends on:** Increment 3  
**Impacted stream:** `task2app-outbound-governance`

Steps:
1. `outbound-pr-uid-string-propagation`
   - `test_file`: `tests/test_github_pr_after_layer_push_async.py`
   - fields:
     - `saas-backend.projects_taskgithubrepooauthbinding.github_user_id`
     - `saas-backend.projects_todo.id`
     - `saas-backend.projects_todo_comment.content`

## Self-Check
1. **薄切片端到端**：Increment 1 覆盖「用户触发授权→拉取 token→写缓存」完整路径，且直接产生可见收益（凭据可用、缓存可复用）。  
2. **依赖顺序正确**：先收敛 UID+缓存基础，再扩展到绑定摘要、内网治理、最后到出站异步；后续增量仅依赖前置增量。  
3. **字段命名合规**：所有字段均为 `<service>.<table>.<field>` 三段式；`service` 使用 `runAll/config.yaml` 中存在的 `saas-backend`、`git-oauth`。  
4. **计划边界一致**：不新增 stream，仅在既有 4 条 stream 内补充本主题步骤；`oauth-async-job-future` 仍保持 planned，不激活。  
