# Value Stream: 项目详情页 OAuth 授权状态正确反映

> Derived from design: `docs/specs/oauth-auth-state-not-reflected-design.md`

## Value Summary

用户在项目详情页看到 Git 仓库时，OAuth 授权按钮根据实际授权状态智能展示——已授权隐藏按钮，未授权显示"OAuth 授权"，授权失败显示"重试"，无需授权不显示按钮。不再误导已授权用户重复操作。

## Related Value Streams

- **project-detail-repo-oauth-row-action**: extension — 本流在其基础上增加 `token_status` 感知能力。原流 Increment 1 实现了按钮基础可见性（只看 provider），本流将按钮升级为状态感知（看 token_status）。原流明确将"按仓库展示 OAuth 已绑定状态"列为 Future。
- **create-task-project-repo-access-label**: dependency — 复用 `resolve_repo_access_tokens` 服务和 `TokenStatus` 值对象，已在本流的 `repo_access_token_resolver` 中实现。

## End-to-End Flow

用户进入项目详情页 → 后端返回项目详情（含 `git_repos_status` 每 repo 的 `token_status`）→ 前端根据 `token_status` 条件渲染按钮：
  - `token_available` → 不显示 OAuth 按钮（已授权）
  - `not_bound` → 显示"OAuth 授权"按钮
  - `token_error` → 显示"重试"按钮
  - `not_applicable` → 不显示按钮（公开仓库）
→ 用户点击按钮 → OAuth start 流程 → 授权完成回跳 → 项目详情页刷新 → 按钮消失（状态已变为 `token_available`）

## Value Stages Classification

- **Core value**: 已授权仓库不显示"OAuth 授权"按钮（修复核心误导）
- **Essential support**: 项目详情 API 返回 `git_repos_status`；OAuth 回调后自动刷新状态
- **Enhancement**: 仓库行旁显示授权状态图标（绿勾/警告）
- **Future**: 多账号选择（当前自动选首个 active credential）

## Trigger / Wait / Delivery

- **Trigger**: 用户访问项目详情页
- **Wait points**: 后端 `resolve_repo_access_tokens` 调用 gitOauth `summary-for-user/` API
- **Delivery point**: 按钮状态与用户实际授权状态一致

## Value Increments

### Increment 1: 后端项目详情 API 返回 git_repos_status（Thin Slice）
**Value to user:** 前端能通过单次 API 调用获取所有 repo 的授权状态。  
**Scope:**
- `ProjectSerializer.to_representation()` 新增 `git_repos_status` 字段
- 复用 `resolve_repo_access_tokens` 服务（已实现）
- 单元测试覆盖 `git_repos_status` 输出
**Depends on:** 无（后端独立变更）

### Increment 2: 前端 ProjectDetail 按钮状态感知
**Value to user:** 已授权仓库不再显示"OAuth 授权"按钮，token_error 显示"重试"。  
**Scope:**
- `ProjectDetail.vue` 的 `shouldShowRepoOAuthButton` 加入 `token_status` 判断
- `repoOAuthButtonLabel` 根据 `token_status` 切换文案
- `fetchProjectDetail` 初始化 `gitRepoTokenStatus`
**Depends on:** Increment 1

### Increment 3: OAuth 回调后自动刷新状态 + E2E
**Value to user:** 授权完成后回到项目详情页，按钮自动消失，无需手动刷新。  
**Scope:**
- `ProjectDetail.vue` OAuth 回调后重新拉取项目详情
- Playwright E2E：验证已授权 repo 不显示按钮
**Depends on:** Increment 2

## Mapping To Existing Streams
- `project-detail-repo-oauth-row-action`：本流为其**扩展**，在按钮基础可见性之上增加 `token_status` 感知
- `create-task-project-repo-access-label`：复用 `resolve_repo_access_tokens` 服务和 TokenStatus VO

## Proposed New Stream Entry
- **name**: `project-detail-oauth-token-status-aware`
- **domain**: `项目与工作空间`
- **description**: 项目详情页 OAuth 授权按钮根据 token_status 智能展示，替代原有的"只看 provider"逻辑
