# 设计文档：任务详情页未登录跳转登录页

**日期：** 2026-05-27  
**状态：** 已实现  
**页面：** `http://localhost:4000/tenant/.../workspace/.../task-detail/.../?relayToTrae=true`

---

## 问题

用户在未登录（或会话已失效）时访问任务详情页（含 `relayToTrae=true`），期望自动跳转到 `/auth/login/`，但实际仍停留在任务详情页，页面 API 报错、relay 面板无法正常使用。

---

## 根因（已验证）

### 1. 路由守卫过早信任 `userId` Cookie

`router.js` 中 `checkUserAuthenticated()` 逻辑：

```javascript
const userId = getCookie('userId')
if (userId) {
  return true  // ← 未校验 session / profile
}
```

**复现：**

| 场景 | 最终 URL | 行为 |
|------|----------|------|
| 无任何 Cookie | `/auth/login/?relayToTrae=true` | ✅ 正确跳转 |
| 仅有过期/无效 `userId` Cookie | 停留在 task-detail | ❌ 误判已登录 |

任务详情路由已配置 `meta: { requiresAuth: true }`，问题不在路由定义，而在认证判定函数。

### 2. 登录后无法回到原任务详情（次要）

未认证重定向时仅传递 `query`（如 `relayToTrae=true`），**未保存完整 path**。`Login.vue` 已支持 `localStorage.postLoginRedirect`，但路由守卫未写入。

---

## 方案

### A. 修复认证判定（核心）

**原则：** 受保护路由放行前，必须确认服务端会话有效。

**`checkUserAuthenticated()` 调整：**

1. 若 `window.currentUser?.isAuthenticated === true`，仍调用 profile 校验（或并行快速校验），避免模板注入与真实会话不一致。
2. **删除**「仅有 `userId` Cookie 即返回 true」的短路逻辑。
3. 调用 `GET /api/accounts/users/profile/`（`credentials: 'include'`）：
   - `200` → 已登录
   - `401/403/网络失败` → 未登录；清除本地 `userId` Cookie（若存在），避免下次再次误判

可选：保留 `userId` 存在时的 optimistic 路径，但必须在 profile 返回前 **await** 校验结果（当前代码注释称「避免刷新误跳登录」，应以 profile 为准而非 Cookie  alone）。

### B. 登录重定向保留完整 URL

**`router.beforeEach` 未认证分支：**

```javascript
const returnUrl = to.fullPath  // 含 path + query，如 /tenant/.../task-detail/.../?relayToTrae=true
localStorage.setItem('postLoginRedirect', returnUrl)
next({ path: '/auth/login/' })
```

- 不再把业务 query（`relayToTrae`）误当作登录页 query 传递。
- 登录成功后 `Login.vue` 现有逻辑读取 `postLoginRedirect` 并 `window.location.href` 跳回。

**安全：** `postLoginRedirect` 仅接受站内相对路径（以 `/` 开头、非 `//`），与现有 `githubAppReturnStorage` 同类校验可复用。

### C. 非目标

- 不改变 `accessCode` _guest 访问_ 的产品策略（若未来支持免登录 task-detail，需单独 `meta` 与后端契约；本次仍要求登录）。
- 不修改 relayToTrae 业务逻辑本身。

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `user-auth` / `login` | 前端会话校验与登录后回跳 |
| `task-detail-runtime-relay` | 未登录用户必须先登录再进入 relay 直启面板 |

不涉及数据库 schema 变更。

---

## 领域概念（轻量）

- **Bounded Context：** 用户与认证、任务协作
- **实体：** `User`、会话（Django session + CustomToken）
- **值对象：** `AuthSessionState`（前端：profile 可达 + Cookie 一致）
- **领域服务：** 路由守卫 `checkUserAuthenticated`

---

## 测试计划

### 单元 / 组件

- 新增 `router.auth-guard.test.js`（或 vitest mock `apiFetch`）：
  - 无 Cookie → profile 403 → false
  - 仅有 userId、profile 403 → false，并清除 userId
  - profile 200 → true

### Playwright

- `TaskDetail.unauthenticated-redirect.playwright.test.js`：
  1. 清空 Cookie 访问 `.../task-detail/.../?relayToTrae=true` → URL 为 `/auth/login/`
  2. 仅设置无效 `userId` Cookie → 同样跳转登录页
  3. （可选）mock 登录后 `postLoginRedirect` 回到原 task-detail URL

---

## 验收标准

1. 无 Cookie 访问带 `relayToTrae=true` 的任务详情 → 跳转 `/auth/login/`
2. 仅有无效 `userId` Cookie → 跳转 `/auth/login/`，不渲染任务详情
3. 登录成功后回到原任务详情 URL（含 `relayToTrae=true`）
4. 已登录有效会话 → 正常进入任务详情，行为无回归

---

## 风险

| 风险 | 缓解 |
|------|------|
| 每次受保护路由导航多一次 profile 请求 | profile 轻量；可后续加短期内存缓存（TTL 数秒） |
| 清除 userId 影响并行标签页 | 仅 profile 失败时清除，与真实登出一致 |
