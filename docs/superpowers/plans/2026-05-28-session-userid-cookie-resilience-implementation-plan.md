# 实施计划：Session userId Cookie 业务韧性

> 设计：`docs/superpowers/specs/2026-05-28-session-userid-cookie-resilience-design.md`

## Tasks

- [x] **Task 1:** `sessionUserIdUtils` — sync + resolver + tests
- [x] **Task 2:** `AuthSessionGuardService` + `Navbar.logic` cookie sync
- [x] **Task 3:** `useTaskDetail.fetchLayerGitIdentityOptions` → resolver
- [x] **Task 4:** `Navbar.logic` 暴露 `userData.userId`；`Navbar.ui` profilePath 优先 currentUser.userId
- [x] **Task 5:** `UserGitIdentities.vue` fetch/create/setDefault → resolver + test
- [x] **Task 6:** Vitest 全绿
- [ ] **Task 7:** Playwright `GitSiteOAuth.missing-userId-cookie` 断言通过（需本地服务）

## 验证

```bash
cd task2app/front_project/app && npm test -- --run \
  src/utils/sessionUserIdUtils.test.js \
  src/tests/domain/auth/auth_domain_model.test.js \
  src/views/UserGitIdentities.test.js
```
