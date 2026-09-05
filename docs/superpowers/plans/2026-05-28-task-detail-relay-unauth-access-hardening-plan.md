# 实施计划: 任务详情 relayToTrae 未登录访问加固

> 设计: `docs/superpowers/specs/2026-05-28-task-detail-relay-unauth-access-hardening-design.md`

## Task 1: AuthSessionGuard 清除 authToken
- [ ] `auth_session_guard_service.js` 失败时 `localStorage.removeItem('authToken')`
- [ ] Vitest: profile 403 → 清除 userId + authToken

## Task 2: Navbar 对齐 profile 判定
- [ ] `Navbar.logic.vue` fetchCurrentUser 改调 profile API
- [ ] profile 200 → isAuthenticated true；失败 → false

## Task 3: Playwright 回归
- [ ] 扩展 `TaskDetail.unauthenticated-redirect.playwright.test.js`：127.0.0.1 + task 847744505890045952

## 验证命令

```bash
cd task2app/front_project/app && npm test -- --run src/tests/domain/auth/
cd task2app/playwright/front_project && npx playwright test TaskDetail.unauthenticated-redirect.playwright.test.js
```
