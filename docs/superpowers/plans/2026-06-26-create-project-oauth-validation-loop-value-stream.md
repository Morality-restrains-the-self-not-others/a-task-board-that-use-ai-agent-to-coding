# Value Stream: Create Project 页面 GitLab OAuth 仓库校验闭环

> Derived from design: `docs/specs/create-project-gitlab-oauth-validation-design.md`

## Value Summary

用户在 Create Project 页面输入私有 GitLab/GitHub 仓库地址 → 点击 OAuth 授权 → 完成授权后页面自动重校验仓库可访问性，无需手动刷新或重新输入，形成"输入→校验→授权→重校验→通过"的完整闭环。

## Related Value Streams

- **create-task-project-repo-access-label**: modification — 共享 `validate_git_repo_view` + `resolve_repo_access_tokens` 后端代码；本流为 CreateProject 页面增加 `token_status` 字段和回调重校验，CreateTaskModal 的 repo-access-check 也受益于新增的 `token_status` 区分度
- **project-detail-repo-oauth-row-action**: extension — 复用 OAuth 按钮模式 (`repoOAuthAuthorizeUtils.js`、`useProjectGitOAuthCatalog.js`)、provider 路由逻辑；本流将 OAuth 按钮从 ProjectDetail 扩展到 CreateProject 页面并补全回调→重校验闭环
- **oauth-callback-error-toast**: extension — 复用 `gitSiteOAuthCallbackUtils.js` 的 `applyOAuthCallbackFromRoute` 回调检测；CreateProject 页面新增 success 路径处理（不仅处理 error toast，还触发重校验）
- **gitoauth-binding-state-persistence**: dependency — `bind_status` (pending→active→failed) 语义不变；`access-for-user` 的 `bind_status="active"` 过滤规则不变

## End-to-End Flow

```
用户在 CreateProject 输入 GitLab repo URL
  → validate-git-repo API (含 token_status 检测)
  → 返回 is_accessible=false + token_status=not_bound + oauth_provider=gitlab
  → 前端展示友好提示 + OAuth 授权按钮
  → 用户点击 OAuth 授权
  → start-from-gateway → gitOauth authorize_url
  → 浏览器跳转 GitLab 授权页
  → 用户授权 → gitOauth callback → bind → bind_status=active
  → 重定向回 CreateProject 页面 (带 gitlab=ok)
  → 前端检测 OAuth 回调参数 → 自动触发所有仓库行重校验
  → validate-git-repo 使用新绑定的 token 校验
  → is_accessible=true → 错误消失 → 用户可继续创建项目
```

**Trigger:** 用户在 CreateProject 页面输入私有 Git 仓库 URL  
**Wait points:** GitLab/GitHub OAuth 授权页面交互  
**Delivery point:** OAuth 回调后自动重校验通过，仓库状态从"不可访问"变为"可访问"

## Value Increments

### Increment 1: 后端 token_status 区分 + 文案优化 (Thin Slice)

**Value to user:** 用户看到的错误提示能区分"尚未授权"、"授权服务异常"、"授权后仓库仍不可达"三种情况，不再被统一的技术性文案困惑。

**Scope:**
- `resolve_repo_access_tokens` 返回值增加 `token_status` 字段 (`not_bound` / `token_error` / `token_available` / `not_applicable`)
- `_check_gitlab_repo` 的 `not_bound` 场景文案从"未检测到有效绑定令牌"改为"此仓库需要 GitLab 授权后才能访问，请点击下方「OAuth 授权」按钮完成授权"
- `validate_git_repo_view` 响应增加 `token_status` + `oauth_provider` + `oauth_service_provider` 字段
- `project_repo_access_check_view` 同步受益

**Depends on:** 无

**Test:** `tests/test_validate_git_repo_api.py` — 新增 not_bound / token_error / token_available 状态断言

### Increment 2: 前端 OAuth 回调自动重校验

**Value to user:** 完成 GitLab/GitHub OAuth 授权后回到 CreateProject 页面，仓库地址自动重校验，无需手动刷新或重新输入。用户看到的是从"需要授权"到"可访问"的完整闭环。

**Scope:**
- `CreateProject.vue` 在 `onMounted` 调用 `applyOAuthCallbackFromRoute` 检测 OAuth 回调参数
- 若检测到 `gitlab=ok` 或 `github=ok`，遍历所有仓库行触发 `validateGitRepoRow`
- 移除 `startRepoOAuthConnect` 的 `finally` 块中跳转前的无意义校验
- 重校验完成后清除 URL query 参数

**Depends on:** Increment 1（需要 `token_status` 字段驱动按钮显隐）

**Test:** `front_project/app/src/tests/domain/oauth_callback/` — 新增 CreateProject OAuth 回跳 + 重校验 domain model 测试

### Increment 3: 前端 token_status 驱动 OAuth 按钮

**Value to user:** OAuth 按钮只在真正需要授权时才显示，授权服务异常时显示"重试"而非"授权"，减少用户困惑。

**Scope:**
- `CreateProject.vue` 根据 API 返回的 `token_status` 和 `oauth_provider` 决定按钮显隐和文案
- `token_status=not_bound` → 显示"OAuth 授权"按钮
- `token_status=token_error` → 显示"重试"按钮 + 错误提示
- `token_status=token_available` + 仍不可访问 → 显示"仓库不存在或无权访问"（不显示 OAuth 按钮）

**Depends on:** Increment 2

**Test:** Playwright E2E — `CreateProject.gitlab-oauth-full-flow.playwright.test.js`

## Proposed New Stream Entry

- **name**: `create-project-oauth-validation-loop`
- **domain**: `项目与工作空间`
- **description**: CreateProject 页面 OAuth 授权→校验闭环：token_status 区分 + 回调自动重校验 + 按钮智能显隐

## Candidate YAML Steps

1. `validate-git-repo-token-status` (active) — 后端 token_status 字段 + 文案优化
2. `create-project-oauth-callback-revalidate` (active) — 前端 OAuth 回调自动重校验
3. `create-project-oauth-button-smart-visibility` (planned) — token_status 驱动按钮显隐
