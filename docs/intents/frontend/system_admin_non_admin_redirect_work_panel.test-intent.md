# 测试意图：非系统管理员访问系统管理页跳转工作面板

## 对应功能意图

`docs/intents/frontend/system_admin_non_admin_redirect_work_panel.intent.md`

## 测试目标

验证 `SystemAdminRouteGuardService` 对超管放行、非超管跳转工作面板、无租户与会话失败时的 fail-closed 行为。

## 测试分层

| 层级 | 覆盖 |
|------|------|
| Vitest（领域服务） | `resolveAccess` 决策矩阵 |
| Playwright E2E（公网 CDP） | 非超管跳转 work-panel（T7）+ 超管停留 system-admin（T8） |

## 用例矩阵

| ID | 场景 | 期望 | 层级 |
|----|------|------|------|
| T1 | `/me/` 返回 `is_superuser=true` | `{ allowed: true }` | Vitest |
| T2 | `/me/` 非超管且有 `current_company.id` | `{ allowed: false, redirectPath: /tenant/{id}/work-panel/ }` | Vitest |
| T3 | `/me/` 非超管无 current_company、有 companies[0] | 使用 companies[0].id 拼 work-panel | Vitest |
| T4 | `/me/` 非超管且无任何公司 | `redirectPath: '/'` | Vitest |
| T5 | 无法加载 `/me/`（会话不足） | `{ allowed: false, redirectPath: '/' }` fail-closed | Vitest |
| T6 | `is_superuser` 字符串 `'True'` | 视为超管放行 | Vitest |
| T7 | 非超管账号登录后打开 `/system-admin/` | URL 变为 `/tenant/{其公司Id}/work-panel/`，且不再停留在 system-admin | Playwright |
| T8 | 超管 `author@example.com` 登录后打开 `/system-admin/` | URL 仍包含 `/system-admin/`，不跳转 work-panel | Playwright |

## 数据与环境

- Vitest：Mock `apiFetch` + `getCookie('userId')`；租户示例 ID 仅作断言夹具
- Playwright：稳定非超管账号 `e2e.nonadmin.sysadmin.redirect@ljytest.com`（`ensureE2eNonAdminAccount.mjs` 幂等注册/激活；注册密码须经前端 `PasswordHasher` 再提交）；超管对照 `author@example.com`；CDP 9223/9222；站点默认 `https://www.daydaymoney.com`；账号清单见 `task2app/测试.ai.md`

## 通过标准

- T1–T6 全部通过
- T7：`SystemAdmin.non-admin-redirect-work-panel.playwright.test.sh` 通过
- T8：`SystemAdmin.superadmin-stays-on-system-admin.playwright.test.sh` 通过

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 |
| 2026-07-15 | 增补 T7 Playwright E2E + 自建非超管账号 |
| 2026-07-15 | 增补 T8 超管停留对照 E2E；账号写入团队清单 |
