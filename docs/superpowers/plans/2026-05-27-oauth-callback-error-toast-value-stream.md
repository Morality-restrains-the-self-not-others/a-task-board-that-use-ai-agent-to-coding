# Value Stream: OAuth 回调失败 Toast

> 源自设计：`docs/superpowers/specs/2026-05-27-oauth-callback-error-toast-design.md`

## Value Summary

在 **项目详情、任务详情、创建任务** 等 OAuth `return_key` 回跳页面上，当授权失败（如 `?gitlab=profile_failed`）时，用户能**立即看到**与设置页一致的中文 Toast 说明，且 URL 不再残留回调 query，避免「路径有错误码、页面却静默」的困惑。

## End-to-End Flow

```text
[用户在业务页点击 OAuth 授权]
  → [写入 return_key + nextPath，跳转 gitOauth / GitLab 授权页]
  → [用户同意/拒绝授权，gitOauth 处理换票 / profile / 落库]
  → [重定向 /oauth/github-app/callback/?returnKey&provider&gitlab={code}]
  → [GithubAppCallbackContinue 合并 query 回业务页]
  → 【交付点】router.afterEach 消费 gitlab/github query
       ├─ 非 ok → toastService.error(映射文案) + router.replace 清 query
       └─ git-site-oauth 路径 → 跳过 Toast（设置页内联错误）
```

### 价值阶段分类

| 能力 | 分类 | 说明 |
|------|------|------|
| 共享回调文案 + `applyOAuthCallbackFromRoute` | 核心价值（薄切片主体） | 将错误码转为用户可读消息 |
| 全局 `setupOAuthCallbackToastGuard` | 核心价值 | 任意回跳页自动 Toast，无需每页 onMounted |
| 设置页 skip + hints 复用 | 必要支撑 | 避免 Toast 与内联重复 |
| `CreateTaskModal` 补齐 `return_key` | 必要支撑 | 创建任务 OAuth 与项目详情回跳一致 |
| `profile_failed` 根因修复（GitLab API） | Future | 设计明确非目标 |

### 依赖与阻塞点

- **依赖 gitOauth 回调契约**：Location / 中间页必须携带 `gitlab` 或 `github` 错误码（已有）。
- **依赖 `GithubAppCallbackContinue`**：必须把 code 合并进 `nextPath`（已有）。
- **阻塞点**：无 return_key 的入口（原 `CreateTaskModal`）回跳后 Toast 不会触发 → 增量 2 补齐。

## Value Increments

### Increment 1: 回调失败 Toast 薄切片（End-to-End）

**用户价值：** 在项目详情页 OAuth 失败后，看到红色 Toast（如「无法读取 GitLab 用户资料」），URL 自动去掉 `?gitlab=...`。

**范围（最小）：**

- `gitSiteOAuthCallbackUtils.js`（hints + resolve + apply + skip 判断）
- `main.js` 注册 `setupOAuthCallbackToastGuard`
- Vitest：`gitSiteOAuthCallbackUtils.test.js`

**Depends on：** 现有 OAuth 回跳链路（gitOauth + `GithubAppCallbackContinue` + `ProjectDetail` return_key）

**验收：** 手工或 Vitest — `profile_failed` → `toastService.error` + query 清除；`ok` 不 Toast。

---

### Increment 2: return_key 入口对齐

**用户价值：** 创建任务弹窗内 OAuth 失败后，同样回到原页并 Toast。

**范围：**

- `CreateTaskModal.vue` 增加 `createGithubAppReturnKey` / `setGithubAppReturnTarget` / `return_key` 参数

**Depends on：** Increment 1

**验收：** start URL 含 `return_key`；失败回跳后 Toast（可扩 Vitest / 手工）。

---

### Increment 3: 设置页 hints 统一（无行为回归）

**用户价值：** 个人中心 Git 站点 OAuth 页错误文案与全局一致；仍仅内联展示、不弹 Toast。

**范围：**

- `UserGitSiteOAuthSettings.vue` import 共享 hints
- `shouldSkipOAuthCallbackToast` 覆盖三条 `git-site-oauth` 路由

**Depends on：** Increment 1

**验收：** 设置页 `?gitlab=bad_state` 仅内联错误、无 Toast；文案与 utils 一致。

---

### Increment 4: 多入口回归护栏

**用户价值：** 任务详情关联项目 OAuth、GitHub PR 凭据面板等 return_key 回跳页在重构后仍稳定 Toast。

**范围：**

- 文档化回归清单（`view_test`）+ 可选 Playwright
- 与 `task-detail-oauth-repo-url-row-action`、`gitoauth-binding-state-persistence` 交叉验证

**Depends on：** Increment 1–3

**验收：** 任务详情 / PR 凭据面板失败回跳均有 Toast；设置页仍 skip。

## 与现有价值流的关系

| 现有流 | 关系 |
|--------|------|
| `project-detail-repo-oauth-row-action` | 启动 OAuth 已有；本特性补齐**回调失败**可观测性 |
| `task-detail-oauth-repo-url-row-action` | 任务详情行级绑定回跳失败可见 |
| `gitoauth-binding-state-persistence` | 后端 `bind_error` 与前端即时 Toast 互补 |
| `task-detail-oauth-binding-guidance` | 预检/引导失败可视化增强 |

建议 **新增独立价值流** `oauth-callback-error-toast`，并在上述流的 `impacted_steps` 中标注关联（valueStream UI 影响分析）。

## 测试策略说明

| 层级 | 文件 | 编排工具 |
|------|------|----------|
| 单元（前端） | `front_project/app/src/utils/gitSiteOAuthCallbackUtils.test.js` | Vitest（**不在** valueStream pytest `working_dir` 内） |
| 价值流环节 | `view_test/oauth-callback-error-toast-*.md` | valueStream `planned` → 手工 / Playwright |
| 后端契约 | `gitOauth/api/tests.py`（`gitlab=profile_failed` Location） | gitOauth 自有 pytest |

薄切片 **用户可触发的 E2E** 以 Vitest + 手工为主；YAML 中 active 环节可挂 Saas 侧契约测或保持 `planned` 直至前端 pytest 桥接。

## 实施状态（对照设计）

| 增量 | 状态 |
|------|------|
| Increment 1 | 已实现（utils + guard + vitest） |
| Increment 2 | 已实现（CreateTaskModal return_key） |
| Increment 3 | 已实现（Settings import + skip） |
| Increment 4 | 待补（view_test / Playwright 回归） |
