# Value Stream: 任务详情 OAuth session userId 解析

> Derived from design: `docs/superpowers/specs/2026-05-28-task-detail-oauth-session-user-id-design.md`

## Value Summary

已登录用户在任务详情页点击「OAuth 绑定」时，即使用户 ID cookie 缺失，也能完成 connection 预检并跳转 Git 授权。

## Related Value Streams

- **task-detail-oauth-binding-adjustment**（2026-05-25）：extension — 本增量修复行级 OAuth 的 user_id 前置条件
- **task-detail-oauth-repo-url-row-action**：modification — repo-url connection 检查依赖 session userId 解析
- **github-connection-get-user-id**（2026-05-27）：dependency — 后端 connection GET 签名已修复，本增量补前端 user_id 来源

## End-to-End Flow

[用户点击 OAuth 绑定] → [解析 session userId（cookie 或 profile）] → [GET user-scoped connection?repo_url=] → [未连接则 start OAuth 跳转] → [用户完成 Git 授权]

## Value Increments

### Increment 1: session-user-id-resolver-thin-slice（Thin Slice）

**Value to user:** 无 userId cookie 时仍可发起 OAuth 绑定，不再看到「缺少 userId」  
**Scope:** `resolveAuthenticatedUserId()` + `TaskDetailLinkedProjectsPanel` 接入 + Vitest  
**Depends on:** 无

### Increment 2: oauth-regression-guard（Future）

**Value to user:** 同页多仓库、relayToTrae 场景下 OAuth 状态检查稳定  
**Scope:** 扩展 `repo-row-oauth-regression-guard` 用例  
**Depends on:** Increment 1
