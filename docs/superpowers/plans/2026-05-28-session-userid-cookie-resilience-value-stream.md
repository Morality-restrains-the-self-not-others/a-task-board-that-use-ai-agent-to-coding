# 价值流：Session userId Cookie 业务韧性

> 设计：`docs/superpowers/specs/2026-05-28-session-userid-cookie-resilience-design.md`

## Related Value Streams

- `2026-05-28-task-detail-oauth-session-user-id-value-stream.md` — OAuth 薄切片 resolver（已交付）
- `user-auth.frontend-auth-guard-redirect` — profile 会话守卫（扩展 cookie sync）

## Increment 1 — 认证边界 Cookie 同步（✅ 已交付）

**Value:** session 有效时自动恢复 userId cookie，guard 后绝大多数 `getCookie` 调用点可用。

**Steps:** profile 200 → `syncUserIdCookieFromProfile` @ Guard + Navbar + resolver

**Tests:** `sessionUserIdUtils.test.js`, `auth_domain_model.test.js`

## Increment 2 — 高风险 async 路径 resolver

**Value:** Git 身份 CRUD、任务详情 Git 身份不因 cookie race 失败。

**Scope:** `UserGitIdentities.vue`, `useTaskDetail.fetchLayerGitIdentityOptions`（后者 ✅）

**Tests:** `UserGitIdentities.test.js`（新增）

## Increment 3 — Navbar 导航链接

**Value:** 个人资料链接不依赖 cookie 时序。

**Scope:** `Navbar.logic` 写入 `userData.userId`；`Navbar.ui` profilePath 优先 `currentUser.userId`

## Increment 4 — Playwright 回归

**Value:** E2E 锁定「清 cookie + session 有效」场景。

**Tests:** `GitSiteOAuth.missing-userId-cookie.playwright.test.js`, 可选 task-detail smoke
