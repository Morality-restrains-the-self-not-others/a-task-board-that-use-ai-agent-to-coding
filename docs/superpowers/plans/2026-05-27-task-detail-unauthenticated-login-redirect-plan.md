# 实施计划: 任务详情未登录跳转登录页

> 设计: `docs/superpowers/specs/2026-05-27-task-detail-unauthenticated-login-redirect-design.md`  
> 价值流: `docs/superpowers/plans/2026-05-27-task-detail-unauthenticated-login-redirect-value-stream.md`

## Task 1: PostLoginReturnUrl 值对象 + 工具
- [ ] 新增 `domain/auth/value_objects/post_login_return_url_value_object.js`
- [ ] 新增 `utils/authReturnUrl.js`（savePostLoginRedirect）
- [ ] Vitest: 拒绝 `//evil.com`、接受 `/tenant/.../task-detail/...`

## Task 2: AuthSessionGuard 领域服务
- [ ] 新增 `domain/auth/services/auth_session_guard_service.js`
- [ ] 仅以 profile 200 判定已登录；失败时 clear userId
- [ ] Vitest: mock apiFetch 403 → false 且清除 cookie

## Task 3: 接入 router
- [ ] `router.js` 使用 AuthSessionGuard
- [ ] 未认证时 savePostLoginRedirect(to.fullPath) 并 next('/auth/login/')
- [ ] Vitest: router 集成测试（可选 mock）

## Task 4: Playwright 回归
- [ ] `TaskDetail.unauthenticated-redirect.playwright.test.js`
- [ ] 无 cookie / stale userId 均跳转登录

## 验证命令

```bash
cd task2app/front_project/app && npm test -- --run src/tests/domain/auth/
cd task2app/playwright/front_project && npx playwright test TaskDetail.unauthenticated-redirect.playwright.test.js
```
