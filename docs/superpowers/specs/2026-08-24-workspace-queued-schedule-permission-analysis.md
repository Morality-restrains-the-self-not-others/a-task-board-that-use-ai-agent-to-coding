# 工作空间级排队调度 — 权限分析

- 日期：2026-08-24
- 范围：新增 GET/PUT `/api/tenant/{tid}/workspace/{wid}/queue-schedule/`（taskTaskService）

## 端点鉴权矩阵

| 端点 | 认证 | 授权 | 依据 |
|---|---|---|---|
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | Cookie 会话（auth 中间件解析 userID） | `requireTenantMember`（main.go `/api/tenant/` 全局门禁）+ `hasWorkspaceAccess(tenantID, wid, userID)`，`internal` 绕过 | 对齐 handleGetQueuedAutoRun（queued_schedule_handlers.go） |
| PUT 同上 | 同上 | 同上 | 同上 |
| 内部 POST `/api/internal/tasks/queued-schedule/dispatch-once/` | 不变（requireInternalSecret） | 不变 | 既有 |

## 前端权限

- Sidebar 导航项「工作空间的排队调度」：无独立权限码（导航本身无数据），页面数据面由后端端点鉴权兜底；`showMenu` 沿用主导航可见性（permsReady 兜底三主项 + `canSeeMenuKey`）。为避免导航出现在无任何工作空间访问权的用户面前，沿用「工作面板」同款可见性（nav.work_panel 码），数据加载 403 时页面显示无权提示（复用 showRequestError / TenantPageAccessEmpty 模式）。
- 新页面依赖的成员列表仅含任务 title/状态，无敏感字段；不暴露其他租户数据（端点按 tid+wid 双约束）。

## 审计结论
无新角色/新权限码；沿用租户成员 + 工作空间访问双层门禁。✅
