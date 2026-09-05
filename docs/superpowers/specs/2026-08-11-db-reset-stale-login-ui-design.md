# 清库后导航栏仍显示已登录 — 设计文档

- **日期**: 2026-08-11
- **状态**: implemented（/goal 已落地，2026-08-11；待公网精准编译重启验证）
- **迭代**: db-reset-stale-login-ui
- **作者**: claude
- **页面证据**: `https://www.daydaymoney.com/auth/login/?next=...` 导航栏仍渲染 `account-switcher-trigger`，可见文本「未设置昵称」+ 公网 IP `110.87.16.238`，profile 链到 `/user/874599469415428096/profile/`

---

## 1. 问题陈述

在 runAll `http://10.2.150.68:9999/` 执行「清空全部数据库」并重新初始化后，浏览器刷新业务站：用户已被踢到登录页（`/auth/login/`），但 **Navbar 仍展示账号切换器**，看起来像仍处于登录状态。

## 2. 运行时证据（2026-08-11）

| 检查项 | 结果 |
|--------|------|
| `task_auth.auth_user` | 1 行：仅 `bootstrap-admin` |
| `task_auth.auth_customtoken` | **0** 行 |
| `auth_user` 中 `874599469415428096` | **不存在** |
| `auth_user_profile` | 0 行 |

结论：服务端会话真源已空；UI 上的「已登录」**不是**服务端仍认旧用户，而是前端本地态误展示。

## 3. 根因分析

### 3.1 已正确的服务端防线（不是本次回归点）

历史修复已覆盖清库绕过：

| 防线 | 位置 | 作用 |
|------|------|------|
| 活 token 校验 | `resolveUserIDForForwardAuth` + `userHasLiveToken`（OPT-20260807-002） | 仅签名 `userId` cookie、无 `auth_customtoken` 行 → 401 |
| 签名 cookie | `user_id_cookie.go`（OPT-20260807-004） | 裸 `userId` 拒绝 |
| profile 401 清 cookie | `clearResidualAuthCookies`（OPT-20260807-006） | 登录页调 profile 时清 HttpOnly `userId`/`token` |

当前库态（用户不存在 + token=0）下，上述路径应拒绝认证。

### 3.2 本次失效链路（前端 + 清 cookie 路径被绕过）

```
localStorage.currentUserId = 874599469415428096  （清库后仍残留）
        ↓
Navbar.fetchCurrentUser（OPT-20260810-015）优先 getStoredUserId()
        ↓
GET /api/accounts/users/me/ → 401/失败（用户/token 已不存在）
        ↓
❌ 仍执行：
   userData = { isAuthenticated: true, username: '', userId: storedId }
        ↓
AccountSwitcher 显示「未设置昵称」+ fetchPublicClientIp → 公网 IP
        ↓
且因 localStorage 有 id，**不再请求 /profile/** → clearResidualAuthCookies 永不触发
```

关键代码（`taskFE/app/src/components/Navbar.logic.vue`）：

```js
if (userIdFromStorage) {
  const meResponse = await apiFetch(`/api/accounts/users/me/...`)
  if (meResponse.ok) { /* applyMePayload */ return }
  // ❌ /me/ 失败仍标已登录
  userData.value = { isAuthenticated: true, ..., userId: userIdFromStorage }
  return
}
// 仅当 localStorage/cookie 皆空时才走 profile → setLoggedOutUser()
```

对比：无 storage id 时走 profile，`!profileResponse.ok` 会正确 `setLoggedOutUser()`。

### 3.3 与「已登录用户打开登录页仍显示昵称」的兼容

`Navbar.logic.authRouteSession.test.js` 要求：**有效会话**访问 `/auth/login/` 时仍展示昵称。  
修复不得改回「登录页强制 guest」；必须区分：

| `/me/` 结果 | 期望 |
|-------------|------|
| 200 + 有效载荷 | 保持已登录导航（现有行为） |
| 401 / 403 / 404 | **登出 UI** + 清本地 userId/token 缓存；并确保 HttpOnly cookie 被清 |
| 5xx / 网络异常 | 不假装已登录；可 `setLoggedOutUser` 或保留上次已确认会话（本次选：鉴权失败才登出，5xx 走 catch → 已有 `setLoggedOutUser`） |

## 4. 方案

### 4.1 推荐方案（双端小改）

