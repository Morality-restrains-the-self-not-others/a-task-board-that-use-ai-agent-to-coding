# OAuth 回调失败 Toast 设计

**日期：** 2026-05-27  
**状态：** 待批准  
**问题：** 项目详情页（及同类回跳页）在 OAuth 授权失败后 URL 带 `?gitlab=profile_failed`（等），页面无任何错误说明。

---

## 背景与根因

### 回跳链路

1. 前端在发起 OAuth 前调用 `setGithubAppReturnTarget(returnKey, nextPath)`，并请求 `/api/accounts/{github|gitlab}/app/start/?return_key=...&next=...`。
2. 授权完成后，`gitOauth` 重定向到 `/oauth/github-app/callback/?returnKey=...&provider=gitlab&gitlab={code}`。
3. `GithubAppCallbackContinue.vue` 从 `localStorage` 取出 `nextPath`，合并 `gitlab={code}` 后 `router.replace` 回原页面。
4. **缺口：** 多数回跳目标页（如 `ProjectDetail.vue`）未读取 `route.query.gitlab` / `github`，用户看不到失败原因。

### 已有参考实现

`UserGitSiteOAuthSettings.vue` 通过 `GITLAB_CALLBACK_HINTS` / `GITHUB_CALLBACK_HINTS` 与 `applyGithubQueryMessage()` 展示内联错误，并 `router.replace` 清除 query。该页**不**需要 Toast（避免与内联重复）。

### 后端错误码（GitLab 示例）

| code | 含义 |
|------|------|
| `ok` | 成功 |
| `bad_state` | CSRF / session 校验失败 |
| `exchange_failed` | authorization_code 换票失败 |
| `no_refresh` | 无 refresh_token |
| `no_access` | 无 access_token |
| `profile_failed` | 无法读取 GitLab 用户资料 |
| `no_gitlab_id` | 无用户 ID |
| `return_expired` | return_key 过期（中间页兜底） |

GitHub 侧对称，query 键为 `github`。

---

## 目标与非目标

### 目标

- 所有通过 **`return_key` 回跳** 的页面，在 `gitlab`/`github` query 为**非 ok** 时，展示全局 **Toast 错误**（`toastService.error`，约 5s）。
- 处理后从 URL **移除** `gitlab`/`github` query，避免刷新重复弹窗。
- 文案与 `UserGitSiteOAuthSettings` **一致**（抽取共享 hints）。
- 覆盖用户选定的范围：**所有使用 return_key 的入口**。

### 非目标

- 不修复 `profile_failed` 后端根因（GitLab API、token、scope、网络）。
- 不在 `gitlab=ok` 时弹成功 Toast（保持安静；设置页仍可用内联成功文案）。
- 不把「启动 OAuth 失败」的行内红字改为 Toast（`repoOAuthActionErrorByUrl` 等保持不变）。

---

## 价值流影响

| 价值流 | 影响 |
|--------|------|
| `project-detail-repo-oauth-row-action` | 补齐回调失败前端可观测性 |
| `task-detail-oauth-repo-url-row-action` | 任务详情 OAuth 回跳失败可见 |
| `gitoauth-binding-state-persistence` | 与 `bind_error` 互补的即时回调反馈 |
| `task-detail-oauth-binding-guidance` | 预检/绑定失败可视化增强 |

---

## 方案（已选定）

**全局路由守卫 + 共享文案工具**，避免在每个 Vue 文件重复 `onMounted`。

### 1. `gitSiteOAuthCallbackUtils.js`

- 导出 `GITHUB_CALLBACK_HINTS`、`GITLAB_CALLBACK_HINTS`（从 `UserGitSiteOAuthSettings.vue` 迁出）。
- `resolveOAuthCallbackMessage(provider, code)` → `{ severity: 'success'|'error', message }`。
- `applyOAuthCallbackFromRoute(route, router, options)`：
  - 检测 `route.query.gitlab` 或 `route.query.github`（优先处理有值者；若两者皆有，先 gitlab 再 github，或按字母序处理一次导航只消费一个——实现时**只处理第一个非空 provider 参数**）。
  - `severity === 'error'` 时调用 `options.onError(message)`。
  - `router.replace` 删除已消费的 query 键。
  - 返回 `{ provider, code, message, severity } | null`。

