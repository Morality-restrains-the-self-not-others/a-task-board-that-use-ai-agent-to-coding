# 角色权限分析：导航栏任务搜索过滤

- **日期**: 2026-07-20
- **设计**: `2026-07-20-navbar-task-search-filter-design.md`
- **迭代**: navbar-task-search-filter

## 结论

无新权限点；`tasks/search` 以 project-service `workspaces?mine=1` 为唯一可搜索工作空间集合（fail-closed）。

## 强制约束（2026-07-20 加固）

1. 未带 `X-Auth-User-Id` → 401
2. 可搜索范围 = `listAccessibleWorkspaceIDs`（mine=1）；**禁止**在查询失败时枚举租户下全部任务工作空间
3. 带 `workspace_id` 时：不在 mine 集合 → 403（工作空间存在）或 404（不存在）
4. Navbar 客户端不得绕过：即使前端误传无权 workspace，服务端仍拒绝

## 角色矩阵

| 角色 | 搜索输入框 | 调用 search | 打开工作面板 | 打开任务 |
|------|-----------|-------------|-------------|---------|
| 未登录 | 隐藏 | — | — | — |
| 已登录租户成员 | 可见 | ✅ 仅可访问 workspace | ✅ 若有权 | ✅ 若有权 |
| 系统管理员（超管导航） | 隐藏（走系统管理） | — | — | — |
| 无租户上下文 | 隐藏/禁用 | — | — | — |

## 接口权限

| 接口 | 变更 | 鉴权 |
|------|------|------|
| `GET /api/tenant/{tid}/tasks/search/` | 扩展匹配字段与可选 `assignee_ids` | 既有：`X-Auth-*` + workspace access |
| `GET .../company_members/` | 无变更（前端已用） | 既有成员列表门禁 |

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| `assignee_ids` 被滥用于枚举 | 结果仍受 workspace ACL 过滤；limit≤100 |
| 深链 `task_id` 越权打开 | WorkPanel 仅打开本地已加载/有权 todos 中匹配项；否则跳转 task-detail 由服务端 403 |