**A. 前端（主修）— `Navbar.logic.vue` `fetchCurrentUser`**

当 `userIdFromStorage` 存在且 `/me/` 返回 **401/403/404**：

1. `clearCachedAuthToken()`
2. `clearStoredUserId()`
3. `setLoggedOutUser()`
4. **再发一次** `GET /api/accounts/users/profile/`（可忽略 body），触发服务端 `clearResidualAuthCookies`（兼容现有 OPT-006，无需等前端部署与后端同时发布）

**B. 后端（加固）— `handleGetUser` `/me/` 未认证分支**

与 profile 对齐：在写 401 之前调用 `clearResidualAuthCookies(w)`。  
举一反三：检查其它「登录态探测」入口是否同样应在 401 清残留 cookie（至少 `/me/` 与 `profile`）。

### 4.2 拒绝的方案

| 方案 | 原因 |
|------|------|
| 清库时让浏览器主动清 cookie | 跨域/跨进程，runAll 无法操作用户浏览器 |
| 登录页强制始终 guest | 破坏「已登录访问登录页仍显示昵称」 |
| 仅文档说明「请手动清 cookie」 | 违背 OPT-006 已承诺的自动清理预期 |

### 4.3 架构变更

**不需要**更新 `docs/architecture/`（纯 Bug 修复，无组件/数据流增删）。  
**No-ADR**: trivial bugfix, no architectural impact。

### 4.4 Python 新增接口

**不触发**（无新增 Python endpoint）。

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 会话失效后 UI 纠正为未登录 | — | — | — | 纯查询/会话探测失败处理，无系统事实变更，无对应 MQ 事件 |

## 6. 🕸️ Code Review Graph 分析

- **CRG status**: 图存在（Nodes 103 / 过期基线 2026-08-10），MCP `code-review-graph` 未在本会话注册；CLI `search fetchCurrentUser` → 0 节点（图未覆盖 taskFE Navbar 符号）。
- **记录**: `CRG unavailable for symbol impact: graph index miss on Navbar.fetchCurrentUser; proceeded with ripgrep + live DB evidence.`
- **手工爆炸半径**:
  - 修改：`Navbar.logic.vue`、`Navbar.logic.fetchCurrentUser.test.js`（新增）
  - 可能触及：`auth_users.go`（`/me/` 401 清 cookie）、既有 `auth_user_profile.go` / `TestGetUserProfile401ClearsResidualCookies`
  - 回归保护：`Navbar.logic.authRouteSession.test.js`（有效会话登录页仍显示昵称）必须保持绿

## 7. 价值流影响

- **影响流**: `user-auth`（登录态展示/会话探测）
- **新流**: 不需要
- **字段**: 无 schema 变更
- **测试**: 新增 taskFE 单测；可选 taskAuth `/me/` 401 清 cookie 回归
- **完整切片**: 交 `/4-value-stream`（若进入流水线）

## 8. Domain Concept Inventory（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | Auth / Session presentation（taskFE Navbar + taskAuth） |
| Key Entities | User、CustomToken（会话真源） |
| Candidate Aggregates | Auth session（token 行） |
| Domain Events | 无新增（见 §5） |

## 9. 验收标准

1. 清库 + 初始化后，浏览器不手动清 cookie/localStorage，刷新任意页（含登录页）：Navbar **只显示登录/注册**，无 `account-switcher-trigger`。
2. 有效会话用户打开 `/auth/login/`：仍显示昵称（`authRouteSession` 用例绿）。
3. `/me/` 401 响应含 `Set-Cookie` 清理 `userId`/`token`（若做后端加固）。
4. 单测：`localStorage` 有旧 id + `/me/` 401 → `isAuthenticated === false` 且调用了 `clearStoredUserId` 路径。

## 10. 🏛️ 架构变更影响

- **结论**: 不创建 target 架构文件（Bugfix）。
- **已有相关文档**: `docs/superpowers/specs/2026-05-31-dev-database-reset-design.md` §16（清库后 cookie 残留预期）——实施后补充一句：Navbar 不得因 localStorage 残留在 `/me/` 鉴权失败时展示已登录；`/me/` 401 亦清理残留 cookie。

---

## 审批记录

- `python_api_approval`: not_applicable
- `design_approval`: approved_via_goal（2026-08-11）