### 2. `setupOAuthCallbackToastGuard(router)`

在 `main.js` 中 `app.use(router)` 之后注册：

```javascript
router.afterEach((to) => {
  if (shouldSkipOAuthCallbackToast(to.path)) return
  applyOAuthCallbackFromRoute(to, router, {
    onError: (msg) => toastService.error(msg, 5000),
  })
})
```

**跳过 Toast 的路径**（仍可由页面内联处理）：

- `/profile/git-site-oauth/`
- `/tenant/:tenant/profile/git-site-oauth/`
- `/user/:id/profile/git-site-oauth/`

匹配方式：`path` 以 `/profile/git-site-oauth` 结尾或包含 `/profile/git-site-oauth/`。

### 3. `UserGitSiteOAuthSettings.vue` 重构

- 删除本地 hints 常量，改为 import 共享模块。
- 保留 `applyGithubQueryMessage` 与内联 `errorMessage` / `successMessage`（跳过路径不注册 Toast，无重复）。

### 4. `return_key` 入口对齐清单

| 文件 | 当前 | 变更 |
|------|------|------|
| `ProjectDetail.vue` | 有 return_key | 依赖全局 guard，无页面内重复逻辑 |
| `TaskDetailLinkedProjectsPanel.vue` | 有 return_key | 同上 |
| `TaskDetailGithubPrCredentialPanel.vue` | 有 return_key | 同上 |
| `UserGitSiteOAuthSettings.vue` | 有 return_key + 内联 | 共享 hints；路径在 skip 列表 |
| `CreateTaskModal.vue` | **仅 `next`，无 return_key** | **补齐** `createGithubAppReturnKey` + `setGithubAppReturnTarget` + `return_key` 查询参数，与项目详情一致，使创建任务页 OAuth 失败也能回跳并 Toast |

### 5. 测试

| 测试 | 内容 |
|------|------|
| `gitSiteOAuthCallbackUtils.test.js` | `resolveOAuthCallbackMessage('gitlab','profile_failed')` 文案；`applyOAuthCallbackFromRoute` mock router.replace 清 query |
| `oauthCallbackToastGuard.test.js`（或 utils 内） | skip 路径不调用 `onError`；项目路径 mock `onError` |
| `ProjectDetail.test.js` | 可选：集成测 guard 行为，或保留现有 OAuth 启动测 |
| `CreateTaskModal` 相关测 | 断言 start URL 含 `return_key`（若已有测则扩展） |

---

## 领域概念清单（轻量）

| 类型 | 候选 |
|------|------|
| Bounded Context | 前端展示 / Git 站点 OAuth 集成 |
| Value Object | `OAuthCallbackCode`（gitlab/github query 值） |
| Domain Event | `OAuthCallbackCompleted`（含 success/failure） |
| 应用服务 | `applyOAuthCallbackFromRoute`（消费 query + 通知 + 清理 URL） |

---

## 验收标准

1. 项目详情页：`?gitlab=profile_failed` → 红色 Toast「授权失败：无法读取 GitLab 用户资料」，URL 不再含 `gitlab`。
2. 任务详情页（关联项目 OAuth 绑定）：同上。
3. GitHub PR 凭据面板回跳：同上（`github=*`）。
4. 个人中心 Git 站点 OAuth 设置页：仍显示内联错误，**不**弹 Toast。
5. `gitlab=ok`：无 Toast（设置页可有内联成功）。
6. 创建任务弹窗 OAuth：使用 return_key 后，失败回跳至原页并 Toast。

---

## 实施顺序建议

1. 抽取 `gitSiteOAuthCallbackUtils.js` + 单元测试  
2. 注册 `setupOAuthCallbackToastGuard`  
3. 重构 `UserGitSiteOAuthSettings.vue` import  
4. `CreateTaskModal.vue` 补齐 return_key  
5. 手工验证：项目详情 `profile_failed`、任务详情、`ok` 无 Toast、设置页内联  

---

## 开放问题

无（范围已确认为所有 return_key 入口 + 全局 guard）。
