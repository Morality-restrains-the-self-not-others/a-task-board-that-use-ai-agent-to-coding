# 功能意图：非系统管理员访问系统管理页跳转工作面板

## 背景与目标

已登录但非系统管理员（`is_superuser !== true`）的用户若直接打开 `/system-admin/`（及子路由），当前路由守卫对 `requiresAdmin` 为空实现，导致页面可进入。应拒绝访问并自动跳转到该用户当前租户的工作面板 `/tenant/{tenantId}/work-panel/`。

## 范围与边界

- **范围内**：前端 `router.js` 的 `requiresAdmin` 守卫；解析 `/me/`（或等价会话）中的 `is_superuser` 与租户 ID（`current_company.id` / `companies[0].id`）
- **范围外**：后端 `/api/system-admin/*` 的 403 中间件（已有，不改）；系统管理员正常访问路径
- **角色**：系统管理员可进；租户用户/普通用户跳转工作面板

## 约束与风险

- 守卫须 fail-closed：无法确认 `is_superuser` 时不得放行系统管理页
- 无可用租户时回退首页 `/`，不得构造非法 `/tenant//work-panel/`
- 租户 ID 必须来自当前会话用户上下文，禁止写死示例 ID

## 验收标准

1. 已登录非系统管理员访问 `/system-admin/` → 跳转到 `/tenant/{其租户Id}/work-panel/`
2. 已登录非系统管理员访问任意 `/system-admin/*` 子路由 → 同样跳转工作面板
3. 系统管理员（`is_superuser=true`）访问 `/system-admin/` → 可正常进入，不跳转
4. 未登录访问 → 仍走既有登录重定向（`/auth/login/`），不误进系统管理
5. 无租户信息的已登录非管理员 → 跳转 `/`（fail-safe）

## 实施计划

1. 抽取 `SystemAdminRouteGuardService`（可单测）
2. 在 `router.beforeEach` 的 `requiresAdmin` 分支调用并按结果 `next`
3. Vitest 覆盖超管放行 / 非超管跳转 / 无租户回退 / me 失败 fail-closed

## 业务意图 → 事件对照

**无对应事件**：纯前端路由守卫，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 非系统管理员访问系统管理页跳转工作面板 | — | — | — | 纯前端路由守卫，无服务端业务状态变更 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：修复 requiresAdmin 空实现，非超管跳转 work-panel |
| 2026-07-15 | 补 Playwright E2E（自建非超管 `e2e.nonadmin.sysadmin.redirect@ljytest.com`） |
| 2026-07-15 | 账号写入 `测试.ai.md` 清单；超管对照 E2E（`author@example.com` 停留 system-admin） |
