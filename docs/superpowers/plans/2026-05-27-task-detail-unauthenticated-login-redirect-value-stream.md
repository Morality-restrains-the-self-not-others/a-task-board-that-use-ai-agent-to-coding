# Value Stream: 任务详情未登录跳转登录页

> Derived from design: `docs/superpowers/specs/2026-05-27-task-detail-unauthenticated-login-redirect-design.md`

## Value Summary

未登录或会话失效的用户访问带 `relayToTrae=true` 的任务详情页时，被可靠引导至登录页，登录后回到原 URL 继续使用 relay 直启。

## End-to-End Flow

[用户打开 task-detail URL] → [路由守卫校验 profile 会话] → [无效则写 postLoginRedirect + 跳转 /auth/login/] → [登录成功] → [回到 task-detail?relayToTrae=true]

## Value Increments

### Increment 1: 可靠认证守卫（Thin Slice）
**Value to user:** 无效/缺失会话时必定进入登录页，不再卡在空白任务详情  
**Scope:** 修复 `checkUserAuthenticated`；清除 stale `userId`；Playwright 回归  
**Depends on:** nothing

### Increment 2: 登录后回跳原任务详情
**Value to user:** 登录后直接回到 relay 任务详情，无需手动找链接  
**Scope:** `postLoginRedirect` 写入 `to.fullPath`；安全相对路径校验  
**Depends on:** Increment 1
