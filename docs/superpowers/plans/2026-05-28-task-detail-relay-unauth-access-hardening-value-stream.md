# Value Stream: 任务详情 relayToTrae 未登录访问加固

> Derived from design: `docs/superpowers/specs/2026-05-28-task-detail-relay-unauth-access-hardening-design.md`

## Value Summary

未登录或会话失效用户访问 relayToTrae 任务详情时，被可靠引导至登录页；UI 与路由守卫对「已登录」判定一致。

## Related Value Streams

- **2026-05-27-task-detail-unauthenticated-login-redirect**：modification — 在 Increment 1 基础上加固凭据清理与 Navbar 一致性
- **user-auth / frontend-auth-guard-redirect**：extension — 扩展 Vitest + Playwright 覆盖

## End-to-End Flow

[用户打开 task-detail?relayToTrae=true] → [路由守卫 profile 校验] → [失败则清除 userId + authToken + 写 postLoginRedirect] → [跳转 /auth/login/] → [Navbar 同步 profile 判定]

## Value Increments

### Increment 1: 凭据清理加固（Thin Slice）
**Value to user:** 无效会话/token 不会残留，下次访问必定进登录页  
**Scope:** AuthSessionGuard 清除 authToken；Vitest  
**Depends on:** nothing

### Increment 2: Navbar 与守卫对齐
**Value to user:** 不再出现「Navbar 显示未登录但页面可进」的误判  
**Scope:** Navbar.logic.vue 改 profile 判定  
**Depends on:** Increment 1

### Increment 3: Playwright 回归（用户 exact URL）
**Value to user:** 127.0.0.1 + task 847744505890045952 永久回归  
**Scope:** 扩展 unauthenticated-redirect 测试  
**Depends on:** Increment 1
