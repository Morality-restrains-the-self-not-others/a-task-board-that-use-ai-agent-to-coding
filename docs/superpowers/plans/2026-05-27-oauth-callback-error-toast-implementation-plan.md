# OAuth 回调失败 Toast Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** OAuth `return_key` 回跳失败时，在业务页展示全局 Toast 并清除 URL 中的 `gitlab`/`github` query；设置页保持内联错误。

**Architecture:** 领域层 `domain/oauth_callback/` 负责文案目录、路由快照解析与 outcome；应用层 `gitSiteOAuthCallbackUtils.js` 组装仓储并注册 `router.afterEach`；`toastService` 为基础设施注入。

**Tech Stack:** Vue 3, Vue Router, Vitest, 现有 `toastService`

**契约文档:**
- 设计：`docs/superpowers/specs/2026-05-27-oauth-callback-error-toast-design.md`
- 价值流：`docs/superpowers/plans/2026-05-27-oauth-callback-error-toast-value-stream.md`
- DDD：`docs/superpowers/plans/2026-05-27-oauth-callback-error-toast-ddd-model.md`

---

## 当前状态（2026-05-27）

| 增量 | 状态 |
|------|------|
| Increment 1 薄切片（guard + utils） | ✅ 已完成 |
| Increment 2 CreateTaskModal return_key | ✅ 已完成 |
| Increment 3 设置页 hints + skip | ✅ 已完成 |
| DDD 领域层 + utils 门面重构 | ✅ 已完成 |
| Increment 4 多入口回归 | ✅ 自动化（guard 测试）；手工 view_test 可选 |

---

## File Map

| 路径 | 职责 |
|------|------|
| `domain/oauth_callback/**` | 值对象、实体、仓储、领域服务、事件 |
| `utils/gitSiteOAuthCallbackUtils.js` | 应用层：默认仓储、guard、向后兼容 export |
| `main.js` | 注册 `setupOAuthCallbackToastGuard` |
| `views/UserGitSiteOAuthSettings.vue` | import `GITHUB_/GITLAB_CALLBACK_HINTS` |
| `components/CreateTaskModal.vue` | return_key 对齐 |
| `tests/domain/oauth_callback/oauth_callback_domain_model.test.js` | 领域 Vitest |
| `utils/gitSiteOAuthCallbackUtils.test.js` | 应用层 Vitest |
| `Saas_project/view_test/oauth-callback-error-toast-*.md` | 手工回归清单 |

---

### Task 1: 验证领域层与应用层测试（基线）

**Files:**
- Test: `task2app/front_project/app/src/tests/domain/oauth_callback/oauth_callback_domain_model.test.js`
- Test: `task2app/front_project/app/src/utils/gitSiteOAuthCallbackUtils.test.js`

- [x] **Step 1:** 在 `task2app/front_project/app` 运行：

```bash
npm test -- --run utils/gitSiteOAuthCallbackUtils.test.js tests/domain/oauth_callback/oauth_callback_domain_model.test.js
```

- [x] **Step 2:** 确认 14 tests 全部通过

**Expected:** 2 files passed, 12 tests

---

### Task 2: Increment 4 — Playwright 或组件测（项目详情 profile_failed）

**Files:**
- Create（可选）: `task2app/playwright/front_project/tests/OAuthCallback.profile-failed-toast.playwright.test.js`
- Modify: `task2app/front_project/app/src/utils/gitSiteOAuthCallbackUtils.test.js`（若仅扩单元测则可跳过 Playwright）

- [x] **Step 1:** 写失败测试——mock `toastService`，模拟导航到带 `?gitlab=profile_failed` 的项目详情路由，断言 `error` 被调用且文案含「无法读取 GitLab 用户资料」

- [ ] **Step 2:** 若 Playwright：在 `front_project/tests` 增加用例，goto 项目 URL 带 query，断言 Toast DOM 可见

- [x] **Step 3:** 运行测试至绿

```bash
cd task2app/front_project/app && npm test -- --run utils/gitSiteOAuthCallbackUtils.test.js
# 若添加 Playwright:
# cd task2app/playwright/front_project && npm test -- OAuthCallback.profile-failed-toast
```

---

### Task 3: Increment 4 — 手工 view_test 签收

**Files:**
- Test: `task2app/Saas_project/view_test/oauth-callback-error-toast-multi-surface-regression.md`

- [ ] **Step 1:** 按清单验证入口：项目详情、任务详情 OAuth 绑定、GitHub PR 凭据、创建任务弹窗

- [ ] **Step 2:** 验证设置页 `/profile/git-site-oauth/` 仅内联、无 Toast

- [ ] **Step 3:** 在 view_test 文件顶部记录签收日期与执行人（可选）

---

### Task 4: 价值流环节状态（可选）

**Files:**
- Modify: `value-stream.yaml` — `oauth-callback-error-toast` 流

- [x] **Step 1:** 将 `callback-hints-guard-thin-slice` 的 `status` 从 `planned` 改为 `active`（若团队约定 Vitest 即薄切片自动化）

- [ ] **Step 2:** 运行 `cd valueStream && go test ./...` 校验 YAML

---

### Task 5: 端到端手工验收（profile_failed 真实链路）

- [ ] **Step 1:** 启动 runAll / 前端 4000 + gitOauth + GitLab

- [ ] **Step 2:** 项目详情点击 OAuth 授权，触发 `profile_failed`（或模拟回跳 URL）

- [ ] **Step 3:** 确认 Toast + URL 无 `gitlab` 参数；刷新不重复弹窗

---

## 验证命令汇总

```bash
# 前端单元 + 领域
cd task2app/front_project/app
npm test -- --run utils/gitSiteOAuthCallbackUtils.test.js tests/domain/oauth_callback/oauth_callback_domain_model.test.js

# 价值流配置（可选）
cd valueStream && go test ./...
```

## 非本计划范围

- 修复 `profile_failed` 后端根因（GitLab API / token / scope）
- 成功态 Toast（`gitlab=ok` 保持静默）
- 将 Vitest 纳入 valueStream pytest runner（需单独桥接任务）
