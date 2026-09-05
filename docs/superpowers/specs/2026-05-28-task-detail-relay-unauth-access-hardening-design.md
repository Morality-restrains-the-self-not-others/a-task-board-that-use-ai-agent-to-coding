# 设计文档：任务详情 relayToTrae 未登录访问加固

**日期：** 2026-05-28  
**状态：** 待批准  
**页面：** `http://127.0.0.1:4000/tenant/827923618468040704/workspace/827923618602258432/task-detail/847744505890045952/?relayToTrae=true`

---

## 问题

未登录用户访问带 `relayToTrae=true` 的任务详情页，期望被引导至 `/auth/login/`，但报告仍可访问该页（或看到任务详情内容）。

---

## 现状调查（2026-05-28）

### 已有修复（2026-05-27）

| 组件 | 状态 |
|------|------|
| `AuthSessionGuardService` — 仅以 `GET /api/accounts/users/profile/` 200 判定已登录 | ✅ 已实现 |
| `router.js` — 未认证时 `savePostLoginRedirect(to.fullPath)` + 跳转 `/auth/login/` | ✅ 已实现 |
| 清除 stale `userId` Cookie | ✅ 已实现 |
| Playwright `TaskDetail.unauthenticated-redirect.playwright.test.js` | ✅ 3/3 通过 |
| Vitest `auth_domain_model.test.js` | ✅ 通过 |

### 本地复现（用户 exact URL）

| 场景 | 最终 URL | 结果 |
|------|----------|------|
| 无 Cookie / 127.0.0.1:4000 / task `847744505890045952` | `/auth/login/` | ✅ 约 1–5s 后跳转 |
| 无效 `userId` Cookie | `/auth/login/` | ✅ |
| 无效 `authToken` localStorage | `/auth/login/` | ✅ |
| 假 `sessionid`（无 userId） | `/auth/login/` | ✅ |
| `GET profile/` 无凭据 | HTTP 403 | ✅ |
| `GET todos/{id}/` 无凭据 | HTTP 403 | ✅ |

**结论：** 核心路由守卫逻辑正确；自动化测试与用户 URL 均会跳转登录页。

### 仍存在的缺口

1. **认证提示清理不完整**  
   `AuthSessionGuard` 失败时仅清除 `userId` Cookie，未清除 `localStorage.authToken`。无效 token 虽会导致 profile 403 并跳转，但与 Navbar 登出逻辑不一致，且可能在边缘场景留下 stale 凭据。

2. **Navbar 与路由守卫判定不一致**  
   `Navbar.logic.vue` 仍以 `userId` Cookie 作为「是否登录」前置条件；而路由守卫以 profile 为准。  
   可能出现：**会话/token 有效 → 可进入 task-detail**，但 Navbar 显示「登录」按钮 → 用户感知为「未登录仍可访问」。

3. **Playwright 未覆盖用户 exact taskId 与 127.0.0.1 origin**  
   现有测试用 `846269443533955072` 与 `localhost:4000`；需补充用户报告 URL 的回归用例。

4. **（非目标）Django catch-all `home()` 无服务端鉴权**  
   开发态直接访问 Vite:4000，不经过 Django catch-all；服务端鉴权为可选加固，本次不做。

---

## 方案

### A. 加固 AuthSessionGuard（核心）

profile 非 200 或网络失败时：

1. 清除 stale `userId` Cookie（已有）
2. **新增** 清除 `localStorage.authToken`
3. Vitest 覆盖 token 清除行为

### B. 对齐 Navbar 认证源（推荐，小改）

`fetchCurrentUser` 改为：

1. 优先调用 `GET /api/accounts/users/profile/`（与守卫一致）
2. profile 200 → `isAuthenticated: true`，并同步 `window.currentUser`
3. profile 失败 → 未登录态；不再仅凭 `userId` Cookie 短路

这样 UI 与路由守卫对「已登录」的定义一致，消除「Navbar 显示未登录但页面可进」的误判。

### C. Playwright 回归扩展

新增用例：

- Origin: `http://127.0.0.1:4000`
- Task ID: `847744505890045952`
- Query: `relayToTrae=true`
- 断言：无 Cookie → `/auth/login/`

### D. 非目标

- 不改变 `accessCode` 免登录策略（若未来支持，需单独 meta + 后端契约）
- 不引入 profile 长期缓存
- 不修改 relayToTrae 业务逻辑

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `user-auth` / `login` | 前端会话校验、凭据清理、Navbar 一致性 |
| `user-auth` / `frontend-auth-guard-redirect` | 扩展 Vitest + Playwright |
| `task-detail-runtime-relay` | 未认证用户必须先登录再进入 relay 直启面板 |

不涉及数据库 schema 变更。

---

## 领域概念（轻量）

- **Bounded Context：** 用户与认证、任务协作
- **实体：** `User`、会话（Django session + CustomToken）
- **值对象：** `AuthSessionState`、`PostLoginReturnUrl`
- **领域服务：** `AuthSessionGuardService`

---

## 验收标准

1. 无 Cookie、无 authToken 访问用户 exact URL → 跳转 `/auth/login/`
2. 仅有无效 `userId` 或无效 `authToken` → 跳转登录，且两者均被清除
3. profile 200 的有效会话 → 正常进入 task-detail，Navbar 显示已登录
4. 登录成功后回到原 URL（含 `relayToTrae=true`）
5. 全部 Vitest + Playwright 回归通过

---

## 风险

| 风险 | 缓解 |
|------|------|
| Navbar 改 profile 增加一次 API 调用 | 与路由守卫同源；可复用守卫结果（V2） |
| 清除 authToken 影响并行标签页 | 仅 profile 失败时清除，与登出一致 |
