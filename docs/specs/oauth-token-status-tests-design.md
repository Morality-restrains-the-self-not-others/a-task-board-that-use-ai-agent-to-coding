# 设计文档：补充前端单元测试与 E2E 测试

**日期**: 2026-06-28
**背景**: 上一轮交付了 OAuth 授权状态感知功能（`git_repos_status` API + 前端 token_status 判断），但前端 Vitest 单元测试和 Playwright E2E 测试因环境限制未完成。

---

## 1. 待补充项

### 1.1 前端单元测试：`ProjectDetail.test.js` 增加 token_status 用例

**现状**: `ProjectDetail.test.js` 已有 2 个测试：
1. GitHub/GitLab 仓库显示 OAuth 按钮，Bitbucket 不显示（基于 provider 检测）
2. 点击按钮跳转授权页（携带正确参数）

**缺失**: 完全没有 token_status 相关的测试——新增的核心功能没有单测覆盖。

**新增用例**:

| # | 测试 | mock `git_repos_status` | 预期 |
|---|------|------------------------|------|
| 1 | `token_available` 不显示 OAuth 按钮 | `[{repo_url: "...github...", token_status: "token_available", ...}]` | `wrapper.findAll('button').filter(btn => btn.text().includes('OAuth 授权'))` → length 0 |
| 2 | `not_bound` 显示 OAuth 按钮 | `[{repo_url: "...github...", token_status: "not_bound", ...}]` | 按钮可见，label "OAuth 授权" |
| 3 | `token_error` 显示"重试"按钮 | `[{repo_url: "...github...", token_status: "token_error", ...}]` | 按钮可见，label "重试" |
| 4 | `not_applicable` 不显示按钮（Bitbucket 等） | `[{repo_url: "...bitbucket...", token_status: "not_applicable", ...}]` | 无 OAuth 按钮 |
| 5 | 无 `git_repos_status` 字段时向后兼容 | 不返回 `git_repos_status` | 回退到 provider-only 逻辑（已有行为） |

**涉及文件**: `task2app/front_project/app/src/views/ProjectDetail.test.js`（修改，追加测试）

### 1.2 Playwright E2E 测试：项目详情页 OAuth 按钮状态验证

**现状**: 
- `ProjectDetail.revoke-oauth-button.playwright.test.js` — 硬编码验证某用户的项目详情页无 OAuth 按钮（测试账号已授权场景），但没有 mock API，依赖实时数据。
- `ProjectDetail.oauth-auth-redirect-to-project-detail.playwright.test.js` — 验证点击按钮跳转授权页，不验证回跳后状态。

**缺失**: 没有端到端验证 `git_repos_status` → 按钮状态 的完整链路。

**新增测试**: `ProjectDetail.oauth-token-status-button.playwright.test.js`

```
场景 1: API 返回 token_available → 按钮不显示
  1. 登录
  2. 访问项目详情页
  3. page.route() 拦截项目详情 API，注入 git_repos_status: [{token_status: "token_available"}]
  4. 断言: 无 "OAuth 授权" 按钮，无 "重试" 按钮

场景 2: API 返回 not_bound → 按钮显示
  1. 登录
  2. 访问项目详情页
  3. page.route() 拦截项目详情 API，注入 git_repos_status: [{token_status: "not_bound"}]
  4. 断言: 有 "OAuth 授权" 按钮

场景 3: API 返回 token_error → 显示"重试"
  1. mock git_repos_status: [{token_status: "token_error"}]
  2. 断言: 按钮文案为 "重试"
```

**涉及文件**: `task2app/playwright/front_project/tests/ProjectDetail.oauth-token-status-button.playwright.test.js`（新建）

---

## 2. 实现要点

### 2.1 前端单测 mock 策略

遵循已有 `ProjectDetail.test.js` 的 mock 模式：
- 使用 `vi.hoisted()` 声明 mocks
- `mockProjectDetailApi()` 返回包含 `git_repos_status` 的响应
- 使用已有的 `flushRender()` 等待渲染完成
- 使用 `wrapper.findAll('button').filter(btn => btn.text().includes('...'))` 断言

### 2.2 E2E mock 策略

使用 Playwright `page.route()` 拦截 API 调用：
```javascript
await page.route('**/api/tenant/*/projects/*/', async (route) => {
  const response = await route.fetch()
  const body = await response.json()
  body.git_repos_status = [
    { repo_url: body.git_repos[0], token_status: 'token_available', oauth_provider: 'github', oauth_service_provider: 'default' }
  ]
  await route.fulfill({ response, json: body })
})
```

注意：`page.route()` 必须在 `page.goto()` 之前注册。

### 2.3 OAuth provider catalog mock

`ProjectDetail.vue` 在 `onMounted` 中调用 `bootstrapGitOAuthCatalog()` 加载 provider catalog。测试需要 mock 这个 API 调用，否则按钮的 provider 检测会失败。

单测中已在 `mockProjectDetailApi` 中处理了通用 API mock。E2E 中也需要 mock `/api/accounts/git-oauth/providers/` 返回有效的 provider 列表。

---

## 3. 测试文件清单

| 文件 | 操作 | 内容 |
|------|------|------|
| `front_project/app/src/views/ProjectDetail.test.js` | 修改 | 追加 5 个 token_status 测试用例 |
| `playwright/front_project/tests/ProjectDetail.oauth-token-status-button.playwright.test.js` | 新建 | 3 个 E2E 场景 |
| `playwright/front_project/tests/ProjectDetail.oauth-token-status-button.playwright.test.js.testIntent` | 新建 | 意图文档 |
| `playwright/front_project/tests/ProjectDetail.oauth-token-status-button.playwright.test.js.sh` | 新建 | 执行脚本 |

---

## 4. 不变更

- 后端 `project_serializer.py` — 已正确实现
- 前端 `ProjectDetail.vue` — 已正确实现
- 域名 VO `git_repo_token_status.py` — 已正确实现
- 已有测试 — 追加不修改

---

## 5. 验证方式

```bash
# 前端单测
cd task2app/front_project && npm --prefix app run test -- ProjectDetail.test.js

# E2E (需运行中的应用)
cd task2app/playwright && npx playwright test -c front_project/playwright.verify.config.js \
  ProjectDetail.oauth-token-status-button.playwright.test.js
```
